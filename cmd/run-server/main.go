package main

import "github.com/rywk/minigoao/pkg/server"

func main() {
	if err := server.Run(); err != nil {
		panic(err)
	}
}
