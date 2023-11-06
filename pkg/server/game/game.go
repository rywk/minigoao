package game

import (
	"os"
	"os/signal"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/server/net"
	"github.com/rywk/minigoao/pkg/server/player"
)

type GameServer struct {
	l        *net.Listener
	exit     chan error
	shutdown chan struct{}
}

func New() *GameServer {
	// Signal to kill gracefully
	// TODO!!!
	c := make(chan os.Signal, 1)
	shutdown := make(chan struct{})
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			close(shutdown)
		}
	}()

	g := &GameServer{
		exit:     make(chan error),
		shutdown: shutdown,
	}
	g.init()
	return g
}

func (g *GameServer) init() {
	g.l = net.NewListener(constants.Port)
	go g.NewPlayerHandler()
}

func (gs *GameServer) NewPlayerHandler() {
	for c := range gs.l.NewPlayerConnected() {
		go player.RunPlayer(c)
	}
}

func (gs *GameServer) Exit() error {
	select {
	case <-gs.exit:
	case <-gs.shutdown:
	}
	return nil
}

func (g *GameServer) Events() {

}
