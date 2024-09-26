package msgs

import (
	"net"

	owss "github.com/tarndt/wasmws"
)

func DialServer(address string, web, secure bool) (Msgs, error) {
	return DialWS3(address, secure)
}
func DialWS3(address string, secure bool) (Msgs, error) {
	pref := "ws://"
	if secure {
		pref = "wss://"
	}
	address = pref + address + "/upgrader"
	c, err := owss.Dial("websocket", address)
	if err != nil {
		return nil, err
	}
	return &WSM3{
		c: c,
	}, nil
}

type WSM3 struct {
	c net.Conn
}

func (ws *WSM3) IP() string {
	return ws.c.RemoteAddr().String()
}

func (ws *WSM3) Close() {
	ws.c.Close()
}

func (ws *WSM3) Read() (*IncomingData, error) {
	return readMsg(ws.c)
}
func (ws *WSM3) Write(event E, data []byte) error {
	return write(ws.c, event, data)

}

func (ws *WSM3) WriteWithLen(event E, data []byte) error {
	return writeWithLen(ws.c, event, data)

}

func (ws *WSM3) EncodeAndWrite(e E, msg interface{}) error {
	return encodeAndWrite(ws, e, msg)
}
