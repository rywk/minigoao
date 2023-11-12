package texture

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/constants/direction"
)

type A interface {
	Next(direction.D) A
	Dir(direction.D)
	Stopped(direction.D) *ebiten.Image
	Frame() *ebiten.Image
}

// Like A but doesnt automatically refresh
type AF interface {
	Next(direction.D) *ebiten.Image
	Finished() bool
}

type Effect interface {
	Play() bool
	EffectFrame() *ebiten.Image
	EffectOpt(*ebiten.DrawImageOptions) *ebiten.DrawImageOptions
}

type (
	SpriteConfig struct {
		X, Y          int
		Width, Height int
		GridW, GridH  int
		FrameCount    int

		// frames for in game direction
		DirectionLength map[direction.D]int

		// direction ingame->direction in sprites
		RealDirMap map[direction.D]direction.D
	}

	// Animation
	Animation struct {
		// mapping game direction to frame reading direction
		directionMap map[direction.D]*Sprites

		sprites *Sprites

		currentDir direction.D
	}

	// Still is like an Animation but without frames
	Still struct {
		// mapping game direction to image
		directionMap map[direction.D]*ebiten.Image

		currentDir direction.D
	}

	Sprites struct {
		strip  []*ebiten.Image
		len, i int
	}
)

func NewHeadStill(img *ebiten.Image, config SpriteConfig) *Still {
	s := &Still{directionMap: make(map[uint32]*ebiten.Image), currentDir: direction.Front}
	x := config.X
	for _, d := range []direction.D{
		direction.Front,
		direction.Right,
		direction.Left,
		direction.Back,
	} {
		img := img.SubImage(image.Rect(x, config.Y, x+config.Width, config.Y+config.Height)).(*ebiten.Image)
		s.directionMap[d] = img
		x += config.Width
	}
	return s
}

func (s *Still) Dir(d direction.D) {
	if d != direction.Still {
		s.currentDir = d
	}
}

func (s *Still) Next(d direction.D) A {
	if d != direction.Still {
		s.currentDir = d
	}
	return s
}

func (s *Still) Frame() *ebiten.Image {
	return s.directionMap[s.currentDir]
}

func (s *Still) Stopped(d direction.D) *ebiten.Image {
	return s.directionMap[d]
}

func NewEffectAnimation(img *ebiten.Image, config SpriteConfig) *Animation {
	a := &Animation{currentDir: direction.Front}
	y := config.Y
	frames := config.DirectionLength[direction.Right]
	sprits := []*ebiten.Image{}
	if config.GridH == 0 {
		x := config.X
		for i := 0; i < frames; i++ {
			sprits = append(sprits, img.SubImage(image.Rect(x, y, x+config.Width, y+config.Height)).(*ebiten.Image))
			x += config.Width
		}
		a.sprites = NewSprites(sprits)
	} else {
		x := 0
		y := 0
		for j := 0; j < config.GridH; j++ {
			for i := 0; i < config.GridW; i++ {
				sprits = append(sprits, img.SubImage(image.Rect(x, y, x+config.Width, y+config.Height)).(*ebiten.Image))
				x += config.Width
			}
			x = 0
			y += config.Height
		}
		a.sprites = NewSprites(sprits)

	}
	return a
}

func NewEffectGridAnimation(img *ebiten.Image, config SpriteConfig) *Animation {
	a := &Animation{currentDir: direction.Front}
	y := config.Y
	frames := config.DirectionLength[direction.Right]
	sprits := []*ebiten.Image{}
	x := config.X
	for i := 0; i < frames; i++ {
		sprits = append(sprits, img.SubImage(image.Rect(x, y, x+config.Width, y+config.Height)).(*ebiten.Image))
		x += config.Width
	}
	a.sprites = NewSprites(sprits)
	return a
}

func NewBodyAnimation(img *ebiten.Image, config SpriteConfig) *Animation {
	a := &Animation{currentDir: direction.Front, directionMap: make(map[uint32]*Sprites)}
	y := config.Y
	for _, d := range direction.List {
		frames := config.DirectionLength[d]
		sprits := []*ebiten.Image{}
		x := config.X
		for i := 0; i < frames; i++ {
			sprits = append(sprits, img.SubImage(image.Rect(x, y, x+config.Width, y+config.Height)).(*ebiten.Image))
			x += config.Width
		}
		s := NewSprites(sprits)
		a.directionMap[d] = s
		y += config.Height
	}
	return a
}

func (a *Animation) EffectOpt(op *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {
	return op
}

func (a *Animation) Play() bool {
	return a.sprites.Next()
}

func (a *Animation) EffectFrame() *ebiten.Image {
	return a.sprites.Frame()
}

func (a *Animation) Dir(d direction.D) {
	if d != a.currentDir && d != direction.Still {
		a.currentDir = d
	}
}

func (a *Animation) Next(d direction.D) A {
	if d != a.currentDir && d != direction.Still {
		a.directionMap[d].Reset()
		a.currentDir = d
	}
	a.directionMap[d].Next()
	return a
}

func (a *Animation) Frame() *ebiten.Image {
	return a.directionMap[a.currentDir].Frame()
}

func (a *Animation) Stopped(d direction.D) *ebiten.Image {
	return a.directionMap[d].First()
}

func NewSprites(strip []*ebiten.Image) *Sprites {
	return &Sprites{
		strip: strip,
		len:   len(strip),
		i:     0,
	}
}

func (s *Sprites) Next() bool {
	if s.i+1 == s.len {
		s.i = 0
		return false
	} else {
		s.i++
		return true
	}
}

func (s *Sprites) Frame() *ebiten.Image {
	return s.strip[s.i]
}

func (s *Sprites) First() *ebiten.Image {
	s.Reset()
	return s.strip[s.i]
}

func (s *Sprites) Reset() { s.i = 0 }
