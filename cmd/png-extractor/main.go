package main

import (
	"fmt"
	"io"
	"os"
)

var filename = "pkg/client/game/assets/img/axe.png"

func main() {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(bs); i++ {
		if len(bs) <= i+7 {
			break
		}
		if bs[i] == 137 &&
			bs[i+1] == 80 &&
			bs[i+2] == 78 &&
			bs[i+3] == 71 &&
			bs[i+4] == 13 &&
			bs[i+5] == 10 &&
			bs[i+6] == 26 &&
			bs[i+7] == 10 {
			fmt.Print("PNG SIGNATURE", i)
		}
	}
}
