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
