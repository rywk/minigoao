package main

import client "github.com/rywk/minigoao/pkg/client"

func main() {
	if err := client.Run(false); err != nil {
		panic(err)
	}
}
