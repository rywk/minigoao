package main

import (
	"os"
	"strconv"

	"github.com/rywk/minigoao/pkg/server"
)

func main() {
	if len(os.Args) != 3 {
		panic("must provide a port and exposure")
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	exposed, err := strconv.ParseBool(os.Args[2])
	if err != nil {
		panic(err)
	}
	if err := server.NewServer(port).Start(exposed); err != nil {
		panic(err)
	}
}
