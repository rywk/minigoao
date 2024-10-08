package game

import (
	"image/color"
	"log"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/msgs"
)

type KeyConfig struct {
	Front *Input
	Back  *Input
	Left  *Input
	Right *Input

	PotionHP *Input
	PotionMP *Input

	Melee *Input

	// Spell picker
	PickParalize          *Input
	PickParalizeRm        *Input
	PickExplode           *Input
	PickElectricDischarge *Input
	PickResurrect         *Input
	PickHealWounds        *Input

	PotionCooldown time.Duration
	// Cooldown to cast spells or do a melee attack
	// spells and melee attacks trigger this cd
	CooldownAction time.Duration
	// Cooldown for each spell
	CooldownSpells [spell.Len]time.Duration
	// Cooldown for melee
	CooldownMelee time.Duration
}

var DefaultConfig = KeyConfig{
	Front: NewInputPtr(ebiten.KeyS),
	Back:  NewInputPtr(ebiten.KeyW),
	Left:  NewInputPtr(ebiten.KeyA),
	Right: NewInputPtr(ebiten.KeyD),

	PickParalize:          NewInputPtr(ebiten.Key3),
	PickParalizeRm:        NewInputPtr(ebiten.KeyShiftLeft),
	PickExplode:           NewInputPtr(ebiten.Key2),
	PickElectricDischarge: NewInputPtr(ebiten.Key1),
	PickHealWounds:        NewInputPtr(ebiten.KeyControlLeft),
	PickResurrect:         NewInputPtr(ebiten.KeyR),

	PotionHP: NewInputPtr(ebiten.MouseButtonRight),
	PotionMP: NewInputPtr(ebiten.KeyF),

	Melee:          NewInputPtr(ebiten.KeySpace),
	PotionCooldown: time.Millisecond * 300,

	CooldownAction: time.Millisecond * 400,
	CooldownMelee:  time.Millisecond * 900,
	CooldownSpells: [spell.Len]time.Duration{
		0,
		time.Millisecond * 950,   //Paralize
		time.Millisecond * 950,   //RemoveParalize
		time.Millisecond * 950,   //HealWounds
		time.Millisecond * 10000, //Resurrect
		time.Millisecond * 750,   //ElectricDischarge
		time.Millisecond * 1000,  //Explode
	},
}

type Keys struct {
	g            *Game
	cfg          *KeyConfig
	last         *Input
	pressed      map[*Input]bool
	directionMap map[*Input]direction.D

	keysLocked    bool
	chatInputOpen bool
	enterDown     bool
	typer         *typing.Typer
	sentChat      string
	lastChat      time.Time
	openCloseImg  *ebiten.Image

	spell           spell.Spell
	clickDown       bool
	potion          msgs.Item
	lastSpellKey    *Input
	lastPotionInput *Input
	lastPotion      time.Time
	spellPressed    map[*Input]bool
	spellMap        map[*Input]spell.Spell
	potionPressed   map[*Input]bool
	potionMap       map[*Input]msgs.Item
	cursorMode      ebiten.CursorShapeType

	LastAction time.Time
	LastMelee  time.Time
	LastSpells [spell.Len]time.Time
}

func NewKeys(g *Game, cfg *KeyConfig) *Keys {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	k := &Keys{
		g:            g,
		cfg:          cfg,
		typer:        typing.NewTyper(),
		openCloseImg: ebiten.NewImage(8, 14),
		pressed: map[*Input]bool{
			cfg.Front: false,
			cfg.Back:  false,
			cfg.Left:  false,
			cfg.Right: false,
		},
		directionMap: map[*Input]direction.D{
			cfg.Front: direction.Front,
			cfg.Back:  direction.Back,
			cfg.Left:  direction.Left,
			cfg.Right: direction.Right,
			&NoInput:  direction.Still,
		},

		spell:      spell.None,
		lastPotion: time.Now(),
		spellPressed: map[*Input]bool{
			cfg.PickExplode:           false,
			cfg.PickParalize:          false,
			cfg.PickParalizeRm:        false,
			cfg.PickElectricDischarge: false,
			cfg.PickResurrect:         false,
			cfg.PickHealWounds:        false,
		},
		spellMap: map[*Input]spell.Spell{
			cfg.PickExplode:           spell.Explode,
			cfg.PickParalize:          spell.Paralize,
			cfg.PickParalizeRm:        spell.RemoveParalize,
			cfg.PickElectricDischarge: spell.ElectricDischarge,
			cfg.PickResurrect:         spell.Resurrect,
			cfg.PickHealWounds:        spell.HealWounds,
			&NoInput:                  spell.None,
		},
		potionPressed: map[*Input]bool{
			cfg.PotionHP: false,
			cfg.PotionMP: false,
		},
		potionMap: map[*Input]msgs.Item{
			cfg.PotionHP: msgs.ItemHealthPotion,
			cfg.PotionMP: msgs.ItemManaPotion,
			&NoInput:     msgs.ItemNone,
		},

		LastSpells: [spell.Len]time.Time{
			time.Now().Add(-time.Second * 10), //Paralize
			time.Now().Add(-time.Second * 10), //RemoveParalize
			time.Now().Add(-time.Second * 10), //HealWounds
			time.Now().Add(-time.Second * 10), //Resurrect
			time.Now().Add(-time.Second * 10), //ElectricDischarge
			time.Now().Add(-time.Second * 10), //Explode
		},
	}
	k.openCloseImg.Fill(color.RGBA{176, 82, 51, 0})
	return k
}

func (k *Keys) ListenMovement() {
	if k.keysLocked {
		return
	}
	front, back, left, right := k.cfg.Front.IsPressed(),
		k.cfg.Back.IsPressed(),
		k.cfg.Left.IsPressed(),
		k.cfg.Right.IsPressed()

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
		k.last = &NoInput
	}
}

func (k *Keys) MovingTo() direction.D {
	return k.directionMap[k.last]
}

func (k *Keys) MeleeHit() bool {
	if k.keysLocked {
		return false
	}
	hit := false
	if k.cfg.Melee.IsPressed() &&
		time.Since(k.LastMelee) > k.cfg.CooldownMelee &&
		time.Since(k.LastAction) > k.cfg.CooldownAction {
		k.LastMelee = time.Now()
		k.LastAction = k.LastMelee
		hit = true
	}
	return hit
}

func (k *Keys) ListenSpell() {
	if k.keysLocked {
		return
	}

	apoca, inmo, inmoRm, desca, healWounds, resurrect := k.cfg.PickExplode.IsPressed(),
		k.cfg.PickParalize.IsPressed(),
		k.cfg.PickParalizeRm.IsPressed(),
		k.cfg.PickElectricDischarge.IsPressed(),
		k.cfg.PickHealWounds.IsPressed(),
		k.cfg.PickResurrect.IsPressed()

	if apoca && !k.spellPressed[k.cfg.PickExplode] {
		k.spellPressed[k.cfg.PickExplode] = true
		k.lastSpellKey = k.cfg.PickExplode
	} else if !apoca && k.spellPressed[k.cfg.PickExplode] {
		k.spellPressed[k.cfg.PickExplode] = false
	}

	if inmo && !k.spellPressed[k.cfg.PickParalize] {
		k.spellPressed[k.cfg.PickParalize] = true
		k.lastSpellKey = k.cfg.PickParalize
	} else if !inmo && k.spellPressed[k.cfg.PickParalize] {
		k.spellPressed[k.cfg.PickParalize] = false
	}

	if inmoRm && !k.spellPressed[k.cfg.PickParalizeRm] {
		k.spellPressed[k.cfg.PickParalizeRm] = true
		k.lastSpellKey = k.cfg.PickParalizeRm
	} else if !inmoRm && k.spellPressed[k.cfg.PickParalizeRm] {
		k.spellPressed[k.cfg.PickParalizeRm] = false
	}

	if desca && !k.spellPressed[k.cfg.PickElectricDischarge] {
		k.spellPressed[k.cfg.PickElectricDischarge] = true
		k.lastSpellKey = k.cfg.PickElectricDischarge
	} else if !desca && k.spellPressed[k.cfg.PickElectricDischarge] {
		k.spellPressed[k.cfg.PickElectricDischarge] = false
	}

	if healWounds && !k.spellPressed[k.cfg.PickHealWounds] {
		k.spellPressed[k.cfg.PickHealWounds] = true
		k.lastSpellKey = k.cfg.PickHealWounds
	} else if !healWounds && k.spellPressed[k.cfg.PickHealWounds] {
		k.spellPressed[k.cfg.PickHealWounds] = false
	}

	if resurrect && !k.spellPressed[k.cfg.PickResurrect] {
		k.spellPressed[k.cfg.PickResurrect] = true
		k.lastSpellKey = k.cfg.PickResurrect
	} else if !resurrect && k.spellPressed[k.cfg.PickResurrect] {
		k.spellPressed[k.cfg.PickResurrect] = false
	}
	if apoca || inmo || inmoRm || desca || healWounds || resurrect {
		k.spell = k.spellMap[k.lastSpellKey]
	}
}

func (k *Keys) CastSpell() (bool, spell.Spell, int, int) {
	// if spell is picked and mouse mode is not crosshair, activate and set
	if k.spell != spell.None && k.cursorMode != ebiten.CursorShapeCrosshair && k.g.mouseY < ScreenHeight-64 {
		ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)
		k.cursorMode = ebiten.CursorShapeCrosshair
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
		k.clickDown = true
	} else {
		if k.clickDown && k.spell != spell.None && k.g.mouseY < ScreenHeight-64 &&
			time.Since(k.LastSpells[k.spell]) > k.cfg.CooldownSpells[k.spell] &&
			time.Since(k.LastAction) > k.cfg.CooldownAction {
			k.LastAction = time.Now()
			k.LastSpells[k.spell] = k.LastAction
			k.clickDown = false
			pspell := k.spell
			k.spell = spell.None
			k.lastSpellKey = &NoInput
			return true, pspell, k.g.mouseX, k.g.mouseY
		}
		k.clickDown = false
	}

	if k.spell == spell.None && k.cursorMode == ebiten.CursorShapeCrosshair || k.g.mouseY >= ScreenHeight-64 {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		k.cursorMode = ebiten.CursorShapeDefault
	}
	return false, spell.None, 0, 0
}

func (k *Keys) PressedPotion() msgs.Item {
	if k.keysLocked {
		return msgs.ItemNone
	}
	red, blue := k.cfg.PotionHP.IsPressed(),
		k.cfg.PotionMP.IsPressed()

	if red && !k.potionPressed[k.cfg.PotionHP] {
		k.potionPressed[k.cfg.PotionHP] = true
		k.lastPotionInput = k.cfg.PotionHP
	} else if red && !k.potionPressed[k.lastPotionInput] {
		k.lastPotionInput = k.cfg.PotionHP
	} else if !red && k.potionPressed[k.cfg.PotionHP] {
		k.potionPressed[k.cfg.PotionHP] = false

	}

	if blue && !k.potionPressed[k.cfg.PotionMP] {
		k.potionPressed[k.cfg.PotionMP] = true
		k.lastPotionInput = k.cfg.PotionMP
	} else if blue && !k.potionPressed[k.lastPotionInput] {
		k.lastPotionInput = k.cfg.PotionMP
	} else if !blue && k.potionPressed[k.cfg.PotionMP] {
		k.potionPressed[k.cfg.PotionMP] = false
	}

	if !red && !blue {
		k.lastPotionInput = &NoInput
	}
	p := k.potionMap[k.lastPotionInput]
	if p != msgs.ItemNone && time.Since(k.lastPotion) > k.cfg.PotionCooldown {
		k.lastPotion = time.Now()
		return p
	}

	return msgs.ItemNone
}
func (k *Keys) DrawChat(screen *ebiten.Image, x, y int) {
	// Blink the cursor.
	if k.chatInputOpen {
		t := k.typer.Text
		off := len(t) * 3
		if k.g.counter%60 < 30 {
			t += "_"
		}
		text.PrintColAt(screen, t, x-off, y, color.RGBA{176, 82, 51, 0})
	} else if k.sentChat != "" {
		off := len(k.sentChat) * 3
		text.PrintAt(screen, k.sentChat, x-off, y)
	}
	if k.enterDown {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x-3), float64(y))
		screen.DrawImage(k.openCloseImg, op)
	}
}
func (k *Keys) ChatMessage() string {
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if !k.enterDown {
			k.enterDown = true
			if k.chatInputOpen && k.typer.Text != "" {
				k.lastChat = time.Now()
				k.sentChat = strings.Trim(k.typer.Text, "\n")
				k.typer.Text = ""
				k.chatInputOpen = !k.chatInputOpen
				k.keysLocked = !k.keysLocked
				return k.sentChat
			}
			k.chatInputOpen = !k.chatInputOpen
			k.keysLocked = !k.keysLocked
		}
	} else {
		k.enterDown = false
	}

	if k.chatInputOpen {
		r := strings.NewReplacer("\n", "")
		k.typer.Update()
		k.typer.Text = r.Replace(k.typer.Text)
	}
	if time.Since(k.lastChat) > constants.ChatMsgTTL {
		k.sentChat = ""
	}
	return ""
}

var NoInput = Input{Keyboard: -1, Mouse: -1}

func EmptyInput() *Input {
	return &Input{Keyboard: -1, Mouse: -1}
}

type Input struct {
	Mouse    ebiten.MouseButton
	Keyboard ebiten.Key
}

type KBind interface {
	comparable
	V() Input
	VPtr() *Input
	IsPressed() bool
	String() string
	Empty() bool
	Set(Input)
}

func (i *Input) VPtr() *Input {
	return i
}
func (i *Input) V() Input {
	return *i
}

func (i *Input) Empty() bool {
	return *i == NoInput
}

func (i *Input) Set(ni Input) {
	if i == nil {
		i = &Input{}
	}
	i.Keyboard = ni.Keyboard
	i.Mouse = ni.Mouse
}

func (i *Input) IsPressed() bool {
	if i.Keyboard != -1 {
		return ebiten.IsKeyPressed(i.Keyboard)
	}
	return ebiten.IsMouseButtonPressed(i.Mouse)
}

func (i *Input) String() string {
	if i == nil {
		return "nil"
	}
	if *i == NoInput {
		return "no input"
	}
	if i.Keyboard != -1 {
		return i.Keyboard.String()
	}
	switch i.Mouse {
	case ebiten.MouseButtonRight:
		return "MouseButtonRight"
	case ebiten.MouseButtonLeft:
		return "MouseButtonLeft"
	case ebiten.MouseButtonMiddle:
		return "MouseButtonMiddle"
	case ebiten.MouseButton3:
		return "MouseButton3"
	case ebiten.MouseButton4:
		return "MouseButton4"
	}
	return "idk"
}

func NewInput(input interface{}) Input {
	i := Input{}
	switch v := input.(type) {
	case ebiten.MouseButton:
		i.Mouse = v
		i.Keyboard = -1
	case ebiten.Key:
		i.Mouse = -1
		i.Keyboard = v
	default:
		log.Printf("BIG ERROR\n")
	}
	return i
}

func NewInputPtr(input interface{}) *Input {
	i := Input{}
	switch v := input.(type) {
	case ebiten.MouseButton:
		i.Mouse = v
		i.Keyboard = -1
	case ebiten.Key:
		i.Mouse = -1
		i.Keyboard = v
	default:
		log.Printf("BIG ERROR\n")
	}
	return &i
}
