package client

import (
	_ "embed"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game"
	"github.com/rywk/minigoao/pkg/client/game/texture"
)

var (
	//go:embed icon.png
	Icon_png []byte
)

func Run(web bool, serverAddr string) error {
	icon := texture.Decode(Icon_png)
	g := game.NewGame(web, serverAddr)
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowIcon([]image.Image{icon})
	return ebiten.RunGame(g)
}
