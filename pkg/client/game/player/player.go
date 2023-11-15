package player

import (
	"image"
	"image/color"
	"log"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/conc"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/actions"
	"github.com/rywk/minigoao/proto/message/assets"
	"github.com/rywk/tile"
	"golang.org/x/image/math/f64"
)

// Player
type P struct {
	local       *P
	ID          uint32
	Nick        string
	X, Y        int
	DX, DY      int
	Pos         f64.Vec2
	Tile        tile.Tile[thing.Thing]
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

	leftForMove float64
	lastDir     direction.D

	Kill               chan struct{}
	movesBuffered      chan *message.PlayerAction
	directionsBuffered chan *message.PlayerAction

	soundPrevWalk int
	soundboard    *audio2d.SoundBoard
}

type ClientP struct {
	p *P
	// Stats
	HP, MP       int
	MaxHP, MaxMP int

	// Server channels, used to ask permision and recive updates
	Move   chan direction.D
	MoveOk chan bool

	Dir chan direction.D

	UsePotion   chan *message.UsePotion
	UsePotionOk chan *message.UsePotionOk
	PotionUsed  chan *message.PotionUsed

	CastMelee    chan struct{}
	CastMeleeOk  chan *message.CastMeleeOk
	RecivedMelee chan *message.RecivedMelee
	MeleeHit     chan *message.MeleeHit

	CastSpell    chan *message.CastSpell
	CastSpellOk  chan *message.CastSpellOk
	RecivedSpell chan *message.RecivedSpell
	SpellHit     chan *message.SpellHit
}

func NewClientP() *ClientP {
	return &ClientP{
		Move:   make(chan direction.D),
		Dir:    make(chan direction.D),
		MoveOk: make(chan bool),

		UsePotion:   make(chan *message.UsePotion),
		UsePotionOk: make(chan *message.UsePotionOk),
		PotionUsed:  make(chan *message.PotionUsed),

		CastMelee:    make(chan struct{}),
		CastMeleeOk:  make(chan *message.CastMeleeOk),
		RecivedMelee: make(chan *message.RecivedMelee),
		MeleeHit:     make(chan *message.MeleeHit),

		CastSpell:    make(chan *message.CastSpell),
		CastSpellOk:  make(chan *message.CastSpellOk),
		RecivedSpell: make(chan *message.RecivedSpell),
		SpellHit:     make(chan *message.SpellHit),
	}
}

func (p *ClientP) DirToNewPos(d direction.D) (int, int) {
	switch d {
	case direction.Front:
		return p.p.X, p.p.Y + 1
	case direction.Back:
		return p.p.X, p.p.Y - 1
	case direction.Left:
		return p.p.X - 1, p.p.Y
	case direction.Right:
		return p.p.X + 1, p.p.Y
	}
	return 0, 0
}

func (p *P) Nil() bool { return p == nil }

func (p *P) Update(counter int, g *tile.Grid[thing.Thing]) {
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
	if p.Client == nil {
		p.Mover(g)
	}
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

func (p *P) Process(a *message.PlayerAction) {
	p.Dead = a.Dead
	switch a.Action {
	case actions.Despawn:
		close(p.Kill)
	case actions.Dir:
		log.Printf("change dir to %v\n", direction.S(a.D))
		p.directionsBuffered <- a
	case actions.Move:
		log.Printf("new move update from %v!!\n", a.Nick)
		p.movesBuffered <- a
	case actions.Died:
		p.Dead = true
	case actions.Revive:
		log.Printf("revive!!\n")
		p.Dead = false
	}
}

func (p *P) Mover(g *tile.Grid[thing.Thing]) {
	if a, ok := conc.Check(p.directionsBuffered); ok {
		p.Direction = a.D
	}
	if p.leftForMove == 0 {
		if a, ok := conc.Check(p.movesBuffered); ok {
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
			p.Direction = a.D
			p.lastDir = a.D
			p.leftForMove = constants.TileSize
			p.Walking = true
			p.Pos[0] = float64(p.X * constants.TileSize)
			p.Pos[1] = float64(p.Y * constants.TileSize)
			pt, _ := g.At(int16(p.X), int16(p.Y))
			p.X, p.Y = int(a.X), int(a.Y)
			t, _ := g.At(int16(p.X), int16(p.Y))
			pt.SimpleMoveTo(t, p)
		} else {
			p.Walking = false
		}
	}
	if p.leftForMove > 0 {
		vel := 3.0
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
}

func NewRegisterOk(e *message.RegisterOk) (*P, *ClientP) {
	p := actionToP(nil, e.Self)
	p.Client = NewClientP()
	p.Client.MaxHP = int(e.MaxHP)
	p.Client.MaxMP = int(e.MaxMP)
	p.Client.HP = int(e.HP)
	p.Client.MP = int(e.MP)
	p.Client.p = p
	p.HPImg = ebiten.NewImage(30, 3)
	p.MPImg = ebiten.NewImage(30, 3)
	p.HPImg.Fill(color.RGBA{250, 20, 20, 255})
	p.MPImg.Fill(color.RGBA{40, 130, 250, 255})
	return p, p.Client
}

func ProcessNew(l *P, pa *message.PlayerAction, sb *audio2d.SoundBoard) *P {
	p := actionToP(l, pa)
	p.soundboard = sb
	p.Process(pa)
	return p
}

func NewFromLogIn(l *P, e *message.RegisterOk, sb *audio2d.SoundBoard) map[uint32]*P {
	r := map[uint32]*P{}
	for _, s := range e.Spawns {
		r[s.Id] = actionToP(l, s)
		r[s.Id].soundboard = sb
	}
	return r
}

func actionToP(l *P, a *message.PlayerAction) *P {
	p := &P{
		local:     l,
		ID:        a.Id,
		X:         int(a.X),
		Y:         int(a.Y),
		Direction: a.D,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.X) * constants.TileSize,
			float64(a.Y) * constants.TileSize},
		Nick:               a.Nick,
		Dead:               a.Dead,
		Armor:              texture.LoadAnimation(a.Armor),
		Helmet:             texture.LoadStill(a.Helmet),
		Weapon:             texture.LoadAnimation(a.Weapon),
		Shield:             texture.LoadAnimation(a.Shield),
		NakedBody:          texture.LoadAnimation(assets.NakedBody),
		Head:               texture.LoadStill(assets.Head),
		DeadBody:           texture.LoadAnimation(assets.DeadBody),
		DeadHead:           texture.LoadStill(assets.DeadHead),
		movesBuffered:      make(chan *message.PlayerAction, 10),
		directionsBuffered: make(chan *message.PlayerAction, 10),
		Kill:               make(chan struct{}),
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

// thing.Thing
func (p *P) What() uint32       { return thing.Player }
func (p *P) Who() uint32        { return p.ID }
func (p *P) Blocking() bool     { return !p.Dead }
func (p *P) Is(id uint32) bool  { return id == p.ID }
func (p *P) Player(uint32) bool { return false }

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
	assets.SpellApoca:      {-50, -80},
	assets.SpellInmo:       {-30, -55},
	assets.SpellInmoRm:     {-20, -30},
	assets.SpellDesca:      {-45, -70},
	assets.SpellHealWounds: {-22, -20},
	assets.SpellRevive:     {-28, -40},
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
	op.GeoM.Translate(4, -40)
	return op
}

func (pfx *PEffects) Update(counter int) {
	if nfx, ok := conc.Check(pfx.Add); ok {
		pfx.active = append(pfx.active, nfx)
	}
	if counter%3 == 0 {
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
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(pfx.p.Pos[0], pfx.p.Pos[1])
	for _, fx := range pfx.active {
		screen.DrawImage(fx.EffectFrame(), fx.EffectOpt(op))
	}
}
