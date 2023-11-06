package game

import (
	_ "embed"
	"fmt"
	_ "image/png"
	"log"
	"math"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/conc"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/direction"
	"github.com/rywk/minigoao/pkg/messenger"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/events"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
)

const (
	titleFontSize = fontSize * 1.5
	fontSize      = 24
	smallFontSize = fontSize / 4

	headOffset = -20

	defaultServerAddress = "localhost"
)

var (
	titleArcadeFont font.Face
	arcadeFont      font.Face
	smallArcadeFont font.Face
)

func init() {
	tt, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	titleArcadeFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    titleFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	arcadeFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	smallArcadeFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    smallFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type Mode int

const (
	ModeRegister Mode = iota
	ModeGame
	ModeOptions
)

type Game struct {
	mode Mode

	camera *Camera

	typer *typing.Typer

	latency string

	counter int

	// Messaging api
	m *messenger.M

	// Event handlers
	ehs     [events.Len]func([]byte)
	handler *Handler

	// World map info
	world *Map

	// Player session id given by server
	sessionID uint32

	// Player map
	players [constants.MaxConnCount]*player.P
	// player depth matrix (for drawing)
	playersY           []*player.P
	playersYsyncHelper [constants.MaxConnCount]*player.P

	player *player.P

	// events channel
	gameEvent chan *message.Event

	clientReady chan struct{}

	// Here we listen to what the client wants to do
	// and return the result to the player
	client *player.ClientP

	keys *Keys

	lastMove          time.Time
	leftForMove       int // pixels left to complete tile change
	lastDir           direction.D
	lastMoveOkArrived bool
	moveLatency       time.Time

	audioContext *audio.Context
	jumpPlayer   *audio.Player
	hitPlayer    *audio.Player

	timeForStep  time.Duration
	startForStep time.Time

	ViewPort   f64.Vec2
	ZoomFactor float64
	Rotation   int
}

const ScreenWidth, ScreenHeight = 1312, 928

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func NewGame() *Game {
	g := &Game{}
	g.init()
	return g
}

func (g *Game) init() {
	tcp, err := net.Dial("tcp4", defaultServerAddress+constants.Port)
	if err != nil {
		log.Fatal(err)
	}
	g.mode = ModeRegister
	g.typer = typing.NewTyper()
	g.lastMoveOkArrived = true
	g.ViewPort = f64.Vec2{ScreenWidth, ScreenHeight}
	g.ZoomFactor = 1
	g.lastMove = time.Now()
	g.keys = NewKeys(nil)
	g.m = messenger.New(tcp, nil, nil)
	g.handler = NewHandler(g)
	g.gameEvent = make(chan *message.Event, 10)
	// Start handler
	go g.handler.TCP()
	// Only do map stuff after this
	g.clientReady = make(chan struct{})
	close(g.clientReady)
}

func (g *Game) StartGame(nick string) {
	log.Println("Sending register nick")
	g.handler.SendRegister(nick)
	// Explicitly wait and process next event.
	// It has to be the registration ok
	// This creates the local player
	g.handler.RegisterOk(<-g.handler.Events)
	g.playersY = append(g.playersY, g.player)
	// After registration was handled
	// start to consume everything
	go g.handler.Start()
	go g.handler.HandleClient()
	go g.handler.SendPing()
	g.mode = ModeGame
}

func (g *Game) Ready() {
	<-g.clientReady
}

func (g *Game) Update() error {
	switch g.mode {
	case ModeRegister:
		g.updateRegister()
	case ModeGame:
		g.updateGame()
	case ModeOptions:

	}
	return nil
}

func (g *Game) updateRegister() {
	g.typer.Update()
	text := g.typer.Text()
	if strings.HasSuffix(text, "\n") {
		g.StartGame(strings.Trim(text, "\n"))
	}
}

func (g *Game) updateGame() error {
	g.ProcessMovement()
	g.world.Update()
	g.player.Update(g.counter, LocalGrid)
	for i, p := range g.players {
		if p == nil {
			continue
		}
		if _, ok := conc.Check(p.Kill); ok {
			log.Printf("despawn player %v\n", p.Nick)
			g.players[i] = nil
			MustAt(p.X, p.Y).SimpleDespawn(p)
			for iy, py := range g.playersY {
				if i == int(py.ID) {
					g.playersY[iy] = g.playersY[len(g.playersY)-1]
					g.playersY = g.playersY[:len(g.playersY)-1]
					break
				}
			}
			continue
		}
		p.Update(g.counter, LocalGrid)
	}

	sort.Slice(g.playersY, func(i, j int) bool {
		return g.playersY[i].Pos[1] < g.playersY[j].Pos[1]
	})

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		if g.ZoomFactor > -2400 {
			g.ZoomFactor -= 1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		if g.ZoomFactor < 2400 {
			g.ZoomFactor += 1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.Rotation += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.Reset()
	}
	g.counter++
	return nil
}

func (g *Game) ProcessMovement() {
	g.keys.ListenMovement()
	d := g.keys.MovingTo()
	allowed, ok := conc.Check(g.client.MoveOk)
	if ok {
		log.Println("confirmed move", allowed, "in", time.Since(g.moveLatency).String(),
			"with last dir", direction.S(g.lastDir))
		g.lastMoveOkArrived = true
		if !allowed {
			goBackPx := float64(constants.TileSize - g.leftForMove)
			log.Println("move not allowed, go back", goBackPx, "px")
			switch g.lastDir {
			case direction.Front:
				g.player.Pos[1] -= goBackPx
			case direction.Back:
				g.player.Pos[1] += goBackPx
			case direction.Left:
				g.player.Pos[0] += goBackPx
			case direction.Right:
				g.player.Pos[0] -= goBackPx
			}
			g.player.Walking = false
			g.leftForMove = 0
		} else {
			switch g.lastDir {
			case direction.Front:
				g.player.Y += 1
			case direction.Back:
				g.player.Y -= 1
			case direction.Left:
				g.player.X -= 1
			case direction.Right:
				g.player.X += 1
			}
		}
	}
	if g.leftForMove > 0 {
		vel := 4.0
		switch g.lastDir {
		case direction.Front:
			g.player.Pos[1] += vel
		case direction.Back:
			g.player.Pos[1] -= vel
		case direction.Left:
			g.player.Pos[0] -= vel
		case direction.Right:
			g.player.Pos[0] += vel
		}
		g.leftForMove -= int(vel)
	} else {
		g.player.Walking = false
	}
	if g.leftForMove == 0 && g.lastMoveOkArrived && d != direction.Still && time.Since(g.startForStep).Milliseconds() > 90 {
		g.startForStep = time.Now()
		g.player.Direction = d
		x, y := g.client.DirToNewPos(d)
		log.Println(direction.S(d), x, y, g.player.X, g.player.Y)
		t, ok := LocalGrid.At(int16(x), int16(y))
		if !ok || t.Range(func(u thing.Thing) error {
			if u != nil && u.Blocking() {
				return constants.Err{}
			}
			return nil
		}) != nil {
			log.Println("not ok or block")
			g.client.Dir <- d
			return
		}
		g.moveLatency = time.Now()
		g.client.Move <- d
		g.lastDir = d
		g.lastMoveOkArrived = false
		g.leftForMove = constants.TileSize
		g.player.Walking = true
	} else if d != g.player.Direction && d != direction.Still && g.leftForMove < constants.TileSize/2 {
		g.player.Direction = d
		x, y := g.client.DirToNewPos(d)
		log.Println(direction.S(d), x, y, g.player.X, g.player.Y)
		t, ok := LocalGrid.At(int16(x), int16(y))
		if !ok || t.Range(func(u thing.Thing) error {
			if u != nil && u.Blocking() {
				return constants.Err{}
			}
			return nil
		}) != nil {
			log.Println("not ok or block")
			g.client.Dir <- d
			return
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.mode {
	case ModeRegister:
		g.drawRegister(screen)
	case ModeGame:
		g.drawGame(screen)
	case ModeOptions:
	}
}

func (g *Game) drawRegister(screen *ebiten.Image) {
	g.typer.Draw(screen, HalfScreenX, HalfScreenY)
}

func (g *Game) drawGame(screen *ebiten.Image) {
	g.world.Draw()
	for _, p := range g.playersY {
		if p == nil {
			continue
		}
		p.Draw(g.world.Image())
		if g.sessionID == p.ID {
			p.DrawPlayerHPMP(g.world.Image())
		} else {
			p.DrawNick(g.world.Image())
		}
	}
	g.Render(g.world.Image(), screen)

	worldX, worldY := g.ScreenToWorld(ebiten.CursorPosition())

	ebitenutil.DebugPrint(screen,
		fmt.Sprintf("TPS: %0.2f\nMove (WASD/Arrows)\nZoom (QE)\nRotate (R)\nReset (Space)", ebiten.ActualTPS()))
	ebitenutil.DebugPrintAt(screen, g.latency, ScreenWidth-110, 0)

	ebitenutil.DebugPrintAt(screen,
		fmt.Sprintf("%s\nCursor World Pos: %d,%d", g.String(),
			int(worldX/constants.TileSize), int(worldY/constants.TileSize)),
		0, ScreenHeight-32,
	)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
}

type Camera struct {
}

func (g *Game) String() string {
	return fmt.Sprintf(
		"T: [%d,%d], R: %d, S: %f",
		g.player.X, g.player.Y,
		g.Rotation, g.ZoomFactor,
	)
}

func (g *Game) viewportCenter() f64.Vec2 {
	return f64.Vec2{
		g.ViewPort[0] * 0.5,
		g.ViewPort[1] * 0.5,
	}
}

const HalfScreenX, HalfScreenY = ScreenWidth / 2, ScreenHeight / 2

func (g *Game) worldMatrix() ebiten.GeoM {
	m := ebiten.GeoM{}
	m.Translate(-g.player.Pos[0]+HalfScreenX-16, -g.player.Pos[1]+HalfScreenY-16)
	// We want to scale and rotate around center of image / screen
	m.Translate(-g.viewportCenter()[0], -g.viewportCenter()[1])
	m.Scale(
		float64(g.ZoomFactor),
		float64(g.ZoomFactor),
	)
	m.Rotate(float64(g.Rotation) * 2 * math.Pi / 360)
	m.Translate(g.viewportCenter()[0], g.viewportCenter()[1])
	return m
}

func (g *Game) Render(world, screen *ebiten.Image) {
	screen.DrawImage(world, &ebiten.DrawImageOptions{
		GeoM: g.worldMatrix(),
	})
}

func (g *Game) ScreenToWorld(posX, posY int) (float64, float64) {
	inverseMatrix := g.worldMatrix()
	if inverseMatrix.IsInvertible() {
		inverseMatrix.Invert()
		return inverseMatrix.Apply(float64(posX), float64(posY))
	} else {
		// When scaling it can happened that matrix is not invertable
		return math.NaN(), math.NaN()
	}
}

func (g *Game) Reset() {
	g.Rotation = 0
	if g.counter%10 == 0 {
		if g.ZoomFactor == 1 {
			g.ZoomFactor = 2
		} else if g.ZoomFactor == 2 {
			g.ZoomFactor = 0.6
		} else {
			g.ZoomFactor = 1
		}
	}
}
