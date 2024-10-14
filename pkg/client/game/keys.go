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
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
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
}

func (k *KeyConfig) ToMsgs() msgs.KeyConfig {
	return msgs.KeyConfig{
		Front: k.Front.ToMsgs(),
		Back:  k.Back.ToMsgs(),
		Left:  k.Left.ToMsgs(),
		Right: k.Right.ToMsgs(),

		PotionHP: k.PotionHP.ToMsgs(),
		PotionMP: k.PotionMP.ToMsgs(),

		Melee: k.Melee.ToMsgs(),

		// Spell picker
		PickParalize:          k.PickParalize.ToMsgs(),
		PickParalizeRm:        k.PickParalizeRm.ToMsgs(),
		PickExplode:           k.PickExplode.ToMsgs(),
		PickElectricDischarge: k.PickElectricDischarge.ToMsgs(),
		PickResurrect:         k.PickResurrect.ToMsgs(),
		PickHealWounds:        k.PickHealWounds.ToMsgs(),
	}
}

func (k *KeyConfig) FromMsgs(kdb msgs.KeyConfig) {
	k.Front = InputFromMsgs(kdb.Front)
	k.Back = InputFromMsgs(kdb.Back)
	k.Left = InputFromMsgs(kdb.Left)
	k.Right = InputFromMsgs(kdb.Right)

	k.PotionHP = InputFromMsgs(kdb.PotionHP)
	k.PotionMP = InputFromMsgs(kdb.PotionMP)

	k.Melee = InputFromMsgs(kdb.Melee)

	// Spell picker
	k.PickParalize = InputFromMsgs(kdb.PickParalize)
	k.PickParalizeRm = InputFromMsgs(kdb.PickParalizeRm)
	k.PickExplode = InputFromMsgs(kdb.PickExplode)
	k.PickElectricDischarge = InputFromMsgs(kdb.PickElectricDischarge)
	k.PickResurrect = InputFromMsgs(kdb.PickResurrect)
	k.PickHealWounds = InputFromMsgs(kdb.PickHealWounds)
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

	Melee: NewInputPtr(ebiten.MouseButton4),
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
	sentChat      *ebiten.Image
	sentChatOff   int
	lastChat      time.Time
	openCloseImg  *ebiten.Image

	clickDown       bool
	potion          item.Item
	lastSpellKey    *Input
	lastPotionInput *Input
	lastPotion      time.Time
	spellPressed    map[*Input]bool
	spellMap        map[*Input]attack.Spell
	potionPressed   map[*Input]bool
	potionMap       map[*Input]item.Item
	cursorMode      ebiten.CursorShapeType

	LastAction time.Time
	LastMelee  time.Time
	LastSpells [attack.SpellLen]time.Time

	PotionCooldown time.Duration

	ChatMsgImage *ebiten.Image
}

func NewKeys(g *Game, cfg *KeyConfig) *Keys {
	if cfg == nil {
		cfg = &DefaultConfig
	} else {

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

		lastPotion: time.Now(),
		spellPressed: map[*Input]bool{
			cfg.PickExplode:           false,
			cfg.PickParalize:          false,
			cfg.PickParalizeRm:        false,
			cfg.PickElectricDischarge: false,
			cfg.PickResurrect:         false,
			cfg.PickHealWounds:        false,
		},
		spellMap: map[*Input]attack.Spell{
			cfg.PickExplode:           attack.SpellExplode,
			cfg.PickParalize:          attack.SpellParalize,
			cfg.PickParalizeRm:        attack.SpellRemoveParalize,
			cfg.PickElectricDischarge: attack.SpellElectricDischarge,
			cfg.PickResurrect:         attack.SpellResurrect,
			cfg.PickHealWounds:        attack.SpellHealWounds,
			&NoInput:                  attack.SpellNone,
		},
		potionPressed: map[*Input]bool{
			cfg.PotionHP: false,
			cfg.PotionMP: false,
		},
		potionMap: map[*Input]item.Item{
			cfg.PotionHP: item.HealthPotion,
			cfg.PotionMP: item.ManaPotion,
			&NoInput:     item.None,
		},

		LastSpells: [attack.SpellLen]time.Time{
			time.Now().Add(-time.Second * 10), //Paralize
			time.Now().Add(-time.Second * 10), //RemoveParalize
			time.Now().Add(-time.Second * 10), //HealWounds
			time.Now().Add(-time.Second * 10), //Resurrect
			time.Now().Add(-time.Second * 10), //ElectricDischarge
			time.Now().Add(-time.Second * 10), //Explode
		},
		PotionCooldown: constants.PotionCooldown,
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
		time.Since(k.LastMelee) > item.ItemProps[k.g.player.Inv.GetWeapon()].WeaponProp.Cooldown &&
		time.Since(k.LastAction) > k.g.player.Exp.Stats.SwitchCD {
		k.LastMelee = time.Now()
		//k.LastAction = k.LastMelee
		hit = true
	}
	return hit
}

func (k *Keys) ListenSpell() attack.Spell {
	if k.keysLocked {
		return attack.SpellNone
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
		return k.spellMap[k.lastSpellKey]
		//k.spell = k.spellMap[k.lastSpellKey]
	}
	return attack.SpellNone
}

func (k *Keys) CastSpell() (bool, int, int) {
	selectedSpell := k.g.SelectedSpell
	if selectedSpell == attack.SpellNone {
		if k.cursorMode == ebiten.CursorShapeCrosshair || k.g.mouseY >= ScreenHeight-64 {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
			k.cursorMode = ebiten.CursorShapeDefault
		}
		return false, 0, 0
	}

	// if spell is picked and mouse mode is not crosshair, activate and set
	if selectedSpell != attack.SpellNone && k.cursorMode != ebiten.CursorShapeCrosshair && k.g.mouseY < ScreenHeight-64 {
		ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)
		k.cursorMode = ebiten.CursorShapeCrosshair
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
		k.clickDown = true
	} else {
		if k.clickDown && selectedSpell != attack.SpellNone && k.g.mouseY < ScreenHeight-64 &&
			time.Since(k.LastSpells[selectedSpell]) > attack.SpellProps[selectedSpell].BaseCooldown &&
			time.Since(k.LastAction) > k.g.player.Exp.Stats.ActionCD &&
			time.Since(k.LastMelee) > k.g.player.Exp.Stats.SwitchCD {
			k.LastAction = time.Now()
			k.LastSpells[selectedSpell] = k.LastAction
			k.clickDown = false
			k.lastSpellKey = &NoInput
			k.g.SelectedSpell = attack.SpellNone
			return true, k.g.mouseX, k.g.mouseY
		}
		k.clickDown = false
	}
	return false, 0, 0
}

func (k *Keys) PressedPotion() item.Item {
	if k.keysLocked {
		return item.None
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
	if p != item.None && time.Since(k.lastPotion) > k.PotionCooldown {
		k.lastPotion = time.Now()
		return p
	}

	return item.None
}

func (k *Keys) DrawChat2(screen *ebiten.Image, offset ebiten.GeoM) {

	var chatMsg *ebiten.Image
	var off int
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Concat(offset)
	op.GeoM.Translate(k.g.player.Pos[0]+16, k.g.player.Pos[1]-40)

	if k.enterDown {
		//op := &ebiten.DrawImageOptions{}
		//op.GeoM.Translate(float64(x-3), float64(y))
		screen.DrawImage(k.openCloseImg, op)
		return
	}

	// Blink the cursor.
	if k.chatInputOpen {
		t := k.typer.Text
		off = len(t) * 3
		if k.g.counter%60 < 30 {
			t += "_"
			chatMsg = text.PrintImgCol(t, color.RGBA{176, 82, 51, 0})
		} else if len(t) > 0 {
			chatMsg = text.PrintImgCol(t, color.RGBA{176, 82, 51, 0})
		}
	} else if k.sentChat != nil {
		op.GeoM.Translate(-float64(k.sentChatOff), 0)
		screen.DrawImage(k.sentChat, op)
		return
	}

	if chatMsg != nil {

		op.GeoM.Translate(-float64(off), 0)
		screen.DrawImage(chatMsg, op)
	}
}
func (k *Keys) ChatMessage() string {
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if !k.enterDown {
			k.enterDown = true
			if k.chatInputOpen && k.typer.Text != "" {
				k.lastChat = time.Now()
				msg := strings.Trim(k.typer.Text, "\n")
				k.sentChatOff = len(msg) * 3
				k.sentChat = text.PrintImg(msg)
				k.typer.Text = ""
				k.chatInputOpen = !k.chatInputOpen
				k.keysLocked = !k.keysLocked
				return msg
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
		k.sentChat = nil
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

func (i *Input) ToMsgs() msgs.Input {
	return msgs.Input{
		Mouse:    int16(i.Mouse),
		Keyboard: int16(i.Keyboard),
	}
}
func (i *Input) FromMsgs(m msgs.Input) {
	i.Mouse = ebiten.MouseButton(m.Mouse)
	i.Keyboard = ebiten.Key(m.Keyboard)
}
func InputFromMsgs(m msgs.Input) *Input {
	i := &Input{}
	i.Mouse = ebiten.MouseButton(m.Mouse)
	i.Keyboard = ebiten.Key(m.Keyboard)
	return i
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
