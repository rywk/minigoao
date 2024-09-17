package texture

import (
	"bytes"
	"image"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	asset "github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
)

const GrassTextureSize = 128

var (
	assetConfig = map[asset.Image]struct {
		c   SpriteConfig
		img []byte
	}{
		asset.Grass: {
			c: SpriteConfig{
				Width:  GrassTextureSize,
				Height: GrassTextureSize,
			},
			img: img.GrassPatches_png,
		},
		asset.Tiletest: {
			c: SpriteConfig{
				Width:  32,
				Height: 32,
			},
			img: img.Tiletest_png,
		},
		asset.Shroom: {
			c: SpriteConfig{
				Width:  29,
				Height: 29,
			},
			img: img.Ongo_png,
		},
		asset.MeleeHit: {
			c: SpriteConfig{
				Width:  32,
				Height: 32,
				DirectionLength: map[direction.D]int{
					direction.Right: 5,
				},
			},
			img: img.MeleeHit_png,
		},
		asset.SpellApoca: {
			c: SpriteConfig{
				Width:      145,
				Height:     145,
				GridW:      4,
				GridH:      4,
				FrameCount: 16,
			},
			img: img.SpellApoca_png,
		},
		// asset.SpellApoca: {
		// 	c: SpriteConfig{
		// 		Width:  128,
		// 		Height: 100,
		// 		DirectionLength: map[direction.D]int{
		// 			direction.Right: 11,
		// 		},
		// 	},
		// 	img: img.SpellLastTrial_png,
		// },
		asset.SpellDesca: {
			c: SpriteConfig{
				Width:      127,
				Height:     127,
				GridW:      5,
				GridH:      3,
				FrameCount: 15,
			},
			img: img.SpellDesca_png,
		},
		// asset.SpellInmo: {
		// 	c: SpriteConfig{
		// 		Width:  128,
		// 		Height: 128,
		// 		DirectionLength: map[direction.D]int{
		// 			direction.Right: 15,
		// 		},
		// 	},
		// 	img: img.SpellInmo_png,
		// },
		asset.SpellInmo: {
			c: SpriteConfig{
				Width:  96,
				Height: 132,
				DirectionLength: map[direction.D]int{
					direction.Right: 10,
				},
			},
			img: img.SpellParalize_png,
		},
		asset.SpellInmoRm: {
			c: SpriteConfig{
				Width:      68,
				Height:     68,
				GridW:      5,
				GridH:      4,
				FrameCount: 20,
			},
			img: img.SpellInmoRm_png,
		},
		// asset.SpellHealWounds: {
		// 	c: SpriteConfig{
		// 		Width:      68,
		// 		Height:     68,
		// 		GridW:      5,
		// 		GridH:      4,
		// 		FrameCount: 20,
		// 	},
		// 	img: img.SpellHealWounds_png,
		// },
		asset.SpellHealWounds: {
			c: SpriteConfig{
				Width:  100,
				Height: 100,
				DirectionLength: map[direction.D]int{
					direction.Right: 10,
				},
			},
			img: img.SpellHealWounds2_png,
		},
		asset.SpellRevive: {
			c: SpriteConfig{
				Width:      76,
				Height:     76,
				GridW:      5,
				GridH:      6,
				FrameCount: 30,
			},
			img: img.SpellRevive_png,
		},
		asset.NakedBody: {
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
			img: img.BodyNaked_png,
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
			img: img.Head_png,
		},
		asset.DeadBody: {
			c: SpriteConfig{
				Width:  25,
				Height: 29,
				DirectionLength: map[direction.D]int{
					direction.Front: 3,
					direction.Back:  3,
					direction.Left:  3,
					direction.Right: 3,
				},
				RealDirMap: map[direction.D]direction.D{
					direction.Front: direction.Right,
					direction.Back:  direction.Right,
					direction.Left:  direction.Right,
					direction.Right: direction.Right,
				},
			},
			img: img.DeadBody_png,
		},
		asset.DeadHead: {
			c: SpriteConfig{
				Width:  16,
				Height: 16,
				RealDirMap: map[direction.D]direction.D{
					direction.Front: direction.Right,
					direction.Back:  direction.Right,
					direction.Left:  direction.Right,
					direction.Right: direction.Right,
				},
			},
			img: img.DeadHead_png,
		},
		asset.ProHat: {
			c: SpriteConfig{
				Width:  25,
				Height: 32,
				RealDirMap: map[direction.D]direction.D{
					direction.Front: direction.Right,
					direction.Back:  direction.Right,
					direction.Left:  direction.Right,
					direction.Right: direction.Right,
				},
			},
			img: img.HatPro_png,
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
			img: img.DarkArmor_png,
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
			img: img.Axe_png,
		},
		asset.SpecialSword: {
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
			img: img.SwordSpecial_png,
		},
		asset.TowerShield: {
			c: SpriteConfig{
				Width:  42,
				Height: 64,
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
			img: img.ShieldTower_png,
		},
		asset.SilverShield: {
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
			img: img.ShieldSilver_png,
		},
	}
)

var loaded = &sync.Map{}

func LoadAnimation(a asset.Image) A {
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

func LoadStill(a asset.Image) A {
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

func LoadTexture(a asset.Image) T {
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

func LoadEffect(a asset.Image) Effect {
	v, ok := loaded.Load(a)
	if ok {
		return NewEffectAnimation(v.(*ebiten.Image), assetConfig[a].c)
	}
	cfg := assetConfig[a]
	img, _, err := image.Decode(bytes.NewReader(cfg.img))
	if err != nil {
		log.Fatal(err)
	}
	ei := ebiten.NewImageFromImage(img)
	loaded.Store(a, ei)
	return NewEffectAnimation(ei, cfg.c)
}

func Decode(bs []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(bs))
	if err != nil {
		log.Fatal(err)
	}
	return ebiten.NewImageFromImage(img)
}

func AssetFromSpell(s spell.Spell) asset.Image {
	switch s {
	case spell.Explode:
		return asset.SpellApoca
	case spell.Paralize:
		return asset.SpellInmo
	case spell.RemoveParalize:
		return asset.SpellInmoRm
	case spell.ElectricDischarge:
		return asset.SpellDesca
	case spell.Revive:
		return asset.SpellRevive
	case spell.HealWounds:
		return asset.SpellHealWounds
	}
	return 0
}
