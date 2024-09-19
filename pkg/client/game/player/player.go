package player

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/conc"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
	"golang.org/x/image/math/f64"
)

// Player
type P struct {
	local  *P
	ID     uint32
	Nick   string
	X, Y   int32
	DX, DY int
	Pos    f64.Vec2

	Dead        bool
	Inmobilized bool

	Armor, Weapon, Helmet, Shield       texture.A
	NakedBody, Head, DeadBody, DeadHead texture.A
	Walking                             bool
	Direction                           direction.D

	ActiveEffects []texture.Effect
	Effect        *PEffects

	Client       *ClientP
	HPImg, MPImg *ebiten.Image
	HPMPBGImg    *ebiten.Image
	MoveSpeed    float64
	leftForMove  float64
	lastDir      direction.D
	steps        []Step

	soundPrevWalk int
	soundboard    *audio2d.SoundBoard
}

type ClientP struct {
	p *P
	// Stats
	HP, MP       int
	MaxHP, MaxMP int
}

func NewClientP() *ClientP {
	return &ClientP{}
}

func (p *ClientP) DirToNewPos(d direction.D) (int, int) {
	switch d {
	case direction.Front:
		return int(p.p.X), int(p.p.Y + 1)
	case direction.Back:
		return int(p.p.X), int(p.p.Y - 1)
	case direction.Left:
		return int(p.p.X - 1), int(p.p.Y)
	case direction.Right:
		return int(p.p.X + 1), int(p.p.Y)
	}
	return 0, 0
}

func (p *P) Nil() bool { return p == nil }

func (p *P) Update(counter int) {
	if p.Dead {
		p.DeadBody.Dir(p.Direction)
		p.DeadHead.Dir(p.Direction)
	} else {
		if p.Armor != nil {
			p.Armor.Dir(p.Direction)
		} else {
			p.NakedBody.Dir(p.Direction)
		}
		p.Head.Dir(p.Direction)
		if p.Helmet != nil {
			p.Helmet.Dir(p.Direction)
		}
		if p.Weapon != nil {
			p.Weapon.Dir(p.Direction)
		}
		if p.Shield != nil {
			p.Shield.Dir(p.Direction)
		}
	}
	p.UpdateFrames(counter)
}

const PlayerDrawOffsetX, PlayerDrawOffsetY = 3, -14
const PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY = 4, -9

func (p *P) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(p.Pos[0]+PlayerDrawOffsetX, p.Pos[1]+PlayerDrawOffsetY)
	if p.Dead {
		op.GeoM.Translate(2, 5)
		screen.DrawImage(p.DeadBody.Frame(), op)
		op.GeoM.Translate(PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY)
		screen.DrawImage(p.DeadHead.Frame(), op)
		p.Effect.Draw(screen)
		return
	}
	if p.Direction == direction.Left || p.Direction == direction.Front {
		if p.Armor != nil {
			screen.DrawImage(p.Armor.Frame(), op)
		} else {
			screen.DrawImage(p.NakedBody.Frame(), op)
		}
		if p.Shield != nil {
			screen.DrawImage(p.Shield.Frame(), op)
		}
	} else {
		if p.Shield != nil {
			screen.DrawImage(p.Shield.Frame(), op)
		}
		if p.Armor != nil {
			screen.DrawImage(p.Armor.Frame(), op)
		} else {
			screen.DrawImage(p.NakedBody.Frame(), op)
		}
	}
	if p.Weapon != nil {
		screen.DrawImage(p.Weapon.Frame(), op)
	}
	op.GeoM.Translate(PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY)
	screen.DrawImage(p.Head.Frame(), op)
	if p.Helmet != nil {
		op.GeoM.Translate(-3, -6)
		screen.DrawImage(p.Helmet.Frame(), op)
	}
	if p.local == nil {
		p.DrawPlayerHPMP(screen)

	} else {
		p.DrawNick(screen)
	}
	p.Effect.Draw(screen)
}

func (p *P) DrawNick(screen *ebiten.Image) {
	tx, ty := int(p.Pos[0]+14), int(p.Pos[1]+26)
	xoff := (len(p.Nick) * 3)
	ebitenutil.DebugPrintAt(screen, p.Nick, tx-xoff, ty)
}

func (p *P) DrawPlayerHPMP(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(p.Pos[0]+PlayerDrawOffsetX, p.Pos[1]+PlayerDrawOffsetY)
	op.GeoM.Translate(-2, 45)
	hpx, mpx := p.HPImg.Bounds().Max.X, p.MPImg.Bounds().Max.X
	hpx, mpx = p.Client.HP*hpx/p.Client.MaxHP, p.Client.MP*mpx/p.Client.MaxMP
	hpRect := image.Rect(p.HPImg.Bounds().Min.X, p.HPImg.Bounds().Min.Y, hpx, p.HPImg.Bounds().Max.Y)
	mpRect := image.Rect(p.MPImg.Bounds().Min.X, p.MPImg.Bounds().Min.Y, mpx, p.MPImg.Bounds().Max.Y)
	op.GeoM.Translate(-1, -1)
	screen.DrawImage(p.HPMPBGImg, op)
	op.GeoM.Translate(1, 1)
	screen.DrawImage(p.HPImg.SubImage(hpRect).(*ebiten.Image), op)
	op.GeoM.Translate(0, 5)
	screen.DrawImage(p.MPImg.SubImage(mpRect).(*ebiten.Image), op)
	tx, ty := int(p.Pos[0]+14), int(p.Pos[1]+38)
	xoff := (len(p.Nick) * 3) - 1
	ebitenutil.DebugPrintAt(screen, p.Nick, tx-xoff, ty)
}

func (p *P) UpdateFrames(c int) {
	if c%6 == 0 {
		if p.Walking {
			if p.Dead {
				p.DeadBody.Next(p.Direction)
				p.DeadHead.Next(p.Direction)
			} else {
				if p.Armor != nil {
					p.Armor.Next(p.Direction)
				} else {
					p.NakedBody.Next(p.Direction)
				}
				p.Head.Next(p.Direction)
				if p.Helmet != nil {
					p.Helmet.Next(p.Direction)
				}
				if p.Weapon != nil {
					p.Weapon.Next(p.Direction)
				}
				if p.Shield != nil {
					p.Shield.Next(p.Direction)
				}
			}
		} else {
			if p.Dead {
				p.DeadBody.Stopped(p.Direction)
				p.DeadHead.Stopped(p.Direction)
			} else {
				if p.Armor != nil {
					p.Armor.Stopped(p.Direction)
				} else {
					p.NakedBody.Stopped(p.Direction)
				}
				p.Head.Stopped(p.Direction)
				if p.Helmet != nil {
					p.Helmet.Stopped(p.Direction)
				}
				if p.Weapon != nil {
					p.Weapon.Stopped(p.Direction)
				}
				if p.Shield != nil {
					p.Shield.Stopped(p.Direction)
				}
			}
		}
	}
}

func (p *P) SetSoundboard(sb *audio2d.SoundBoard) {
	p.soundboard = sb
}

type Step struct {
	To     typ.P
	Dir    direction.D
	Expect bool
}

func (p *P) WalkSteps(g *grid.Grid) {
	if p.leftForMove > 0 {
		vel := p.MoveSpeed
		if p.leftForMove < vel {
			vel = p.leftForMove
		}
		switch p.lastDir {
		case direction.Front:
			p.Pos[1] += vel
		case direction.Back:
			p.Pos[1] -= vel
		case direction.Left:
			p.Pos[0] -= vel
		case direction.Right:
			p.Pos[0] += vel
		}
		p.leftForMove -= vel
	}
	if p.leftForMove != 0 {
		return
	}
	if len(p.steps) == 0 {
		p.Walking = false
		return
	}
	step := p.steps[0]
	p.steps = p.steps[1:]
	if p.X == step.To.X && p.Y == step.To.Y {
		p.Direction = step.Dir
		return
	}
	if !p.Dead {
		if !p.Walking {
			p.soundboard.PlayFrom(assets.Walk1, p.local.X, p.local.Y, p.X, p.Y)
			p.soundPrevWalk = 1
		} else {
			if p.soundPrevWalk == 1 {
				p.soundboard.PlayFrom(assets.Walk2, p.local.X, p.local.Y, p.X, p.Y)
				p.soundPrevWalk = 2
			} else {
				p.soundboard.PlayFrom(assets.Walk1, p.local.X, p.local.Y, p.X, p.Y)
				p.soundPrevWalk = 1
			}
		}
	}
	p.Direction = step.Dir
	p.lastDir = step.Dir
	p.leftForMove = constants.TileSize
	p.Walking = true
	p.Pos[0] = float64(p.X * constants.TileSize)
	p.Pos[1] = float64(p.Y * constants.TileSize)
	g.Move(0, typ.P{X: int32(p.X), Y: int32(p.Y)}, step.To)
	p.X, p.Y = step.To.X, step.To.Y
}

func (p *P) AddStep(e *msgs.EventPlayerMoved) {
	p.steps = append(p.steps, Step{
		To:  e.Pos,
		Dir: e.Dir,
	})
}

func NewLogin(e *msgs.EventPlayerLogin) *P {
	p := Create(e)
	p.Client = NewClientP()
	p.Client.MaxHP = int(e.MaxHP)
	p.Client.MaxMP = int(e.MaxMP)
	p.Client.HP = int(e.HP)
	p.Client.MP = int(e.MP)
	p.Client.p = p
	p.HPImg = ebiten.NewImage(30, 3)
	p.MPImg = ebiten.NewImage(30, 3)
	p.HPMPBGImg = ebiten.NewImage(32, 10)
	p.HPImg.Fill(color.RGBA{255, 60, 60, 255})
	p.MPImg.Fill(color.RGBA{40, 130, 250, 255})
	p.HPMPBGImg.Fill(color.RGBA{0, 0, 0, 200})
	return p
}

func Create(a *msgs.EventPlayerLogin) *P {
	p := &P{
		ID:        uint32(a.ID),
		X:         a.Pos.X,
		Y:         a.Pos.Y,
		Direction: a.Dir,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.Pos.X) * constants.TileSize,
			float64(a.Pos.Y) * constants.TileSize},
		Nick:      a.Nick,
		Dead:      a.Dead,
		MoveSpeed: float64(a.Speed),
		Armor:     texture.LoadAnimation(assets.DarkArmour),
		Helmet:    texture.LoadStill(assets.ProHat),
		Weapon:    texture.LoadAnimation(assets.SpecialSword),
		Shield:    texture.LoadAnimation(assets.SilverShield),
		NakedBody: texture.LoadAnimation(assets.NakedBody),
		Head:      texture.LoadStill(assets.Head),
		DeadBody:  texture.LoadAnimation(assets.DeadBody),
		DeadHead:  texture.LoadStill(assets.DeadHead),
	}
	p.Effect = &PEffects{
		p:      p,
		active: make([]texture.Effect, 0),
		Add:    make(chan texture.Effect, 10),
	}
	return p
}

func CreateFromLogin(local *P, a *msgs.EventNewPlayer) *P {
	p := &P{
		local:     local,
		ID:        uint32(a.ID),
		X:         a.Pos.X,
		Y:         a.Pos.Y,
		Direction: a.Dir,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.Pos.X) * constants.TileSize,
			float64(a.Pos.Y) * constants.TileSize},
		Nick:      a.Nick,
		Dead:      a.Dead,
		MoveSpeed: float64(a.Speed),
		Armor:     texture.LoadAnimation(assets.DarkArmour),
		Helmet:    texture.LoadStill(assets.ProHat),
		Weapon:    texture.LoadAnimation(assets.SpecialSword),
		Shield:    texture.LoadAnimation(assets.SilverShield),
		NakedBody: texture.LoadAnimation(assets.NakedBody),
		Head:      texture.LoadStill(assets.Head),
		DeadBody:  texture.LoadAnimation(assets.DeadBody),
		DeadHead:  texture.LoadStill(assets.DeadHead),
	}
	p.Effect = &PEffects{
		p:      p,
		active: make([]texture.Effect, 0),
		Add:    make(chan texture.Effect, 10),
	}
	return p
}

func CreatePlayerSpawned(local *P, a *msgs.EventPlayerSpawned) *P {
	p := &P{
		local:     local,
		ID:        uint32(a.ID),
		X:         a.Pos.X,
		Y:         a.Pos.Y,
		Direction: a.Dir,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.Pos.X) * constants.TileSize,
			float64(a.Pos.Y) * constants.TileSize},
		Nick:      a.Nick,
		Dead:      a.Dead,
		MoveSpeed: float64(a.Speed),
		Armor:     texture.LoadAnimation(assets.DarkArmour),
		Helmet:    texture.LoadStill(assets.ProHat),
		Weapon:    texture.LoadAnimation(assets.SpecialSword),
		Shield:    texture.LoadAnimation(assets.SilverShield),
		NakedBody: texture.LoadAnimation(assets.NakedBody),
		Head:      texture.LoadStill(assets.Head),
		DeadBody:  texture.LoadAnimation(assets.DeadBody),
		DeadHead:  texture.LoadStill(assets.DeadHead),
	}
	p.Effect = &PEffects{
		p:      p,
		active: make([]texture.Effect, 0),
		Add:    make(chan texture.Effect, 10),
	}
	return p
}

// maps.YSortable
func (p *P) ValueY() float64 { return p.Pos[1] }

type PEffects struct {
	p      *P
	active []texture.Effect
	Add    chan texture.Effect
}

func (pfx *PEffects) NewMeleeHit() {
	pfx.Add <- texture.LoadEffect(assets.MeleeHit)
}

func (pfx *PEffects) NewSpellHit(s spell.Spell) {
	a := texture.AssetFromSpell(s)
	if a != assets.Nothing {
		pfx.Add <- NewSpellOffset(a)
	}
}

func (pfx *PEffects) NewAttackNumber(dmg int) {
	if dmg > 0 {
		pfx.Add <- &AtkDmgFxTxt{img: ebiten.NewImage(40, 40), dmg: strconv.FormatInt(int64(dmg), 10)}
	}
}

type SpellOffset struct {
	x, y int
	fx   texture.Effect
}

var spellOffsets = map[assets.Image]struct{ x, y int }{
	assets.SpellApoca:      {-50, -90},
	assets.SpellInmo:       {-30, -55},
	assets.SpellInmoRm:     {-20, -30},
	assets.SpellDesca:      {-45, -70},
	assets.SpellHealWounds: {-36, -38},
	assets.SpellResurrect:  {-24, -36},
}

func NewSpellOffset(a assets.Image) *SpellOffset {
	off := spellOffsets[a]
	return &SpellOffset{off.x, off.y, texture.LoadEffect(a)}
}

func (as *SpellOffset) Play() bool {
	return as.fx.Play()
}

func (as *SpellOffset) EffectFrame() *ebiten.Image {
	return as.fx.EffectFrame()
}

func (as *SpellOffset) EffectOpt(op *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {
	op.GeoM.Translate(float64(as.x), float64(as.y))
	return op
}

type AtkDmgFxTxt struct {
	dmg string
	img *ebiten.Image
	y   int
}

func (adt *AtkDmgFxTxt) Play() bool {
	adt.img.Clear()
	ebitenutil.DebugPrintAt(adt.img, adt.dmg, 0, 20-adt.y)
	if adt.y == 20 {
		adt.y = 0
		return false
	}
	adt.y++
	return true
}

func (adt *AtkDmgFxTxt) EffectFrame() *ebiten.Image {
	return adt.img
}

func (a *AtkDmgFxTxt) EffectOpt(op *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {
	op.GeoM.Translate(6, -50)
	return op
}

func (pfx *PEffects) Update(counter int) {
	if nfx, ok := conc.Check(pfx.Add); ok {
		pfx.active = append(pfx.active, nfx)
	}
	if counter%2 == 0 {
		i := 0
		for _, fx := range pfx.active {
			if fx.Play() {
				pfx.active[i] = fx
				i++
			}
		}
		for j := i; j < len(pfx.active); j++ {
			pfx.active[j] = nil
		}
		pfx.active = pfx.active[:i]
	}
}

func (pfx *PEffects) Draw(screen *ebiten.Image) {
	for _, fx := range pfx.active {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pfx.p.Pos[0], pfx.p.Pos[1])
		screen.DrawImage(fx.EffectFrame(), fx.EffectOpt(op))
	}
}
