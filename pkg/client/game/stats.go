package game

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/texture"
)

type Stats struct {
	modeBig      bool
	x, y         float64
	moving       bool
	moveX, moveY float64
	g            *Game

	modeSwitchCooldown time.Duration
	lastSwitch         time.Time

	bigBarOffsetStart, bigBarOffsetEnd int
	bigStatsPlaceholder                *ebiten.Image
	bigHpBar, bigMpBar                 *ebiten.Image
	bigHpBarRect, bigMpBarRect         image.Rectangle

	miniBarOffsetStart, miniBarOffsetEnd int
	miniStatsPlaceholder                 *ebiten.Image
	miniHpBar, miniMpBar                 *ebiten.Image
	miniHpBarRect, miniMpBarRect         image.Rectangle
}

func NewStats(g *Game, x, y float64) *Stats {
	return &Stats{
		modeBig:             true,
		modeSwitchCooldown:  time.Millisecond * 700,
		lastSwitch:          time.Now(),
		x:                   x,
		y:                   y,
		g:                   g,
		bigBarOffsetStart:   32,
		bigBarOffsetEnd:     6,
		bigStatsPlaceholder: texture.Decode(img.PlaceholderStats_png),
		bigHpBar:            texture.Decode(img.BigHPBar_png),
		bigMpBar:            texture.Decode(img.BigMPBar_png),

		miniBarOffsetStart:   14,
		miniBarOffsetEnd:     4,
		miniStatsPlaceholder: texture.Decode(img.MiniPlaceholderStats_png),
		miniHpBar:            texture.Decode(img.MiniHPBar_png),
		miniMpBar:            texture.Decode(img.MiniMPBar_png),
	}
}

func (s *Stats) Update() {
	s.Move()
	if s.modeBig {
		hp := mapValue(float64(s.g.client.HP), 0, float64(s.g.client.MaxHP), float64(s.bigBarOffsetStart), float64(s.bigHpBar.Bounds().Max.X-s.bigBarOffsetEnd))
		s.bigHpBarRect = image.Rect(s.bigHpBar.Bounds().Min.X, s.bigHpBar.Bounds().Min.Y, int(hp), s.bigHpBar.Bounds().Max.Y)

		mp := mapValue(float64(s.g.client.MP), 0, float64(s.g.client.MaxMP), float64(s.bigBarOffsetStart), float64(s.bigMpBar.Bounds().Max.X-s.bigBarOffsetEnd))
		s.bigMpBarRect = image.Rect(s.bigMpBar.Bounds().Min.X, s.bigMpBar.Bounds().Min.Y, int(mp), s.bigMpBar.Bounds().Max.Y)
	} else {
		hp := mapValue(float64(s.g.client.HP), 0, float64(s.g.client.MaxHP), float64(s.miniBarOffsetStart), float64(s.miniHpBar.Bounds().Max.X-s.miniBarOffsetEnd))
		s.miniHpBarRect = image.Rect(s.miniHpBar.Bounds().Min.X, s.miniHpBar.Bounds().Min.Y, int(hp), s.miniHpBar.Bounds().Max.Y)

		mp := mapValue(float64(s.g.client.MP), 0, float64(s.g.client.MaxMP), float64(s.miniBarOffsetStart), float64(s.miniMpBar.Bounds().Max.X-s.miniBarOffsetEnd))
		s.miniMpBarRect = image.Rect(s.miniMpBar.Bounds().Min.X, s.miniMpBar.Bounds().Min.Y, int(mp), s.miniMpBar.Bounds().Max.Y)
	}
}

func (s *Stats) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	if s.modeBig {
		op.GeoM.Translate(s.x, s.y)
		screen.DrawImage(s.bigStatsPlaceholder, op)
		screen.DrawImage(s.bigHpBar.SubImage(s.bigHpBarRect).(*ebiten.Image), op)
		screen.DrawImage(s.bigMpBar.SubImage(s.bigMpBarRect).(*ebiten.Image), op)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.client.HP), int(s.x)+250, int(s.y)+13)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.client.MP), int(s.x)+250, int(s.y)+50)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.player.X), int(s.x)+6, int(s.y)+24)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.player.Y), int(s.x)+6, int(s.y)+44)
	} else {
		op.GeoM.Translate(s.x, s.y)
		screen.DrawImage(s.miniStatsPlaceholder, op)
		screen.DrawImage(s.miniHpBar.SubImage(s.miniHpBarRect).(*ebiten.Image), op)
		screen.DrawImage(s.miniMpBar.SubImage(s.miniMpBarRect).(*ebiten.Image), op)
	}
}

func (s *Stats) Move() {
	cx, cy := ebiten.CursorPosition()
	if cx > int(s.x)+2 && cx < int(s.x)+10 && cy > int(s.y)+2 && cy < int(s.y)+10 {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) && time.Since(s.lastSwitch) > s.modeSwitchCooldown {
			s.modeBig = !s.modeBig
			s.lastSwitch = time.Now()
		}
	}
	var rect image.Rectangle
	if s.modeBig {
		rect = s.bigStatsPlaceholder.Bounds()
	} else {
		rect = s.miniStatsPlaceholder.Bounds()
	}
	if cx > int(s.x)+rect.Min.X && cx < int(s.x)+rect.Max.X && cy > int(s.y)+rect.Min.Y && cy < int(s.y)+rect.Max.Y || s.moving {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !s.moving {
				s.moveX = float64(cx) - s.x
				s.moveY = float64(cy) - s.y
			}
			s.moving = true
			s.x, s.y = float64(cx)-s.moveX, float64(cy)-s.moveY
		} else {
			s.moving = false
		}
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
