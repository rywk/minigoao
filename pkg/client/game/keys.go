package game

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/potion"
	"github.com/rywk/minigoao/pkg/constants/spell"
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
	PickInmo       ebiten.Key
	PickInmoRm     ebiten.Key
	PickApoca      ebiten.Key
	PickDesca      ebiten.Key
	PickResurrect  ebiten.Key
	PickHealWounds ebiten.Key

	MeleeCooldown  time.Duration
	SpellCooldown  time.Duration
	SwitchCooldown time.Duration
	PotionCooldown time.Duration
}

var DefaultConfig = KeyConfig{
	Front: ebiten.KeyS,
	Back:  ebiten.KeyW,
	Left:  ebiten.KeyA,
	Right: ebiten.KeyD,

	PickInmo:       ebiten.Key3,
	PickInmoRm:     ebiten.KeyShiftLeft,
	PickApoca:      ebiten.Key2,
	PickDesca:      ebiten.Key1,
	PickHealWounds: ebiten.KeyControlLeft,
	PickResurrect:  ebiten.KeyR,

	PotionHP: ebiten.KeyQ,
	PotionMP: ebiten.KeyF,

	Melee:          ebiten.KeySpace,
	MeleeCooldown:  time.Millisecond * 950,
	SpellCooldown:  time.Millisecond * 950,
	SwitchCooldown: time.Millisecond * 700,
	PotionCooldown: time.Millisecond * 300,
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
	lastPotion    time.Time
	spellPressed  map[ebiten.Key]bool
	spellMap      map[ebiten.Key]spell.Spell
	potionPressed map[ebiten.Key]bool
	potionMap     map[ebiten.Key]spell.Spell

	spellsX, spellsY     float64
	stSpellsX, stSpellsY float64
	movingSpells         bool
	placeholder          *ebiten.Image
	iconImg              *ebiten.Image
	selectedImg          *ebiten.Image

	cursorMode ebiten.CursorShapeType
}

func NewCombatKeys(cfg *KeyConfig) *CombatKeys {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	ck := &CombatKeys{
		cfg:        cfg,
		spell:      spell.None,
		lastMelee:  time.Now(),
		lastCast:   time.Now(),
		lastPotion: time.Now(),
		spellPressed: map[ebiten.Key]bool{
			cfg.PickApoca:      false,
			cfg.PickInmo:       false,
			cfg.PickInmoRm:     false,
			cfg.PickDesca:      false,
			cfg.PickResurrect:  false,
			cfg.PickHealWounds: false,
		},
		spellMap: map[ebiten.Key]spell.Spell{
			cfg.PickApoca:      spell.Apoca,
			cfg.PickInmo:       spell.Inmo,
			cfg.PickInmoRm:     spell.InmoRm,
			cfg.PickDesca:      spell.Desca,
			cfg.PickResurrect:  spell.Revive,
			cfg.PickHealWounds: spell.HealWounds,
			-1:                 spell.None,
		},
		potionPressed: map[ebiten.Key]bool{
			cfg.PotionHP: false,
			cfg.PotionMP: false,
		},
		potionMap: map[ebiten.Key]potion.Potion{
			cfg.PotionHP: potion.Red,
			cfg.PotionMP: potion.Blue,
			-1:           potion.None,
		},
		placeholder: texture.Decode(img.PlaceholderSpellbar_png),
		iconImg:     texture.Decode(img.SpellbarIcons_png),
		selectedImg: texture.Decode(img.SpellSelector_png),
		spellsX:     ScreenWidth - 386,
		spellsY:     ScreenHeight - 100,
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
	apoca, inmo, inmoRm, desca, healWounds, resurrect := ebiten.IsKeyPressed(ck.cfg.PickApoca),
		ebiten.IsKeyPressed(ck.cfg.PickInmo),
		ebiten.IsKeyPressed(ck.cfg.PickInmoRm),
		ebiten.IsKeyPressed(ck.cfg.PickDesca),
		ebiten.IsKeyPressed(ck.cfg.PickHealWounds),
		ebiten.IsKeyPressed(ck.cfg.PickResurrect)

	if apoca && !ck.spellPressed[ck.cfg.PickApoca] {
		ck.spellPressed[ck.cfg.PickApoca] = true
		ck.lastSpellKey = ck.cfg.PickApoca
	} else if apoca && !ck.spellPressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickApoca
	} else if !apoca && ck.spellPressed[ck.cfg.PickApoca] {
		ck.spellPressed[ck.cfg.PickApoca] = false

	}

	if inmo && !ck.spellPressed[ck.cfg.PickInmo] {
		ck.spellPressed[ck.cfg.PickInmo] = true
		ck.lastSpellKey = ck.cfg.PickInmo
	} else if inmo && !ck.spellPressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickInmo
	} else if !inmo && ck.spellPressed[ck.cfg.PickInmo] {
		ck.spellPressed[ck.cfg.PickInmo] = false
	}

	if inmoRm && !ck.spellPressed[ck.cfg.PickInmoRm] {
		ck.spellPressed[ck.cfg.PickInmoRm] = true
		ck.lastSpellKey = ck.cfg.PickInmoRm
	} else if inmoRm && !ck.spellPressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickInmoRm
	} else if !inmoRm && ck.spellPressed[ck.cfg.PickInmoRm] {
		ck.spellPressed[ck.cfg.PickInmoRm] = false
	}

	if desca && !ck.spellPressed[ck.cfg.PickDesca] {
		ck.spellPressed[ck.cfg.PickDesca] = true
		ck.lastSpellKey = ck.cfg.PickDesca
	} else if desca && !ck.spellPressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickDesca
	} else if !desca && ck.spellPressed[ck.cfg.PickDesca] {
		ck.spellPressed[ck.cfg.PickDesca] = false
	}

	if healWounds && !ck.spellPressed[ck.cfg.PickHealWounds] {
		ck.spellPressed[ck.cfg.PickHealWounds] = true
		ck.lastSpellKey = ck.cfg.PickHealWounds
	} else if healWounds && !ck.spellPressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickHealWounds
	} else if !healWounds && ck.spellPressed[ck.cfg.PickHealWounds] {
		ck.spellPressed[ck.cfg.PickHealWounds] = false
	}

	if resurrect && !ck.spellPressed[ck.cfg.PickResurrect] {
		ck.spellPressed[ck.cfg.PickResurrect] = true
		ck.lastSpellKey = ck.cfg.PickResurrect
	} else if resurrect && !ck.spellPressed[ck.lastSpellKey] {
		ck.lastSpellKey = ck.cfg.PickResurrect
	} else if !resurrect && ck.spellPressed[ck.cfg.PickResurrect] {
		ck.spellPressed[ck.cfg.PickResurrect] = false
	}

	ck.spell = ck.spellMap[ck.lastSpellKey]
}

func (ck *CombatKeys) CastSpell() (bool, spell.Spell, int, int) {
	ck.SetSpell()
	// if spell is picked and mouse mode is not crosshair, activate and set
	if ck.spell != spell.None && ck.cursorMode != ebiten.CursorShapeCrosshair {
		ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)
		ck.cursorMode = ebiten.CursorShapeCrosshair
	}

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

	if ck.spell == spell.None && ck.cursorMode == ebiten.CursorShapeCrosshair {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		ck.cursorMode = ebiten.CursorShapeDefault
	}
	return false, spell.None, 0, 0
}

func (ck *CombatKeys) ShowSpellPicker(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(ck.spellsX), float64(ck.spellsY))
	screen.DrawImage(ck.placeholder, op)
	opselect := &ebiten.DrawImageOptions{}
	switch ck.spell {
	case spell.Revive:
		opselect.GeoM.Translate(float64(ck.spellsX+5), float64(ck.spellsY-8))
		screen.DrawImage(ck.selectedImg, opselect)
	case spell.HealWounds:
		opselect.GeoM.Translate(float64(ck.spellsX+60), float64(ck.spellsY-8))
		screen.DrawImage(ck.selectedImg, opselect)
	case spell.InmoRm:
		opselect.GeoM.Translate(float64(ck.spellsX+115), float64(ck.spellsY-8))
		screen.DrawImage(ck.selectedImg, opselect)
	case spell.Inmo:
		opselect.GeoM.Translate(float64(ck.spellsX+175), float64(ck.spellsY-8))
		screen.DrawImage(ck.selectedImg, opselect)
	case spell.Desca:
		opselect.GeoM.Translate(float64(ck.spellsX+230), float64(ck.spellsY-8))
		screen.DrawImage(ck.selectedImg, opselect)
	case spell.Apoca:
		opselect.GeoM.Translate(float64(ck.spellsX+284), float64(ck.spellsY-8))
		screen.DrawImage(ck.selectedImg, opselect)
	}
	screen.DrawImage(ck.iconImg, op)
}

func (ck *CombatKeys) MoveSpellPicker() {
	cx, cy := ebiten.CursorPosition()
	rect := ck.placeholder.Bounds()
	if cx > int(ck.spellsX)+rect.Min.X && cx < int(ck.spellsX)+rect.Max.X && cy > int(ck.spellsY)+rect.Min.Y && cy < int(ck.spellsY)+rect.Max.Y || ck.movingSpells {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !ck.movingSpells {
				ck.stSpellsX = float64(cx) - ck.spellsX
				ck.stSpellsY = float64(cy) - ck.spellsY
			}
			ck.movingSpells = true
			ck.spellsX, ck.spellsY = float64(cx)-ck.stSpellsX, float64(cy)-ck.stSpellsY
		} else {
			ck.movingSpells = false
		}
	}
}

func (ck *CombatKeys) PressedPotion() potion.Potion {
	red, blue := ebiten.IsKeyPressed(ck.cfg.PotionHP),
		ebiten.IsKeyPressed(ck.cfg.PotionMP)

	if red && !ck.potionPressed[ck.cfg.PotionHP] {
		ck.potionPressed[ck.cfg.PotionHP] = true
		ck.lastPotionKey = ck.cfg.PotionHP
	} else if red && !ck.potionPressed[ck.lastPotionKey] {
		ck.lastPotionKey = ck.cfg.PotionHP
	} else if !red && ck.potionPressed[ck.cfg.PotionHP] {
		ck.potionPressed[ck.cfg.PotionHP] = false

	}

	if blue && !ck.potionPressed[ck.cfg.PotionMP] {
		ck.potionPressed[ck.cfg.PotionMP] = true
		ck.lastPotionKey = ck.cfg.PotionMP
	} else if blue && !ck.potionPressed[ck.lastPotionKey] {
		ck.lastPotionKey = ck.cfg.PotionMP
	} else if !blue && ck.potionPressed[ck.cfg.PotionMP] {
		ck.potionPressed[ck.cfg.PotionMP] = false
	}

	if !red && !blue {
		ck.lastPotionKey = -1
	}
	p := ck.potionMap[ck.lastPotionKey]
	if p != potion.None && time.Since(ck.lastPotion) > ck.cfg.PotionCooldown {
		ck.lastPotion = time.Now()
		return p
	}
	return potion.None
}
