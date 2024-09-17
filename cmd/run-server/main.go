package main

import (
	"os"
	"strconv"

	"github.com/rywk/minigoao/pkg/server"
)

func main() {
	if len(os.Args) != 2 {
		panic("must provide a port")
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := server.NewServer(port).Start(); err != nil {
		panic(err)
	}
}
