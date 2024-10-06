package msgs

import (
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

type MMsgs interface {
	Address() string
	NewConn() (Msgs, error)
}

type TCPServer struct {
	tcp net.Listener
}

func ListenTCP(address string) (MMsgs, error) {
	tcp, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &TCPServer{tcp}, nil
}
func (s *TCPServer) Address() string {
	return s.tcp.Addr().String()
}
func (s *TCPServer) NewConn() (Msgs, error) {
	c, err := s.tcp.Accept()
	if err != nil {
		return nil, err
	}
	return New(c), nil
}

type WSServer struct {
	addr     string
	newConns chan Msgs
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewUpgraderMiddleware() (MMsgs, http.HandlerFunc) {
	wss := &WSServer{}
	wss.newConns = make(chan Msgs, 100)
	return wss, func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Printf("upgrade: %v %v", r.RemoteAddr, err)
			return
		}
		wss.newConns <- &WSM{c}
		log.Printf("upgraded %v", r.RemoteAddr)
	}
}

type WSM struct {
	c *websocket.Conn
}

func (ws *WSM) IP() string {
	return ws.c.RemoteAddr().String()
}

func (ws *WSM) Close() {
	ws.c.Close()
}

func (ws *WSM) Read() (*IncomingData, error) {
	_, r, err := ws.c.NextReader()
	if err != nil {
		return nil, err
	}
	return readMsg(r)
}
func (ws *WSM) Write(event E, data []byte) error {
	w, err := ws.c.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	return write(w, event, data)
}

func (ws *WSM) WriteWithLen(event E, data []byte) error {
	w, err := ws.c.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	return writeWithLen(w, event, data)
}

func (ws *WSM) EncodeAndWrite(e E, msg interface{}) error {
	return encodeAndWrite(ws, e, msg)
}

func (s *WSServer) Address() string {
	return s.addr
}

func (s *WSServer) NewConn() (Msgs, error) {
	return <-s.newConns, nil
}
