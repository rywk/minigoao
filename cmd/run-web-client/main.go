package main

import (
	"os"

	client "github.com/rywk/minigoao/pkg/client"
)

func main() {
	defaultServer := ""
	if len(os.Args) == 2 {
		defaultServer = os.Args[1]
	}
	if err := client.Run(true, defaultServer); err != nil {
		panic(err)
	}
}
