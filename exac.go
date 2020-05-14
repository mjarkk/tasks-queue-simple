package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"
)

type printer struct {
	num  int
	buff bytes.Buffer
	lock sync.Mutex
}

func (s *printer) Write(p []byte) (n int, err error) {
	parts := strings.Split(string(p), "\n")
	for i, part := range parts {
		if len(part) > 0 {
			part = "#" + strconv.Itoa(s.num+1) + " " + part
		}
		parts[i] = part
	}

	fmt.Print(strings.Join(parts, "\n"))

	s.lock.Lock()
	n, err = s.buff.Write(p)
	s.lock.Unlock()
	return
}

func (s *printer) String() (out string) {
	s.lock.Lock()
	out = s.buff.String()
	s.lock.Unlock()
	return
}

func (c *Command) saveToFile() (savedToFile string, err error) {
	if !strings.HasPrefix(c.CMD, "#!") {
		c.CMD = "#!/bin/bash" + c.CMD
	}

	f, err := ioutil.TempFile("", "queue-item")
	if err != nil {
		return
	}
	defer f.Close()
	f.Chmod(0777)
	f.Write([]byte(c.CMD))
	filename := f.Name()

	return filename, nil
}

// Exec executes a program defined in the config
func (c *Command) Exec() (signalled bool, err error) {
	execFileName, err := c.saveToFile()
	if err != nil {
		fmt.Println("Can't parse start argument, err:", err)
	}
	defer os.Remove(execFileName)

	cmd := exec.Command(execFileName)
	cmd.Env = append(os.Environ())

	output := &printer{num: c.OrderNumber}
	cmd.Stderr = output
	cmd.Stdout = output

	err = cmd.Start()
	if err != nil {
		return false, err
	}

	dune := make(chan error)
	go func() { dune <- cmd.Wait() }()

	select {
	case <-stopSignal:
		cmd.Process.Signal(syscall.SIGSTOP)
		time.Sleep(time.Second * 1)
		cmd.Process.Signal(syscall.SIGKILL)
		defer func() { stopSignalDune <- struct{}{} }()
		return true, errors.New(output.String())
	case err := <-dune:
		err = errors.New(output.String())
		return false, err
	}
}

// scanWordsWithNewLines is a copy of bufio.ScanWords but this also captures new lines
// For specific comments about this function take a look at: bufio.ScanWords
func scanWordsWithNewLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
	}
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if isSpace(r) {
			return i + width, data[start:i], nil
		}
	}
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	return start, nil, nil
}

// isSpace is also copied from the bufio package and has been modified to also captures new lines
// For specific comments about this function take a look at: bufio.isSpace
func isSpace(r rune) bool {
	if r <= '\u00FF' {
		switch r {
		case ' ', '\t', '\v', '\f':
			return true
		case '\u0085', '\u00A0':
			return true
		}
		return false
	}
	if '\u2000' <= r && r <= '\u200a' {
		return true
	}
	switch r {
	case '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000':
		return true
	}
	return false
}
