package main

import "github.com/rywk/minigoao/pkg/client"

func main() {
	if err := client.Run(); err != nil {
		panic(err)
	}
}
