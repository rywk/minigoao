package server

import (
	"github.com/rywk/minigoao/pkg/server/game"
)

func Run() error {
	g := game.New()
	return g.Exit()
}
