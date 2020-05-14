# Run command in a queue file
```sh
./tasks-queue-simple /some/absolute/path/to/queue/file.json
```

## Contents of the json queue file
```json
[
  {
    "user": "userid string here",
    "cmd": "#!/bin/bash\necho \"Hello world!\""
  }
  ...
]
```

## Build the program
```sh
go build
```

## Test
Change the abstract paths in [test/queue.json](test/queue.json) to your paths
```sh
go build
cd test
go build
cd ..
./tasks-queue-simple `pwd`/test/queue.json
``
