# Concurrent Websocket - Golang

A high performance websocket implementation that uses netpoll with read and
write concurrency to allow a high number of concurrent websocket connections.

## Prerequisite

Requires Go 1.11 or later, because of the usage of modules.

## Quick start

```sh
# assume the following codes in example.go file
$ cat example.go
```

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/blakerouse/concurrent-websocket"
)

func main() {
    r := gin.Default()

    concurrency := 128
	wh, _ := websocket.NewHandler(func(c *websocket.Channel, op websocket.OpCode, data []byte) {
        // echo
        c.Send(op, data)
    }, concurrency, concurrency)
	r.GET("/ws", wh.UpgradeHandler)

	r.Run() // listen and serve on 0.0.0.0:8080
}
```

```
# run example.go and visit 0.0.0.0:8080/ping on browser
$ go run example.go
```