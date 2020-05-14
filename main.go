package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strings"
)

type Command struct {
	User        string `json:"user"`
	CMD         string `json:"cmd"`
	OrderNumber int
}

var stopSignal = make(chan os.Signal, 1)
var stopSignalDune = make(chan struct{})

func (c *Command) Print(s ...interface{}) {
	fmt.Printf("#%d %s", c.OrderNumber+1, fmt.Sprintln(s...))
}

func main() {
	args := os.Args
	if len(args) == 1 {
		fmt.Println("No queue json file provided")
		os.Exit(1)
	}
	if len(args) > 2 {
		fmt.Println("To meany arguments")
		os.Exit(1)
	}

	fileName := args[1]
	path.IsAbs(fileName)

	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println("Unable to read queue file, error:", err)
		os.Exit(1)
	}

	queue := []Command{}
	err = json.Unmarshal(fileBytes, &queue)
	if err != nil {
		fmt.Println("Unable to read queue file, error:", err)
		os.Exit(1)
	}

	exit := make(chan error, 1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)

	go func() {
		stopSignal <- <-c
		<-stopSignalDune
		exit <- errors.New("OS stop signal")
	}()

	go func() {
		signalled := false
		for i, command := range queue {
			command.OrderNumber = i

			cmdLines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(command.CMD, "\r", "\n"), "\n\n", "\n"), "\n")
			if len(cmdLines) > 2 {
				cmdLines = cmdLines[:2]
			}
			command.Print("Running next command:", strings.Join(cmdLines, "\\n"))

			signalled, err = command.Exec()
			if signalled {
				break
			}
			if err != nil {
				command.Print("Command failed with output:", err)
			}
		}
		if !signalled {
			fmt.Println("All commands executed, exiting")
			exit <- nil
		}
	}()

	err = <-exit
	signal.Stop(c)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
