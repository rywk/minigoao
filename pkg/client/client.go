package client

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game"
)

func Run(web bool, serverAddr string) error {
	g := game.NewGame(web, serverAddr)
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetVsyncEnabled(false)
	return ebiten.RunGame(g)
}
