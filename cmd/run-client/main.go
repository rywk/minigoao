package main

import (
	_ "embed"

	"github.com/rywk/minigoao/pkg/client"
)

//go:embed config.txt
var config []byte

func main() {
	if err := client.Run(false, string(config)); err != nil {
		panic(err)
	}
}
