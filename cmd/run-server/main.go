package main

import (
	"log"
	"os"
	"strconv"

	"github.com/rywk/minigoao/pkg/server"
)

func main() {
	if len(os.Args) != 4 {
		panic("must provide a port and exposure")
	}
	log.Printf("TCP port: %v", os.Args[1])
	log.Printf("WEB port: %v", os.Args[2])
	exposed, err := strconv.ParseBool(os.Args[3])
	if err != nil {
		panic(err)
	}
	if err := server.NewServer(os.Args[1], os.Args[2]).Start(exposed); err != nil {
		panic(err)
	}
}
