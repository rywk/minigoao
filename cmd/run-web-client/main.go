package main

import client "github.com/rywk/minigoao/pkg/client"

func main() {
	if err := client.Run(true); err != nil {
		panic(err)
	}
}
