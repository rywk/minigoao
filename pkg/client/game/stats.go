package game

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/typ"
)

type Checkbox struct {
	Pos           typ.P
	W, H          int32
	LastPressed   time.Time
	Cooldown      time.Duration
	ImgOn, ImgOff *ebiten.Image
	On            bool
}

func NewCheckbox(pos typ.P, on, off *ebiten.Image) *Checkbox {
	return &Checkbox{
		Pos:         pos,
		W:           int32(on.Bounds().Dx()),
		H:           int32(on.Bounds().Dy()),
		LastPressed: time.Now(),
		Cooldown:    time.Millisecond * 700,
		ImgOn:       on,
		ImgOff:      off,
		On:          false,
	}
}

func (b *Checkbox) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(b.Pos.X), float64(b.Pos.Y))
	if b.On {
		screen.DrawImage(b.ImgOn, op)
	} else {
		screen.DrawImage(b.ImgOff, op)
	}
}

func (b *Checkbox) Update() {
	cx, cy := ebiten.CursorPosition()
	if cx > int(b.Pos.X) && cx < int(b.Pos.X+b.W) && cy > int(b.Pos.Y) && cy < int(b.Pos.Y+b.H) {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) && time.Since(b.LastPressed) > b.Cooldown {
			b.LastPressed = time.Now()
			b.On = !b.On
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

	spellIconImgs    *ebiten.Image
	selectedSpellImg *ebiten.Image

	bluePotionImg, redPotionImg *ebiten.Image

	potionSignal          time.Duration
	manaPotionSignalImg   *ebiten.Image
	healthPotionSignalImg *ebiten.Image
	potionAlpha           uint8
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

		hudBg:            texture.Decode(img.HudBg_png),
		spellIconImgs:    texture.Decode(img.SpellbarIcons2_png),
		selectedSpellImg: texture.Decode(img.SpellSelector_png),
		bluePotionImg:    texture.Decode(img.BluePotion_png),
		redPotionImg:     texture.Decode(img.RedPotion_png),

		potionSignal:          time.Millisecond * 300,
		manaPotionSignalImg:   ebiten.NewImage(32, 32),
		healthPotionSignalImg: ebiten.NewImage(32, 32),
		potionAlpha:           0,
	}
	s.x = 0
	s.y = float64(ScreenHeight - s.hudBg.Bounds().Dy())
	return s
}

func RedAlpha(a uint8) color.Color {
	return color.RGBA{174, 0, 18, a}
}
func BlueAlpha(a uint8) color.Color {
	return color.RGBA{0, 18, 174, a}
}

func (s *Hud) Update() {

	if s.potionAlpha > 0 {
		s.potionAlpha -= 2
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
	// if s.potionAlpha != 0 {
	// 	if s.g.lastPotionUsed == msgs.Item(potion.Blue) {
	// 		s.manaPotionSignalImg.Clear()
	// 		s.manaPotionSignalImg.Fill(BlueAlpha(uint8(s.potionAlpha)))
	// 		op := &ebiten.DrawImageOptions{}
	// 		op.GeoM.Translate(s.x+304, s.y+32)
	// 		screen.DrawImage(s.manaPotionSignalImg, op)
	// 	} else if s.g.lastPotionUsed == msgs.Item(potion.Red) {
	// 		s.healthPotionSignalImg.Clear()
	// 		s.healthPotionSignalImg.Fill()
	// 		s.healthPotionSignalImg.Fill(RedAlpha(uint8(s.potionAlpha)))
	// 		op := &ebiten.DrawImageOptions{}
	// 		op.GeoM.Translate(s.x+304, s.y)
	// 		screen.DrawImage(s.healthPotionSignalImg, op)
	// 	}
	// }
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x+304, s.y)
	screen.DrawImage(s.redPotionImg, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x+304, s.y+32)
	screen.DrawImage(s.bluePotionImg, op)

}

func (s *Hud) ShowSpellPicker(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	spellsX, spellsY := float64(ScreenWidth-340), s.y
	op.GeoM.Translate(spellsX, spellsY)
	spellsX -= 10
	opselect := &ebiten.DrawImageOptions{}
	switch s.g.combatKeys.spell {
	case spell.Resurrect:
		opselect.GeoM.Translate(float64(spellsX+22), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.HealWounds:
		opselect.GeoM.Translate(float64(spellsX+77), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.RemoveParalize:
		opselect.GeoM.Translate(float64(spellsX+132), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.Paralize:
		opselect.GeoM.Translate(float64(spellsX+184), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.ElectricDischarge:
		opselect.GeoM.Translate(float64(spellsX+234), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case spell.Explode:
		opselect.GeoM.Translate(float64(spellsX+288), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	}
	screen.DrawImage(s.spellIconImgs, op)
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
