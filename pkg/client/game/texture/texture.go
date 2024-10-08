package texture

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type EmptyTexture struct{}

func (t *EmptyTexture) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {}

type Texture struct {
	i *ebiten.Image
}

type T interface {
	Draw(*ebiten.Image, *ebiten.DrawImageOptions)
}

func (t *Texture) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	screen.DrawImage(t.i, options)
}

func NewTexture(img *ebiten.Image, config SpriteConfig) *Texture {
	t := &Texture{}

	t.i = img.SubImage(image.Rect(config.X, config.Y, config.X+config.Width, config.Y+config.Height)).(*ebiten.Image)
	return t
}

func NewFloorTexture(img *ebiten.Image, config SpriteConfig, x, y int) *Texture {
	t := &Texture{}
	sx, sy := config.X+config.Width*x, config.Y+config.Height*y
	ex, ey := config.X+config.Width+config.Width*x, config.Y+config.Height+config.Height*y
	t.i = img.SubImage(image.Rect(sx, sy, ex, ey)).(*ebiten.Image)
	return t
}
