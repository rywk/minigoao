package player

import (
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/conc"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/direction"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/actions"
	"github.com/rywk/tile"
	"golang.org/x/image/math/f64"
)

// Player
type P struct {
	ID     uint32
	Nick   string
	X, Y   int
	DX, DY int
	Pos    f64.Vec2

	Body, Weapon, Head texture.A
	Walking            bool
	Direction          direction.D

	ActiveEffects []texture.AF
	Effects       chan texture.AF

	Client       *ClientP
	HPImg, MPImg *ebiten.Image

	leftForMove int
	lastDir     direction.D

	Kill               chan struct{}
	movesBuffered      chan *message.PlayerAction
	directionsBuffered chan *message.PlayerAction
}

type ClientP struct {
	p *P
	// Stats
	HP, MP       int
	MaxHP, MaxMP int

	// cooldowns and shit

	Move   chan direction.D
	MoveOk chan bool

	Dir chan direction.D
}

func NewClientP() *ClientP {
	return &ClientP{
		Move:   make(chan direction.D),
		Dir:    make(chan direction.D),
		MoveOk: make(chan bool),
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
	p.Body.Dir(p.Direction)
	p.Head.Dir(p.Direction)
	p.Weapon.Dir(p.Direction)
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
	screen.DrawImage(p.Body.Frame(), op)
	screen.DrawImage(p.Weapon.Frame(), op)
	op.GeoM.Translate(PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY)
	screen.DrawImage(p.Head.Frame(), op)
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

	hpRect := image.Rect(p.HPImg.Bounds().Min.X, p.HPImg.Bounds().Min.Y, p.HPImg.Bounds().Max.X, p.HPImg.Bounds().Max.Y)
	screen.DrawImage(p.HPImg.SubImage(hpRect).(*ebiten.Image), op)
	op.GeoM.Translate(0, 5)
	screen.DrawImage(p.MPImg, op)
	tx, ty := int(p.Pos[0]+14), int(p.Pos[1]+38)
	xoff := (len(p.Nick) * 3) - 1
	ebitenutil.DebugPrintAt(screen, p.Nick, tx-xoff, ty)
}

func (p *P) UpdateFrames(c int) {
	if p.Walking {
		if c%6 == 0 {
			p.Body.Next(p.Direction)
			p.Head.Next(p.Direction)
			p.Weapon.Next(p.Direction)
		}
	} else {
		p.Body.Stopped(p.Direction)
		p.Head.Stopped(p.Direction)
		p.Weapon.Stopped(p.Direction)
	}
}

func (p *P) Process(a *message.PlayerAction) {
	switch a.Action {
	case actions.Spawn:
	case actions.Despawn:
		close(p.Kill)
	case actions.Dir:
		log.Printf("change dir to %v\n", direction.S(a.D))
		p.directionsBuffered <- a
	case actions.Move:
		log.Printf("new move update from %v!!\n", a.Nick)
		p.movesBuffered <- a
	}
}

func (p *P) Mover(g *tile.Grid[thing.Thing]) {
	if a, ok := conc.Check(p.directionsBuffered); ok {
		p.Direction = a.D
	}
	if p.leftForMove == 0 {
		if a, ok := conc.Check(p.movesBuffered); ok {
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
		vel := 4.0
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
		p.leftForMove -= int(vel)
	}
}

func ProcessNew(pa *message.PlayerAction) *P {
	p := actionToP(pa)
	p.Process(pa)
	return p
}

func NewRegisterOk(e *message.RegisterOk, c *ClientP) *P {
	p := actionToP(e.Self)
	p.Client = c
	p.Client.p = p
	p.HPImg = ebiten.NewImage(30, 3)
	p.MPImg = ebiten.NewImage(30, 3)
	p.HPImg.Fill(color.RGBA{250, 20, 20, 255})
	p.MPImg.Fill(color.RGBA{40, 130, 250, 255})
	return p
}

func NewFromLogIn(e *message.RegisterOk) map[uint32]*P {
	r := map[uint32]*P{}
	for _, s := range e.Spawns {
		r[s.Id] = actionToP(s)
	}
	return r
}

func actionToP(a *message.PlayerAction) *P {
	return &P{
		ID:        a.Id,
		X:         int(a.X),
		Y:         int(a.Y),
		Direction: a.D,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.X) * constants.TileSize,
			float64(a.Y) * constants.TileSize},
		Nick:               a.Nick,
		Body:               texture.LoadAnimation(a.Body),
		Weapon:             texture.LoadAnimation(a.Weapon),
		Head:               texture.LoadStill(a.Head),
		movesBuffered:      make(chan *message.PlayerAction, 10),
		directionsBuffered: make(chan *message.PlayerAction, 10),
		Kill:               make(chan struct{}),
		// actions & effects are buffered,
		// we want to do them in order
		// but we dont want to block whos sending
		Effects: make(chan texture.AF, 10),
	}
}

func (p *P) What() uint32       { return thing.Player }
func (p *P) Who() uint32        { return p.ID }
func (p *P) Blocking() bool     { return true }
func (p *P) Is(id uint32) bool  { return id == p.ID }
func (p *P) Player(uint32) bool { return false }
