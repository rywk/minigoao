package game

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
)

type Checkbox struct {
	g             *Game
	Pos           typ.P
	W, H          int32
	ImgOn, ImgOff *ebiten.Image
	On            bool
	Pressed       bool
}

func NewCheckbox(g *Game) *Checkbox {
	on, off := texture.Decode(img.CheckboxOn_png), texture.Decode(img.CheckboxOff_png)
	return &Checkbox{
		g:      g,
		W:      int32(on.Bounds().Dx()),
		H:      int32(on.Bounds().Dy()),
		ImgOn:  on,
		ImgOff: off,
		On:     false,
	}
}
func (b *Checkbox) Draw(screen *ebiten.Image, x, y int) {
	b.Pos = typ.P{X: int32(x), Y: int32(y)}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(b.Pos.X), float64(b.Pos.Y))
	if b.On {
		screen.DrawImage(b.ImgOn, op)
	} else {
		screen.DrawImage(b.ImgOff, op)
	}
}

func (b *Checkbox) Update() {
	cx, cy := b.g.mouseX, b.g.mouseY
	if cx > int(b.Pos.X) && cx < int(b.Pos.X+b.W) && cy > int(b.Pos.Y) && cy < int(b.Pos.Y+b.H) {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !b.Pressed {
				b.On = !b.On
			}
			b.Pressed = true
		} else {
			b.Pressed = false
		}
	}
}

type Button struct {
	g       *Game
	Pos     typ.P
	W, H    int32
	Img     *ebiten.Image
	Icon    *ebiten.Image
	Over    bool
	pressed bool
}

func NewButton(g *Game, i *ebiten.Image) *Button {
	return &Button{
		g:    g,
		W:    int32(i.Bounds().Dx()),
		H:    int32(i.Bounds().Dy()),
		Img:  i,
		Icon: texture.Decode(img.ConfigIcon_png),
	}
}
func (b *Button) Draw(screen *ebiten.Image, x, y int) {
	b.Pos = typ.P{X: int32(x), Y: int32(y)}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(b.Pos.X), float64(b.Pos.Y))
	if b.Over {
		op.ColorScale.ScaleAlpha(.6)
	}
	screen.DrawImage(b.Img, op)
	screen.DrawImage(b.Icon, op)

}

func (b *Button) Pressed() bool {
	cx, cy := b.g.mouseX, b.g.mouseY
	v := false
	if cx > int(b.Pos.X) && cx < int(b.Pos.X+b.W) && cy > int(b.Pos.Y) && cy < int(b.Pos.Y+b.H) {
		b.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !b.pressed {
				v = true
			}
			b.pressed = true
		} else {
			b.pressed = false
		}
	} else {
		b.Over = false
	}
	return v
}

type Slider struct {
	g   *Game
	Pos image.Point

	W, H      int32
	Knob      *ebiten.Image
	Line      *ebiten.Image
	drawOp    *ebiten.DrawImageOptions
	Over      bool
	on        bool
	pressed   bool
	SliderPos image.Point

	value int
}

func NewSlider(g *Game) *Slider {
	volKnob := ebiten.NewImage(10, 22)
	s := &Slider{
		drawOp: &ebiten.DrawImageOptions{},
		g:      g,
		W:      int32(volKnob.Bounds().Dx()),
		H:      int32(volKnob.Bounds().Dy()),
		Knob:   volKnob,
		Line:   ebiten.NewImage(200, 4),
		value:  160,
	}
	s.Pos.X = int(200 - s.value)
	s.g.SoundBoard.SetVolume(float64(s.value))

	s.Knob.Fill(color.White)
	s.Line.Fill(color.White)
	return s
}
func (b *Slider) Draw(screen *ebiten.Image, x, y int) {
	b.SliderPos = image.Pt(x, y)
	b.drawOp.ColorScale.Reset()
	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(float64(b.SliderPos.X), float64(b.SliderPos.Y)+9)
	screen.DrawImage(b.Line, b.drawOp)
	if b.Over || b.on {
		b.drawOp.ColorScale.Reset()
		b.drawOp.ColorScale.ScaleAlpha(.6)
	} else {
		b.drawOp.ColorScale.Reset()
		b.drawOp.ColorScale.ScaleAlpha(1)
	}
	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(float64(x+int(b.Pos.X)), float64(y+int(b.Pos.Y)))
	screen.DrawImage(b.Knob, b.drawOp)

}
func (b *Slider) Update() {
	cx, cy := b.g.mouseX, b.g.mouseY
	knobRect := b.Knob.Bounds().
		Add(b.Pos).
		Add(b.SliderPos).
		Add(image.Pt(ScreenWidth-300, ScreenHeight-664))

	sliderRect := b.Line.Bounds().
		Add(b.SliderPos).
		Add(image.Pt(ScreenWidth-300, ScreenHeight-664))
	//	log.Printf("pos : %v ", knobRect)

	if cx > int(knobRect.Min.X) && cx < int(knobRect.Max.X) && cy > int(knobRect.Min.Y) && cy < int(knobRect.Max.Y) {
		b.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !b.pressed {
				b.on = true
			}
			b.pressed = true
		} else {
			b.pressed = false
		}
	} else {
		b.Over = false
	}
	if b.on {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			b.on = false
			b.pressed = false
		}
		b.value = int(mapValue(float64(cx), float64(sliderRect.Min.X), float64(sliderRect.Max.X), 200, 0))
		b.Pos.X = int(200 - b.value)
		b.g.SoundBoard.SetVolume(float64(b.value))
	}
}

type Options struct {
	g          *Game
	drawOp     *ebiten.DrawImageOptions
	W, H       int32
	Background *ebiten.Image

	volumeSlider *Slider

	keyBinders []*NKeyBinder[*Input]
}

func NewOptions(g *Game) *Options {
	bg := ebiten.NewImage(300, 600)
	s := &Options{
		drawOp:       &ebiten.DrawImageOptions{},
		g:            g,
		W:            int32(bg.Bounds().Dx()),
		H:            int32(bg.Bounds().Dy()),
		volumeSlider: NewSlider(g),
		Background:   bg,
	}

	keyBindXStart := ScreenWidth - 210
	keyBindWidth := 200

	start := 150
	height := 38
	//meleeX := actionsBarStart
	meleeKeyBinder := NKeyBinderOpt[*Input, struct{}, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start, keyBindXStart+keyBindWidth, start+height),
		Active:     g.keys.cfg.Melee,
		actionMap:  map[*Input]struct{}{},
		pressedMap: map[*Input]bool{},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, meleeKeyBinder)

	resuKeyBinder := NKeyBinderOpt[*Input, spell.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height, keyBindXStart+keyBindWidth, start+height*2),
		Active:     g.keys.cfg.PickResurrect,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, resuKeyBinder)

	healKeyBinder := NKeyBinderOpt[*Input, spell.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*2, keyBindXStart+keyBindWidth, start+height*3),
		Active:     g.keys.cfg.PickHealWounds,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, healKeyBinder)

	remoKeyBinder := NKeyBinderOpt[*Input, spell.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*3, keyBindXStart+keyBindWidth, start+height*4),
		Active:     g.keys.cfg.PickParalizeRm,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, remoKeyBinder)

	paraKeyBinder := NKeyBinderOpt[*Input, spell.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*4, keyBindXStart+keyBindWidth, start+height*5),
		Active:     g.keys.cfg.PickParalize,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, paraKeyBinder)

	descaKeyBinder := NKeyBinderOpt[*Input, spell.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*5, keyBindXStart+keyBindWidth, start+height*6),
		Active:     g.keys.cfg.PickElectricDischarge,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, descaKeyBinder)

	apocaKeyBinder := NKeyBinderOpt[*Input, spell.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*6, keyBindXStart+keyBindWidth, start+height*7),
		Active:     g.keys.cfg.PickExplode,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, apocaKeyBinder)

	upKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*7, keyBindXStart+keyBindWidth, start+height*8),
		Active:     g.keys.cfg.Back,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, upKeyBinder)

	leftKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*8, keyBindXStart+keyBindWidth, start+height*9),
		Active:     g.keys.cfg.Left,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, leftKeyBinder)

	downKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*9, keyBindXStart+keyBindWidth, start+height*10),
		Active:     g.keys.cfg.Front,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, downKeyBinder)

	rightKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*10, keyBindXStart+keyBindWidth, start+height*11),
		Active:     g.keys.cfg.Right,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, rightKeyBinder)

	redsKeyBinder := NKeyBinderOpt[*Input, msgs.Item, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*11, keyBindXStart+keyBindWidth, start+height*12),
		Active:     g.keys.cfg.PotionHP,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, redsKeyBinder)

	bluesKeyBinder := NKeyBinderOpt[*Input, msgs.Item, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*12, keyBindXStart+keyBindWidth, start+height*13),
		Active:     g.keys.cfg.PotionMP,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, bluesKeyBinder)

	return s
}

func (b *Options) Update() {
	for _, kb := range b.keyBinders {
		kb.Update()
	}
	b.volumeSlider.Update()

}
func (b *Options) Draw(screen *ebiten.Image) {
	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(ScreenWidth-300, ScreenHeight-664)
	b.Background.Fill(color.Black)

	start := 18
	height := 38
	text.PrintBigAt(b.Background, "Vol", 18, start-2)
	text.PrintBigAt(b.Background, "Keys", 18, start+height)
	keyX := 24
	text.PrintBigAt(b.Background, "Piña", keyX, start+height*2)
	text.PrintBigAt(b.Background, "Resu", keyX, start+height*3)
	text.PrintBigAt(b.Background, "Cura", keyX, start+height*4)
	text.PrintBigAt(b.Background, "Remo", keyX, start+height*5)
	text.PrintBigAt(b.Background, "Para", keyX, start+height*6)
	text.PrintBigAt(b.Background, "Desca", keyX, start+height*7)
	text.PrintBigAt(b.Background, "Apoca", keyX, start+height*8)
	text.PrintBigAt(b.Background, "Up", keyX, start+height*9)
	text.PrintBigAt(b.Background, "Left", keyX, start+height*10)
	text.PrintBigAt(b.Background, "Down", keyX, start+height*11)
	text.PrintBigAt(b.Background, "Right", keyX, start+height*12)
	text.PrintBigAt(b.Background, "Rojas", keyX, start+height*13)
	text.PrintBigAt(b.Background, "Azules", keyX, start+height*14)

	inputX := keyX + 40
	b.volumeSlider.Draw(b.Background, inputX+8, start)

	screen.DrawImage(b.Background, b.drawOp)
	for _, kb := range b.keyBinders {
		kb.Draw(screen)
	}
}

type Hud struct {
	x, y float64
	g    *Game

	modeSwitchCooldown time.Duration
	lastSwitch         time.Time

	barOffsetStart, barOffsetEnd int
	hpBar, mpBar                 *ebiten.Image
	hpBarRect, mpBarRect         image.Rectangle

	hudBg *ebiten.Image

	//spellIconImgs    *ebiten.Image
	selectedSpellImg *ebiten.Image

	bluePotionImg, redPotionImg *ebiten.Image

	manaPotionSignalImg   *ebiten.Image
	healthPotionSignalImg *ebiten.Image
	potionAlpha           float64

	optionsOpen   bool
	options       *Options
	optionsButton *Button

	keyBinders []*KeyBinder[*Input]

	lastHudPotion     time.Time
	hudPotionCooldown time.Duration
}

func NewHud(g *Game) *Hud {
	btnImg := ebiten.NewImage(32, 32)
	btnImg.Fill(color.RGBA{101, 32, 133, 0})
	s := &Hud{
		g:                  g,
		lastSwitch:         time.Now(),
		modeSwitchCooldown: time.Millisecond * 700,
		barOffsetStart:     32,
		barOffsetEnd:       6,

		optionsOpen:   false,
		optionsButton: NewButton(g, btnImg),
		options:       NewOptions(g),

		hpBar: texture.Decode(img.HpBar_png),
		mpBar: texture.Decode(img.MpBar_png),

		hudBg: texture.Decode(img.HudBg_png),
		//spellIconImgs:    texture.Decode(img.SpellbarIcons2_png),
		selectedSpellImg: texture.Decode(img.SpellSelector_png),
		bluePotionImg:    texture.Decode(img.BluePotion_png),
		redPotionImg:     texture.Decode(img.RedPotion_png),

		manaPotionSignalImg:   ebiten.NewImage(32, 32),
		healthPotionSignalImg: ebiten.NewImage(32, 32),
		potionAlpha:           0,

		lastHudPotion:     time.Now(),
		hudPotionCooldown: time.Millisecond * 250,
	}
	s.x = 0
	s.y = float64(ScreenHeight - s.hudBg.Bounds().Dy())

	cooldownBarImg := texture.Decode(img.CooldownBase_png)

	redPotionKeyBinder := KeyBinderOpt[*Input, msgs.Item, bool]{
		Desc:       "+30 Health",
		Rect:       image.Rect(304, int(s.y), 336, ScreenHeight-32),
		Active:     g.keys.cfg.PotionHP,
		Item:       msgs.ItemHealthPotion,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, redPotionKeyBinder)

	bluePotionKeyBinder := KeyBinderOpt[*Input, msgs.Item, bool]{
		Desc:   "+5% Mana",
		Rect:   image.Rect(304, int(s.y+32), 336, ScreenHeight),
		Active: g.keys.cfg.PotionMP,
		Item:   msgs.ItemManaPotion,

		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, bluePotionKeyBinder)

	actionsBarStart := 450

	meleeX := actionsBarStart
	meleeKeyBinder := KeyBinderOpt[*Input, struct{}, bool]{
		Desc:       "Piña",
		IconImg:    texture.Decode(img.IconMelee_png),
		Rect:       image.Rect(meleeX, int(s.y), meleeX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.Melee,
		Spell:      spell.None,
		actionMap:  map[*Input]struct{}{},
		pressedMap: map[*Input]bool{},
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownMelee,
			Last:       &g.keys.LastMelee,
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, meleeKeyBinder)

	iconX := actionsBarStart + SpellIconWidth + 10
	resurrectSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:       "Resu",
		IconImg:    texture.Decode(img.IconResurrect_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickResurrect,
		Spell:      spell.Resurrect,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownSpells[spell.Resurrect],
			Last:       &g.keys.LastSpells[spell.Resurrect],
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, resurrectSpellKeyBinder)

	iconX += SpellIconWidth
	healSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:    "Cura",
		IconImg: texture.Decode(img.IconHeal_png),

		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickHealWounds,
		Spell:      spell.HealWounds,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownSpells[spell.HealWounds],
			Last:       &g.keys.LastSpells[spell.HealWounds],
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, healSpellKeyBinder)

	iconX += SpellIconWidth
	rmParalizeSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:       "Remo",
		IconImg:    texture.Decode(img.IconRmParalize_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickParalizeRm,
		Spell:      spell.RemoveParalize,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownSpells[spell.RemoveParalize],
			Last:       &g.keys.LastSpells[spell.RemoveParalize],
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, rmParalizeSpellKeyBinder)

	iconX += SpellIconWidth
	paralizeSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:       "Para",
		IconImg:    texture.Decode(img.IconParalize_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickParalize,
		Spell:      spell.Paralize,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownSpells[spell.Paralize],
			Last:       &g.keys.LastSpells[spell.Paralize],
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, paralizeSpellKeyBinder)

	iconX += SpellIconWidth
	electricDischargeSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:       "Desca",
		IconImg:    texture.Decode(img.IconElectricDischarge_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickElectricDischarge,
		Spell:      spell.ElectricDischarge,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownSpells[spell.ElectricDischarge],
			Last:       &g.keys.LastSpells[spell.ElectricDischarge],
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, electricDischargeSpellKeyBinder)

	iconX += SpellIconWidth
	explodeSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:       "Apoca",
		IconImg:    texture.Decode(img.IconExplode_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickExplode,
		Spell:      spell.Explode,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			BaseImg:    cooldownBarImg,
			CD:         g.keys.cfg.CooldownSpells[spell.Explode],
			Last:       &g.keys.LastSpells[spell.Explode],
			GlobalCD:   g.keys.cfg.CooldownAction,
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, explodeSpellKeyBinder)

	for _, kb := range s.keyBinders {
		kb.g = g
	}

	s.healthPotionSignalImg.Fill(RedAlpha(uint8(60)))
	s.manaPotionSignalImg.Fill(BlueAlpha(uint8(90)))

	return s
}

func RedAlpha(a uint8) color.Color {
	return color.RGBA{168, 0, 16, a}
}
func BlueAlpha(a uint8) color.Color {
	return color.RGBA{0, 18, 174, a}
}

func (s *Hud) Update() {
	for _, kb := range s.keyBinders {
		kb.Update()
	}

	if s.potionAlpha > 0 {
		s.potionAlpha -= .04
	}

	hp := mapValue(float64(s.g.client.HP), 0, float64(s.g.client.MaxHP), float64(s.barOffsetStart), float64(s.hpBar.Bounds().Max.X-s.barOffsetEnd))
	s.hpBarRect = image.Rect(s.hpBar.Bounds().Min.X, s.hpBar.Bounds().Min.Y, int(hp), s.hpBar.Bounds().Max.Y)

	mp := mapValue(float64(s.g.client.MP), 0, float64(s.g.client.MaxMP), float64(s.barOffsetStart), float64(s.mpBar.Bounds().Max.X-s.barOffsetEnd))
	s.mpBarRect = image.Rect(s.mpBar.Bounds().Min.X, s.mpBar.Bounds().Min.Y, int(mp), s.mpBar.Bounds().Max.Y)

	if s.optionsButton.Pressed() {
		s.optionsOpen = !s.optionsOpen
	}
	if s.optionsOpen {
		s.options.Update()
	}
}

func (s *Hud) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x, s.y)
	screen.DrawImage(s.hudBg, op)
	s.optionsButton.Draw(screen, ScreenWidth-64, int(s.y)+16)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x, s.y)
	screen.DrawImage(s.hpBar.SubImage(s.hpBarRect).(*ebiten.Image), op)
	screen.DrawImage(s.mpBar.SubImage(s.mpBarRect).(*ebiten.Image), op)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.client.HP), int(s.x)+250, int(s.y)+10)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.client.MP), int(s.x)+250, int(s.y)+38)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.player.X), int(s.x)+7, int(s.y)+12)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.player.Y), int(s.x)+7, int(s.y)+35)
	s.ShowSpellPicker(screen)
	if s.potionAlpha > 0 {
		if s.g.lastPotionUsed == msgs.ItemManaPotion {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(s.x+304, s.y+32)
			op.ColorScale.ScaleAlpha(float32(s.potionAlpha))
			screen.DrawImage(s.manaPotionSignalImg, op)
		} else if s.g.lastPotionUsed == msgs.ItemHealthPotion {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(s.x+304, s.y)
			op.ColorScale.ScaleAlpha(float32(s.potionAlpha))
			screen.DrawImage(s.healthPotionSignalImg, op)
		}
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x+304, s.y+1)
	screen.DrawImage(s.redPotionImg, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x+304, s.y+31)
	screen.DrawImage(s.bluePotionImg, op)

	for _, kb := range s.keyBinders {
		kb.Draw(screen)
	}
	for _, kb := range s.keyBinders {
		kb.DrawTooltips(screen, s.g.mouseX, s.g.mouseY)
	}
	if s.optionsOpen {
		s.options.Draw(screen)
	}
}

const (
	SpellIconWidth = 50
)

func (s *Hud) ShowSpellPicker(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	spellsX, spellsY := float64(510), s.y
	op.GeoM.Translate(spellsX, spellsY)
	opselect := &ebiten.DrawImageOptions{}
	switch s.g.keys.spell {
	case spell.Resurrect:
		opselect.GeoM.Translate(float64(spellsX), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.HealWounds:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.RemoveParalize:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*2), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.Paralize:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*3), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.ElectricDischarge:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*4), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.Explode:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*5), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	}
}

func mapValue(v, start1, stop1, start2, stop2 float64) float64 {
	newval := (v-start1)/(stop1-start1)*(stop2-start2) + start2
	if start2 < stop2 {
		if newval > stop2 {
			newval = stop2
		} else if newval < start2 {
			newval = start2
		}
	} else {
		if newval > start2 {
			newval = start2
		} else if newval < stop2 {
			newval = stop2
		}
	}
	return newval
}

type KeyBinder[A KBind] struct {
	g            *Game
	counter      int
	Img          *ebiten.Image
	IconImg      *ebiten.Image
	CooldownInfo *Cooldown
	Over         bool
	Rect         image.Rectangle
	Active       A
	Spell        spell.Spell
	Item         msgs.Item
	Desc         string
	Change       func(old, new A)
	Open         bool
	Clicked      bool
	Selected     A
}

type Cooldown struct {
	BaseImg    *ebiten.Image
	Img        *ebiten.Image
	CD         time.Duration
	Last       *time.Time
	GlobalCD   time.Duration
	GlobalLast *time.Time
}

const (
	iconX = 50
	iconY = 64
)

func (cd *Cooldown) UpdateImage() {
	if cd == nil {
		return
	}
	now := time.Now()
	dt := now.Sub(*cd.Last)
	v := mapValue(float64(dt), float64(cd.CD), 0, 0, iconY)
	gdt := now.Sub(*cd.GlobalLast)
	gv := mapValue(float64(gdt), float64(cd.GlobalCD), 0, 0, iconY)

	if gv > v {
		if gv < 1 {
			cd.Img = nil
			return
		}
		cd.Img = ebiten.NewImageFromImage(cd.BaseImg.SubImage(image.Rect(0, 0, iconX, int(gv))))
	} else {
		if v < 1 {
			cd.Img = nil
			return
		}
		cd.Img = ebiten.NewImageFromImage(cd.BaseImg.SubImage(image.Rect(0, 0, iconX, int(v))))
	}
}

func (cd *Cooldown) Draw(screen *ebiten.Image, x, y int) {
	if cd == nil {
		return
	}
	if cd.Img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	//ny := iconY - cd.Img.Bounds().Dy()
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(cd.Img, op)
}

type KeyBinderOpt[A KBind, B any, C any] struct {
	Spell        spell.Spell
	Item         msgs.Item
	Desc         string
	CooldownInfo *Cooldown
	IconImg      *ebiten.Image
	Rect         image.Rectangle
	Active       A
	actionMap    map[A]B
	pressedMap   map[A]C
}

func (opt KeyBinderOpt[A, B, C]) NewKeyBinder(selected A) *KeyBinder[A] {
	kb := &KeyBinder[A]{
		Spell:        opt.Spell,
		Item:         opt.Item,
		IconImg:      opt.IconImg,
		Desc:         opt.Desc,
		Active:       opt.Active,
		Selected:     selected,
		Rect:         opt.Rect,
		CooldownInfo: opt.CooldownInfo,
		Change: func(old, new A) {
			Replace(opt.actionMap, new, old)
			Replace(opt.pressedMap, new, old)
		},
		Img: ebiten.NewImage(opt.Rect.Dx(), opt.Rect.Dy()),
	}
	kb.Img.Fill(color.RGBA{60, 60, 60, 200})
	return kb
}

func (kb *KeyBinder[A]) Mouse() {
	x, y := kb.g.mouseX, kb.g.mouseY
	if x > kb.Rect.Min.X && x < kb.Rect.Max.X && y > kb.Rect.Min.Y && y < kb.Rect.Max.Y {
		kb.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !kb.Clicked {
				kb.Clicked = true
				if kb.Spell != spell.None {
					kb.g.keys.spell = kb.Spell
					kb.g.keys.lastSpellKey = kb.Active.VPtr()
				} else if kb.Item != msgs.ItemNone {
					if time.Since(kb.g.stats.lastHudPotion) > kb.g.stats.hudPotionCooldown {
						kb.g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: kb.Item}
						kb.g.stats.lastHudPotion = time.Now()
					}
				}
				kb.Open = !kb.Open
			}
		} else {
			kb.Clicked = false
		}
	} else {
		kb.Over = false
	}
}

func (kb *KeyBinder[A]) Update() {
	if kb.CooldownInfo != nil {
		kb.CooldownInfo.UpdateImage()
	}
	kb.Mouse()
	kb.counter++
}

func (kb *KeyBinder[A]) Draw(screen *ebiten.Image) {
	kb.CooldownInfo.Draw(screen, kb.Rect.Min.X, kb.Rect.Min.Y)
	if kb.IconImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		screen.DrawImage(kb.IconImg, op)
	}
}

func (kb *KeyBinder[A]) DrawTooltips(screen *ebiten.Image, x, y int) {
	op := &ebiten.DrawImageOptions{}
	if x > ScreenWidth-300 {
		x -= 180
	}
	if kb.Over {
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		op.ColorScale.ScaleAlpha(0.3)
		screen.DrawImage(kb.Img, op)
		//text.PrintAtBg(screen, kb.Active.String(), x+16, y-48)
		text.PrintAt(screen, kb.Desc, x+15, y-30)
	}
}

type NKeyBinder[A KBind] struct {
	g        *Game
	counter  int
	Img      *ebiten.Image
	Over     bool
	Rect     image.Rectangle
	Active   A
	Change   func(old, new A)
	Open     bool
	Clicked  bool
	Selected A
}

type NKeyBinderOpt[A KBind, B any, C any] struct {
	g          *Game
	Rect       image.Rectangle
	Active     A
	actionMap  map[A]B
	pressedMap map[A]C
}

func (opt NKeyBinderOpt[A, B, C]) NewKeyBinder(selected A) *NKeyBinder[A] {
	kb := &NKeyBinder[A]{
		g:        opt.g,
		Active:   opt.Active,
		Selected: selected,
		Rect:     opt.Rect,
		Change: func(old, new A) {
			Replace(opt.actionMap, new, old)
			Replace(opt.pressedMap, new, old)
		},
		Img: ebiten.NewImage(opt.Rect.Dx(), opt.Rect.Dy()),
	}
	kb.Img.Fill(color.RGBA{60, 60, 60, 200})
	return kb
}

func (kb *NKeyBinder[A]) Mouse() {
	x, y := kb.g.mouseX, kb.g.mouseY
	if x > kb.Rect.Min.X && x < kb.Rect.Max.X && y > kb.Rect.Min.Y && y < kb.Rect.Max.Y {
		kb.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !kb.Clicked {
				kb.Clicked = true
				if kb.Open {
					if kb.Selected.V() != kb.Active.V() && !kb.Selected.Empty() {
						kb.Active.Set(kb.Selected.V())
					}
					kb.Selected.Set(NoInput)
				}
				kb.Open = !kb.Open
			}
		} else {
			kb.Clicked = false
		}
	} else if kb.Open {
		kb.Over = false
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !kb.Clicked {
				kb.Clicked = true
				if kb.Open {
					if kb.Selected.V() != kb.Active.V() && !kb.Selected.Empty() {
						kb.Active.Set(kb.Selected.V())
					}
					kb.Selected.Set(NoInput)
				}
				kb.Open = !kb.Open
			}
		} else {
			kb.Clicked = false
		}
	} else {
		kb.Over = false
	}
}

func (kb *NKeyBinder[A]) GetInput() {
	input := NoInput
	keys := inpututil.AppendPressedKeys([]ebiten.Key{})
	if len(keys) != 0 {
		for i, key := range keys {
			if key != ebiten.KeyEnter {
				input = NewInput(keys[i])
				break
			}
		}
	}
	rmb := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	if rmb {
		input = NewInput(ebiten.MouseButtonRight)
	}
	mmb := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	if mmb {
		input = NewInput(ebiten.MouseButtonMiddle)
	}
	mb3 := ebiten.IsMouseButtonPressed(ebiten.MouseButton3)
	if mb3 {
		input = NewInput(ebiten.MouseButton3)
	}
	mb4 := ebiten.IsMouseButtonPressed(ebiten.MouseButton4)
	if mb4 {
		input = NewInput(ebiten.MouseButton4)
	}
	if !input.Empty() {
		kb.Selected.Set(input)
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if kb.Selected.V() != kb.Active.V() && !kb.Selected.Empty() {
			kb.Active.Set(kb.Selected.V())
		}
		kb.Open = false
		kb.Selected.Set(NoInput)
	}
}

func (kb *NKeyBinder[A]) Update() {
	kb.Mouse()
	if kb.Open {
		kb.GetInput()
	}
	kb.counter++
}

func (kb *NKeyBinder[A]) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	if kb.Over && !kb.Open {
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		op.ColorScale.ScaleAlpha(0.3)
		screen.DrawImage(kb.Img, op)
	}
	if !kb.Open {
		text.PrintBigAtBg(screen, kb.Active.String(), kb.Rect.Min.X+20, kb.Rect.Min.Y+1)
		return
	}
	op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
	screen.DrawImage(kb.Img, op)
	if !kb.Selected.Empty() {
		text.PrintBigAtBg(screen, kb.Selected.String(), kb.Rect.Min.X+20, kb.Rect.Min.Y+1)
	} else {
		str := " "
		if kb.counter%60 < 30 {
			str = "_"
		}
		text.PrintBigAtBg(screen, str, kb.Rect.Min.X+20, kb.Rect.Min.Y+1)
	}
}

func Replace[A comparable, B any, T map[A]B](m T, new, old A) {
	if m == nil {
		return
	}
	m[new] = m[old]
	delete(m, old)
}
