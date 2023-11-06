package net

import (
	"errors"
	"log"
	"net"
	"sync/atomic"

	"github.com/rywk/minigoao/pkg/messenger"
	"github.com/rywk/minigoao/proto/message"
	"google.golang.org/protobuf/proto"
)

const MaxConnCount = 50

type Listener struct {
	tcp net.Listener

	register   chan net.Conn
	unregister chan *Conn

	online    map[int]bool
	connCount *atomic.Uint32
	conns     [MaxConnCount]*Conn

	newPlayer chan *Conn

	shutdown chan struct{}
}

func NewListener(port string) *Listener {
	tcp, err := net.Listen("tcp4", port)
	if err != nil {
		panic(err)
	}

	cm := &atomic.Uint32{}
	cm.Store(0)

	l := &Listener{
		tcp:        tcp,
		connCount:  cm,
		conns:      [MaxConnCount]*Conn{},
		register:   make(chan net.Conn),
		unregister: make(chan *Conn),
		newPlayer:  make(chan *Conn),
	}

	go l.listen()
	go l.handleConnections()

	return l
}

func (l *Listener) listen() {
	for {
		c, err := l.tcp.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		l.register <- c
	}
}

func (l *Listener) handleConnections() {
	for {
		select {
		case c, ok := <-l.register:
			if !ok {
				return
			}
			id := l.connCount.Add(1) - 1
			conn := NewConn(l, id, c).Start()
			l.conns[id] = conn
			l.newPlayer <- conn
			log.Println("Client connected")
		case c, ok := <-l.unregister:
			if !ok {
				return
			}
			// "we" need to work on an index reallocation thingy
			// should pretty simple tbh, just when is >len() start to search from 0 (first logged in already left 🙄)
			l.conns[c.ID] = nil
		}
	}
}

func (l *Listener) NewPlayerConnected() chan *Conn {
	return l.newPlayer
}

func (l *Listener) Range(fn func(c *Conn)) {
	for i := 0; i < MaxConnCount; i++ {
		if l.conns[i] == nil {
			continue
		}
		fn(l.conns[i])
	}
}

func (l *Listener) RangeIds(fn func(c *Conn), ids ...int) {
	for _, id := range ids {
		if l.conns[id] == nil {
			continue
		}
		fn(l.conns[id])
	}
}

func (l *Listener) Stop() {
	<-l.shutdown
}

func (l *Listener) Conn(id int) *Conn {
	return l.conns[id]
}

type Conn struct {
	ID   uint32
	L    *Listener
	Addr net.Addr
	m    *messenger.M
	tcp  net.Conn
	i, o chan *message.Event
}

func NewConn(l *Listener, id uint32, c net.Conn) *Conn {
	return &Conn{
		ID:   id,
		m:    messenger.New(c, nil, nil),
		tcp:  c,
		Addr: c.RemoteAddr(),
		i:    make(chan *message.Event, 10),
		o:    make(chan *message.Event, 10),
	}
}

func (c *Conn) Send() chan<- *message.Event {
	return c.o
}

func (c *Conn) Recive() <-chan *message.Event {
	return c.i
}

func (c *Conn) Start() *Conn {
	go c.startToListen()
	go c.startToSend()
	return c
}

func (c *Conn) startToListen() {
	for {
		msg, err := c.m.Read()
		if err != nil {
			if errors.Is(err, proto.Error) {
				log.Println("PROTO ERROR!!! READING CLIENT", err)
				continue
			}
			log.Println("ERROR!!! READING CLIENT", err)
			break
		}
		c.i <- msg
	}
	log.Println(c.Addr.String(), "listener exited")
	close(c.i)
}

func (c *Conn) startToSend() {
	for msg := range c.o {
		err := c.m.Write(msg)
		if err != nil {
			log.Println("ERROR!!! WRITING TO CLIENT", err)
			break
		}
	}
	log.Println(c.Addr.String(), "writer exited")
}
