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

	meleeImg *ebiten.Image

	keyBinders []*KeyBinder[*Input]
}

func NewHud(g *Game) *Hud {
	s := &Hud{
		g:                  g,
		lastSwitch:         time.Now(),
		modeSwitchCooldown: time.Millisecond * 700,
		barOffsetStart:     32,
		barOffsetEnd:       6,

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
	}
	s.x = 0
	s.y = float64(ScreenHeight - s.hudBg.Bounds().Dy())

	cooldownBarImg := texture.Decode(img.CooldownBase_png)

	redPotionKeyBinder := KeyBinderOpt[*Input, msgs.Item, bool]{
		Desc:       "+30 Health",
		Rect:       image.Rect(304, int(s.y), 336, ScreenHeight-32),
		Active:     g.keys.cfg.PotionHP,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, redPotionKeyBinder)

	bluePotionKeyBinder := KeyBinderOpt[*Input, msgs.Item, bool]{
		Desc:       "+5% Mana",
		Rect:       image.Rect(304, int(s.y+32), 336, ScreenHeight),
		Active:     g.keys.cfg.PotionMP,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, bluePotionKeyBinder)

	spellbarOffset := 300
	iconX := ScreenWidth - spellbarOffset
	resurrectSpellKeyBinder := KeyBinderOpt[*Input, spell.Spell, bool]{
		Desc:       "Resurrects a target",
		IconImg:    texture.Decode(img.IconResurrect_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickResurrect,
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
		Desc:    "Heal. Mana cost: 400",
		IconImg: texture.Decode(img.IconHeal_png),

		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickHealWounds,
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
		Desc:       "Removes the paralize of a target",
		IconImg:    texture.Decode(img.IconRmParalize_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickParalizeRm,
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
		Desc:       "Paralize a target",
		IconImg:    texture.Decode(img.IconParalize_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickParalize,
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
		Desc:       "Damage with an electric discharge",
		IconImg:    texture.Decode(img.IconElectricDischarge_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickElectricDischarge,
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
		Desc:       "Damage with an explosion",
		IconImg:    texture.Decode(img.IconExplode_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickExplode,
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

	meleeX := ScreenWidth - spellbarOffset - 60
	meleeKeyBinder := KeyBinderOpt[*Input, struct{}, bool]{
		Desc:       "Melee hit",
		IconImg:    texture.Decode(img.IconMelee_png),
		Rect:       image.Rect(meleeX, int(s.y), meleeX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.Melee,
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

}

func (s *Hud) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x, s.y)
	screen.DrawImage(s.hudBg, op)
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
}

const (
	SpellIconWidth = 50
)

func (s *Hud) ShowSpellPicker(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	spellsX, spellsY := float64(ScreenWidth-300), s.y
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

func (kb *KeyBinder[A]) GetInput() {
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

func (kb *KeyBinder[A]) Update() {
	kb.CooldownInfo.UpdateImage()
	kb.Mouse()
	if kb.Open {
		kb.GetInput()
	}
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
	if kb.Over && !kb.Open {
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		op.ColorScale.ScaleAlpha(0.3)
		screen.DrawImage(kb.Img, op)
		text.PrintAtBg(screen, kb.Active.String(), x+16, y-48)
		text.PrintAt(screen, kb.Desc, x+15, y-30)
	}
	if !kb.Open {
		return
	}
	op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
	screen.DrawImage(kb.Img, op)
	if !kb.Selected.Empty() {
		text.PrintBigAtBg(screen, kb.Selected.String(), x+18, y-63)
	} else {
		str := " "
		if kb.counter%60 < 30 {
			str = "_"
		}
		text.PrintBigAtBg(screen, str, x+18, y-63)
	}
	text.PrintBigAt(screen, kb.Desc, x+15, y-30)
}

func Replace[A comparable, B any, T map[A]B](m T, new, old A) {
	if m == nil {
		return
	}
	m[new] = m[old]
	delete(m, old)
}
