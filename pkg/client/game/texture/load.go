package texture

import (
	"bytes"
	"image"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/texture/assets"
	"github.com/rywk/minigoao/pkg/direction"
	asset "github.com/rywk/minigoao/proto/message/assets"
)

const GrassTextureSize = 128

var (
	assetConfig = map[asset.Asset]struct {
		c   SpriteConfig
		img []byte
	}{
		asset.Grass: {
			c: SpriteConfig{
				Width:  GrassTextureSize,
				Height: GrassTextureSize,
			},
			img: assets.GrassPatches_png,
		},
		asset.Tiletest: {
			c: SpriteConfig{
				Width:  32,
				Height: 32,
			},
			img: assets.Tiletest_png,
		},
		asset.DarkArmour: {
			c: SpriteConfig{
				Width:  25,
				Height: 45,
				DirectionLength: map[direction.D]int{
					direction.Front: 6,
					direction.Back:  6,
					direction.Left:  5,
					direction.Right: 5,
				},
				RealDirMap: map[direction.D]direction.D{
					direction.Front: direction.Right,
					direction.Back:  direction.Right,
					direction.Left:  direction.Right,
					direction.Right: direction.Right,
				},
			},
			img: assets.Body_png,
		},
		asset.WarAxe: {
			c: SpriteConfig{
				Width:  25,
				Height: 45,
				DirectionLength: map[direction.D]int{
					direction.Front: 6,
					direction.Back:  6,
					direction.Left:  5,
					direction.Right: 5,
				},
				RealDirMap: map[direction.D]direction.D{
					direction.Front: direction.Right,
					direction.Back:  direction.Right,
					direction.Left:  direction.Right,
					direction.Right: direction.Right,
				},
			},
			img: assets.Axe_png,
		},
		asset.Head: {
			c: SpriteConfig{
				Width:  17,
				Height: 50,
				RealDirMap: map[direction.D]direction.D{
					direction.Front: direction.Right,
					direction.Back:  direction.Right,
					direction.Left:  direction.Right,
					direction.Right: direction.Right,
				},
			},
			img: assets.Head_png,
		},
		asset.Shroom: {
			c: SpriteConfig{
				Width:  29,
				Height: 29,
			},
			img: assets.Ongo_png,
		},
	}
)

var loaded = &sync.Map{}

func LoadAnimation(a asset.Asset) A {
	v, ok := loaded.Load(a)
	if ok {
		return NewBodyAnimation(v.(*ebiten.Image), assetConfig[a].c)
	}
	cfg := assetConfig[a]
	img, _, err := image.Decode(bytes.NewReader(cfg.img))
	if err != nil {
		log.Fatal(err)
	}
	ei := ebiten.NewImageFromImage(img)
	loaded.Store(a, ei)
	return NewBodyAnimation(ei, cfg.c)
}

func LoadStill(a asset.Asset) A {
	v, ok := loaded.Load(a)
	if ok {
		return NewHeadStill(v.(*ebiten.Image), assetConfig[a].c)
	}
	cfg := assetConfig[a]
	img, _, err := image.Decode(bytes.NewReader(cfg.img))
	if err != nil {
		log.Fatal(err)
	}
	ei := ebiten.NewImageFromImage(img)
	loaded.Store(a, ei)
	return NewHeadStill(ei, cfg.c)
}

func LoadTexture(a asset.Asset) T {
	if a == asset.Nothing {
		return &EmptyTexture{}
	}
	v, ok := loaded.Load(a)
	if ok {
		return NewTexture(v.(*ebiten.Image), assetConfig[a].c)
	}
	cfg := assetConfig[a]
	img, _, err := image.Decode(bytes.NewReader(cfg.img))
	if err != nil {
		log.Fatal(err)
	}
	ei := ebiten.NewImageFromImage(img)
	loaded.Store(a, ei)
	return NewTexture(ei, cfg.c)
}
