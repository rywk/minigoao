package messenger

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"

	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/events"
	"google.golang.org/protobuf/proto"
)

var (
	prefixSize = 4
)

type M struct {
	tcp       net.Conn
	tcpReader io.Reader
}

func New(tcp net.Conn, udp net.PacketConn, addr net.Addr) *M {
	return &M{
		tcp:       tcp,
		tcpReader: bufio.NewReader(tcp),
	}
}

func (m *M) Read() (*message.Event, error) {
	prefixHolder := make([]byte, prefixSize)
	_, err := m.tcpReader.Read(prefixHolder)
	if err != nil {
		return nil, err
	}
	msgSize := binary.BigEndian.Uint32(prefixHolder)
	eb := make([]byte, msgSize)
	_, err = m.tcpReader.Read(eb)
	if err != nil {
		return nil, err
	}
	e := message.Event{}
	return &e, proto.Unmarshal(eb, &e)
}

func (m *M) Write(e *message.Event) error {
	bs := events.Bytes(e)
	length := make([]byte, prefixSize)
	//log.Println(len(bs))
	binary.BigEndian.PutUint32(length, uint32(len(bs)))
	_, err := m.tcp.Write(append(length, bs...))
	if err != nil {
		return err
	}
	return nil
}
