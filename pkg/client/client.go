package client

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game"
)

func Run() error {
	g := game.NewGame()
	g.Ready()
	log.Println("Running game..")
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	return ebiten.RunGame(g)
}
