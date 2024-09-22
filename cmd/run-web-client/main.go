package main

import (
	_ "embed"
	"os"

	"github.com/rywk/minigoao/pkg/client"
)

//go:embed config.txt
var config []byte

func main() {
	defaultServer := string(config)
	if len(os.Args) == 2 {
		defaultServer = os.Args[1]
	}
	if err := client.Run(true, defaultServer); err != nil {
		panic(err)
	}
}
