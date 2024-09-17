package main

import client "github.com/rywk/minigoao/pkg/client"

func main() {
	if err := client.Run(); err != nil {
		panic(err)
	}
}
