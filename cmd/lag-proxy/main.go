package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rywk/minigoao/pkg/msgs"
)

// go run ./cmd/lag-proxy/main.go 192.168.0.1:$1 127.0.0.1:$2 $3
func main() {
	if len(os.Args) != 4 {
		panic("usage: 127.0.0.1:5555 127.0.0.1:5555 10")
	}

	realAddr := os.Args[1]
	proxyAddr := os.Args[2]

	ms, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic("need a number of milliseconds to lag, got: " + os.Args[3])
	}

	lag := time.Millisecond * time.Duration(ms)
	server, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		panic(err)
	}
	log.Printf("Listening at %v\n", proxyAddr)
	for {
		conn, err := server.Accept()
		go func(conn net.Conn) {
			if err != nil {
				panic(err)
			}
			game, err := net.Dial("tcp", realAddr)
			if err != nil {
				panic(err)
			}
			client := msgs.New(conn)
			gameServer := msgs.New(game)

			wg := &sync.WaitGroup{}
			wg.Add(2)
			go proxy(lag, wg, client, gameServer, "CLIENT -> SERVER")
			go proxy(lag, wg, gameServer, client, "SERVER -> CLIENT")
			wg.Wait()
			gameServer.Close()
		}(conn)
	}
}

func proxy(lag time.Duration, wg *sync.WaitGroup, from, to *msgs.M, logs string) {
	defer func() {
		if r := recover(); r != nil {
			// if e, ok := r.(error); ok {
			// 	//fmt.Printf("Disconnected [%v]", strings.Join(strings.Split(e.Error(), ":")[:3], ":"))
			// }
		}
		wg.Done()
	}()
	for {
		im, err := from.Read()
		if err != nil {
			to.Close()
			panic(err)
		}
		time.Sleep(lag)
		//log.Printf("%v: %v", logs, im.Event.String())
		if im.Event.Len() == -1 {
			err = to.WriteWithLen(im.Event, im.Data)
			if err != nil {
				panic(err)
			}
			continue
		}
		err = to.Write(im.Event, im.Data)
		if err != nil {
			panic(err)
		}
	}
}
