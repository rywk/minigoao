package texture

import (
	"bytes"
	"image"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/assets/img/body"
	"github.com/rywk/minigoao/pkg/client/game/assets/img/spellimg"
	asset "github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
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
			img: spellimg.MeleeHit_png,
		},
		// asset.SpellApoca: {
		// 	c: SpriteConfig{
		// 		Width:      145,
		// 		Height:     145,
		// 		GridW:      4,
		// 		GridH:      4,
		// 		FrameCount: 16,
		// 	},
		// 	img: spellimg.SpellApoca_png,
		// },
		asset.SpellApoca: {
			c: SpriteConfig{
				Width:      80,
				Height:     80,
				GridW:      7,
				GridH:      4,
				FrameCount: 28,
			},
			img: spellimg.SpellApoca2_png,
		},
		asset.SpellDesca: {
			c: SpriteConfig{
				Width:      127,
				Height:     127,
				GridW:      5,
				GridH:      3,
				FrameCount: 15,
			},
			img: spellimg.SpellDesca_png,
		},

		asset.SpellInmo: {
			c: SpriteConfig{
				Width:  96,
				Height: 132,
				DirectionLength: map[direction.D]int{
					direction.Right: 10,
				},
			},
			img: spellimg.SpellParalize_png,
		},
		asset.SpellInmoRm: {
			c: SpriteConfig{
				Width:      68,
				Height:     68,
				GridW:      5,
				GridH:      4,
				FrameCount: 20,
			},
			img: spellimg.SpellInmoRm_png,
		},

		asset.SpellHealWounds: {
			c: SpriteConfig{
				Width:  95,
				Height: 95,
				DirectionLength: map[direction.D]int{
					direction.Right: 10,
				},
			},
			img: spellimg.SpellHealWoundsNew_png,
		},
		asset.SpellResurrect: {
			c: SpriteConfig{
				Width:      76,
				Height:     76,
				GridW:      5,
				GridH:      6,
				FrameCount: 30,
			},
			img: spellimg.SpellResurrect_png,
		},
		asset.NakedBody: {
			c: SpriteConfig{
				Width:  26,
				Height: 46,
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
			img: body.BodyNaked_png,
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
			img: body.Head_png,
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
			img: body.DeadBody_png,
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
			img: body.DeadHead_png,
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

func DecodeIconFromImage(bs []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(bs))
	if err != nil {

		log.Fatal(err)
	}
	return ebiten.NewImageFromImage(ebiten.NewImageFromImage(img).SubImage(image.Rect(0, img.Bounds().Dy()-32, 32, 32)))
}

func AssetFromSpell(s attack.Spell) asset.Image {
	switch s {
	case attack.SpellExplode:
		return asset.SpellApoca
	case attack.SpellParalize:
		return asset.SpellInmo
	case attack.SpellRemoveParalize:
		return asset.SpellInmoRm
	case attack.SpellElectricDischarge:
		return asset.SpellDesca
	case attack.SpellResurrect:
		return asset.SpellResurrect
	case attack.SpellHealWounds:
		return asset.SpellHealWounds
	}
	return 0
}

var itemAssetConfig = map[item.Item]SpriteConfig{
	item.WeaponWindSword: {
		Width:  28,
		Height: 48,
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
	item.WeaponMightySword: {
		Width:  28,
		Height: 48,
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
	item.WeaponFireStaff: {
		Width:  28,
		Height: 48,
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
	item.WeaponDarkDagger: {
		Width:  28,
		Height: 48,
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
	item.ArmorDark: {
		Width:  26,
		Height: 46,
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
	item.ArmorShadow: {
		Width:  28,
		Height: 48,
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
	item.ShieldArcane: {
		Width:  28,
		Height: 46,
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
	item.ShieldTower: {
		Width:  28,
		Height: 46,
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
	item.HatMage: {
		Width:  26,
		Height: 32,
		RealDirMap: map[direction.D]direction.D{
			direction.Front: direction.Right,
			direction.Back:  direction.Right,
			direction.Left:  direction.Right,
			direction.Right: direction.Right,
		},
	},
	item.HelmetPaladin: {
		Width:  26,
		Height: 32,
		RealDirMap: map[direction.D]direction.D{
			direction.Front: direction.Right,
			direction.Back:  direction.Right,
			direction.Left:  direction.Right,
			direction.Right: direction.Right,
		},
	},
}

type ItemData struct {
	SD        SpriteConfig
	Animation *ebiten.Image
	Icon      *ebiten.Image
}

var itemAssetLoaded = map[item.Item]*ItemData{
	item.None: {Animation: ebiten.NewImage(1, 1), Icon: ebiten.NewImage(1, 1)},
}

func init() {
	for i := range item.ItemLen {
		if i == item.None {
			continue
		}
		itemAssetLoaded[i] = &ItemData{
			Animation: DecodeItem(i, false),
			Icon:      DecodeItem(i, true),
		}
	}

}
func DecodeItem(i item.Item, icon bool) *ebiten.Image {
	if i == item.None {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(item.GetAsset(i, icon)))
	if err != nil {
		log.Fatal(err)
	}
	return ebiten.NewImageFromImage(img)
}
func LoadItemIcon(a item.Item) *ebiten.Image {
	if a == item.None {
		return nil
	}
	return itemAssetLoaded[a].Icon
}

// for wearable items weapon, shield armor
func LoadItemAninmatio(a item.Item) A {
	if a == item.None {
		return nil
	}
	return NewBodyAnimation(itemAssetLoaded[a].Animation, itemAssetConfig[a])
}

// for head items, doesnt animate, just direction change
func LoadItemHead(a item.Item) A {
	if a == item.None {
		return nil
	}
	return NewHeadStill(itemAssetLoaded[a].Animation, itemAssetConfig[a])
}
