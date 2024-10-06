package msgs

import (
	"context"
	"crypto/tls"
	"net"

	wswasm "github.com/coder/websocket"
	"github.com/gorilla/websocket"
)

func DialServer(address string, web, secure bool) (Msgs, error) {
	if web {
		return DialWS2(address, secure)
	}
	return DialTCP(address)
}

func DialTCP(address string) (Msgs, error) {
	c, err := net.Dial("tcp4", address)
	if err != nil {
		return nil, err
	}
	return New(c), nil
}

func SkipVerification() (*tls.Config, error) {
	return &tls.Config{InsecureSkipVerify: true}, nil
}

func DialWS2(address string, secure bool) (Msgs, error) {

	pref := "ws://"
	if secure {
		pref = "wss://"
	}
	address = pref + address + "/upgrader"
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
	ws.c.CloseNow()
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
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &WSM{c}, nil

}
