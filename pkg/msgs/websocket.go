package msgs

import (
	"context"
	"log"
	"net"
	"net/http"

	wswasm "github.com/coder/websocket"
	"github.com/gorilla/websocket"
)

type MMsgs interface {
	Address() string
	NewConn() (Msgs, error)
}

func DialServer(address string, web bool) (Msgs, error) {
	if web {
		return DialWS2(address)
	}
	return DialTCP(address)
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

func DialTCP(address string) (Msgs, error) {
	c, err := net.Dial("tcp4", address)
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

func ListenWS(address string) (MMsgs, error) {
	wss := &WSServer{addr: address}
	wss.newConns = make(chan Msgs, 100)
	http.HandleFunc("/upgrader", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		wss.newConns <- &WSM{c}

	})
	go func() {
		log.Fatal(http.ListenAndServe(address, nil))
	}()
	return wss, nil
}

func (s *WSServer) Address() string {
	return s.addr
}

func (s *WSServer) NewConn() (Msgs, error) {
	return <-s.newConns, nil
}

func DialWS2(address string) (Msgs, error) {
	address = "ws://" + address + "/upgrader"
	ctx := context.TODO()
	c, _, err := wswasm.Dial(ctx, address, nil)
	if err != nil {
		return nil, err
	}
	return &WSM2{
		c:    c,
		addr: address,
	}, nil
}

type WSM2 struct {
	addr string
	c    *wswasm.Conn
}

func (ws *WSM2) IP() string {
	return ws.addr
}

func (ws *WSM2) Close() {
	//ws.c.CloseNow()
}

func (ws *WSM2) Read() (*IncomingData, error) {
	_, r, err := ws.c.Reader(context.TODO())
	if err != nil {
		return nil, err
	}
	return readMsg(r)
}
func (ws *WSM2) Write(event E, data []byte) error {
	w, err := ws.c.Writer(context.TODO(), wswasm.MessageBinary)
	if err != nil {
		return err
	}
	defer w.Close()
	return write(w, event, data)
}

func (ws *WSM2) WriteWithLen(event E, data []byte) error {
	w, err := ws.c.Writer(context.TODO(), websocket.BinaryMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	return writeWithLen(w, event, data)
}

func (ws *WSM2) EncodeAndWrite(e E, msg interface{}) error {
	return encodeAndWrite(ws, e, msg)
}

func DialWS(address string) (Msgs, error) {
	url := "ws://" + address + "/upgrader"
	log.Print(url)
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &WSM{c}, nil

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
