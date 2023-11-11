package game

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants/potion"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/direction"
)

type KeyConfig struct {
	Front ebiten.Key
	Back  ebiten.Key
	Left  ebiten.Key
	Right ebiten.Key

	PotionHP ebiten.Key
	PotionMP ebiten.Key

	Melee ebiten.Key

	// Spell picker
	PickInmo   ebiten.Key
	PickInmoRm ebiten.Key
	PickApoca  ebiten.Key

	MeleeCooldown  time.Duration
	SpellCooldown  time.Duration
	SwitchCooldown time.Duration
}

var DefaultConfig = KeyConfig{
	Front: ebiten.KeyS,
	Back:  ebiten.KeyW,
	Left:  ebiten.KeyA,
	Right: ebiten.KeyD,

	PickInmo:   ebiten.Key3,
	PickInmoRm: ebiten.KeyShiftLeft,
	PickApoca:  ebiten.Key2,

	Melee:          ebiten.KeySpace,
	MeleeCooldown:  time.Millisecond * 950,
	SpellCooldown:  time.Millisecond * 950,
	SwitchCooldown: time.Millisecond * 700,
}

type Keys struct {
	cfg     *KeyConfig
	last    ebiten.Key
	pressed map[ebiten.Key]bool

	directionMap map[ebiten.Key]direction.D
}

func NewKeys(cfg *KeyConfig) *Keys {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	k := &Keys{
		cfg: cfg,
		pressed: map[ebiten.Key]bool{
			cfg.Front: false,
			cfg.Back:  false,
			cfg.Left:  false,
			cfg.Right: false,
		},
		directionMap: map[ebiten.Key]direction.D{
			cfg.Front: direction.Front,
			cfg.Back:  direction.Back,
			cfg.Left:  direction.Left,
			cfg.Right: direction.Right,
			-1:        direction.Still,
		},
	}
	return k
}

func (k *Keys) ListenMovement() {
	front, back, left, right := ebiten.IsKeyPressed(k.cfg.Front),
		ebiten.IsKeyPressed(k.cfg.Back),
		ebiten.IsKeyPressed(k.cfg.Left),
		ebiten.IsKeyPressed(k.cfg.Right)

	if front && !k.pressed[k.cfg.Front] {
		k.pressed[k.cfg.Front] = true
		k.last = k.cfg.Front
	} else if front && !k.pressed[k.last] {
		k.last = k.cfg.Front
	} else if !front && k.pressed[k.cfg.Front] {
		k.pressed[k.cfg.Front] = false

	}

	if back && !k.pressed[k.cfg.Back] {
		k.pressed[k.cfg.Back] = true
		k.last = k.cfg.Back
	} else if back && !k.pressed[k.last] {
		k.last = k.cfg.Back
	} else if !back && k.pressed[k.cfg.Back] {
		k.pressed[k.cfg.Back] = false
	}

	if left && !k.pressed[k.cfg.Left] {
		k.pressed[k.cfg.Left] = true
		k.last = k.cfg.Left
	} else if left && !k.pressed[k.last] {
		k.last = k.cfg.Left
	} else if !left && k.pressed[k.cfg.Left] {
		k.pressed[k.cfg.Left] = false
	}

	if right && !k.pressed[k.cfg.Right] {
		k.pressed[k.cfg.Right] = true
		k.last = k.cfg.Right
	} else if right && !k.pressed[k.last] {
		k.last = k.cfg.Right
	} else if !right && k.pressed[k.cfg.Right] {
		k.pressed[k.cfg.Right] = false
	}

	if !front && !back && !left && !right {
		k.last = -1
	}
}

func (k *Keys) MovingTo() direction.D {
	return k.directionMap[k.last]
}

type CombatKeys struct {
	cfg           *KeyConfig
	spell         spell.Spell
	potion        potion.Potion
	lastSpellKey  ebiten.Key
	lastPotionKey ebiten.Key
	lastCast      time.Time
	lastMelee     time.Time
	pressed       map[ebiten.Key]bool
	spellMap      map[ebiten.Key]spell.Spell
	potionMap     map[ebiten.Key]spell.Spell

	placeholder                     *ebiten.Image
	iconApoca, iconInmo, iconInmoRm *ebiten.Image
}

func NewCombatKeys(cfg *KeyConfig) *CombatKeys {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	ck := &CombatKeys{
		cfg:       cfg,
		spell:     spell.None,
		lastMelee: time.Now(),
		lastCast:  time.Now(),
		pressed: map[ebiten.Key]bool{
			cfg.PickApoca:  false,
			cfg.PickInmo:   false,
			cfg.PickInmoRm: false,
		},
		spellMap: map[ebiten.Key]spell.Spell{
			cfg.PickApoca:  spell.Apoca,
			cfg.PickInmo:   spell.Inmo,
			cfg.PickInmoRm: spell.InmoRm,
			-1:             spell.None,
		},
		potionMap: map[ebiten.Key]potion.Potion{
			cfg.PotionHP: potion.Red,
			cfg.PotionMP: potion.Blue,
			-1:           spell.None,
		},
		placeholder: texture.Decode(img.PlaceholderSpellIcon_png),
		iconApoca:   texture.Decode(img.IconSpellApoca_png),
		iconInmo:    texture.Decode(img.IconSpellInmo_png),
		iconInmoRm:  texture.Decode(img.IconSpellInmoRm_png),
	}
	return ck
}

func (ck *CombatKeys) MeleeHit() bool {
	hit := false
	if ebiten.IsKeyPressed(ck.cfg.Melee) &&
		time.Since(ck.lastMelee) > ck.cfg.MeleeCooldown &&
		time.Since(ck.lastCast) > ck.cfg.SwitchCooldown {
		ck.lastMelee = time.Now()
		hit = true
	}
	return hit
}

func (ck *CombatKeys) SetSpell() {
	apoca, inmo, inmoRm := ebiten.IsKeyPressed(ck.cfg.PickApoca),
		ebiten.IsKeyPressed(ck.cfg.PickInmo),
		ebiten.IsKeyPressed(ck.cfg.PickInmoRm)

	if apoca && !ck.pressed[ck.cfg.PickApoca] {
		ck.pressed[ck.cfg.PickApoca] = true
		ck.lastSpellKey = ck.cfg.PickApoca
	} else if apoca && !ck.pressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickApoca
	} else if !apoca && ck.pressed[ck.cfg.PickApoca] {
		ck.pressed[ck.cfg.PickApoca] = false

	}

	if inmo && !ck.pressed[ck.cfg.PickInmo] {
		ck.pressed[ck.cfg.PickInmo] = true
		ck.lastSpellKey = ck.cfg.PickInmo
	} else if inmo && !ck.pressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickInmo
	} else if !inmo && ck.pressed[ck.cfg.PickInmo] {
		ck.pressed[ck.cfg.PickInmo] = false
	}

	if inmoRm && !ck.pressed[ck.cfg.PickInmoRm] {
		ck.pressed[ck.cfg.PickInmoRm] = true
		ck.lastSpellKey = ck.cfg.PickInmoRm
	} else if inmoRm && !ck.pressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickInmoRm
	} else if !inmoRm && ck.pressed[ck.cfg.PickInmoRm] {
		ck.pressed[ck.cfg.PickInmoRm] = false
	}
	ck.spell = ck.spellMap[ck.lastSpellKey]
}

func (ck *CombatKeys) CastSpell() (bool, spell.Spell, int, int) {
	ck.SetSpell()
	if ck.spell != spell.None && ebiten.IsMouseButtonPressed(ebiten.MouseButton0) &&
		time.Since(ck.lastCast) > ck.cfg.SpellCooldown &&
		time.Since(ck.lastMelee) > ck.cfg.SwitchCooldown {
		ck.lastCast = time.Now()
		x, y := ebiten.CursorPosition()
		pspell := ck.spell
		ck.spell = spell.None
		ck.lastSpellKey = -1
		return true, pspell, x, y
	}
	return false, spell.None, 0, 0
}

func (ck *CombatKeys) ShowSpellPicker(screen *ebiten.Image, op *ebiten.DrawImageOptions) {
	screen.DrawImage(ck.placeholder, op)
	switch ck.spell {
	case spell.Apoca:
		screen.DrawImage(ck.iconApoca, op)
	case spell.Inmo:
		screen.DrawImage(ck.iconInmo, op)
	case spell.InmoRm:
		screen.DrawImage(ck.iconInmoRm, op)
	case spell.None:
		// placeholder
	}
}

func (ck *CombatKeys) PressedPotion() {
	red, blue := ebiten.IsKeyPressed(ck.cfg.PotionHP),
		ebiten.IsKeyPressed(ck.cfg.PotionMP)

	if red && !ck.pressed[ck.cfg.PotionHP] {
		ck.pressed[ck.cfg.PotionHP] = true
		ck.lastPotionKey = ck.cfg.PotionHP
	} else if red && !ck.pressed[ck.lastPotionKey] {
		ck.lastPotionKey = ck.cfg.PotionHP
	} else if !red && ck.pressed[ck.cfg.PotionHP] {
		ck.pressed[ck.cfg.PotionHP] = false

	}

	if blue && !ck.pressed[ck.cfg.PotionMP] {
		ck.pressed[ck.cfg.PotionMP] = true
		ck.lastPotionKey = ck.cfg.PotionMP
	} else if blue && !ck.pressed[ck.lastPotionKey] {
		ck.lastPotionKey = ck.cfg.PotionMP
	} else if !blue && ck.pressed[ck.cfg.PotionMP] {
		ck.pressed[ck.cfg.PotionMP] = false
	}

	ck.potion = ck.spellMap[ck.lastPotionKey]
}
