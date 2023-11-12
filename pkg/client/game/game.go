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
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/conc"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/potion"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/messenger"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/assets"
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
	stats  *Stats

	keys       *Keys
	combatKeys *CombatKeys

	lastMove          time.Time
	leftForMove       float64 // pixels left to complete tile change
	lastDir           direction.D
	lastMoveOkArrived bool
	moveLatency       time.Time
	soundPrevWalk     int

	audioContext *audio.Context

	timeForStep  time.Duration
	startForStep time.Time
	SoundBoard   *audio2d.SoundBoard

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
	g.SoundBoard = audio2d.NewSoundBoard()
	g.mode = ModeRegister
	g.typer = typing.NewTyper()
	g.lastMoveOkArrived = true
	g.ViewPort = f64.Vec2{ScreenWidth, ScreenHeight}
	g.ZoomFactor = 1
	g.lastMove = time.Now()
	g.keys = NewKeys(nil)
	g.combatKeys = NewCombatKeys(nil)
	// Only do map stuff after this
	g.clientReady = make(chan struct{})
	close(g.clientReady)
}

func (g *Game) StartGame(nick string) {
	tcp, err := net.Dial("tcp4", defaultServerAddress+constants.Port)
	if err != nil {
		log.Fatal(err)
	}
	g.m = messenger.New(tcp, nil, nil)
	g.handler = NewHandler(g)
	g.gameEvent = make(chan *message.Event, 10)
	// Start handler
	go g.handler.TCP()
	log.Println("Sending register nick")
	g.handler.SendRegister(nick)
	// Explicitly wait and process next event.
	// It has to be the registration ok
	// This creates the local player
	g.handler.RegisterOk(<-g.handler.Events)
	g.playersY = append(g.playersY, g.player)
	g.stats = NewStats(g, 15, ScreenHeight-95)
	// After registration was handled
	// start to consume everything
	go g.handler.Start()
	go g.handler.HandleClient()
	go g.handler.SendPing()
	g.mode = ModeGame

	g.SoundBoard.Play(assets.Spawn)
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
	g.ProcessCombat()
	g.world.Update()
	g.player.Update(g.counter, LocalGrid)
	g.player.Effect.Update(g.counter)
	for i, p := range g.players {
		if p == nil {
			continue
		}
		if _, ok := conc.Check(p.Kill); ok {
			log.Printf("despawn player %v\n", p.Nick)
			g.players[i] = nil
			if !p.Dead {
				MustAt(p.X, p.Y).SimpleDespawn(p)
			}
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
		p.Effect.Update(g.counter)
	}
	sort.Slice(g.playersY, func(i, j int) bool {
		return g.playersY[i].Pos[1] < g.playersY[j].Pos[1]
	})
	g.stats.Update()
	g.combatKeys.MoveSpellPicker()
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.Reset()
	}
	g.counter++
	return nil
}

func (g *Game) ProcessCombat() {
	if g.combatKeys.MeleeHit() {
		g.client.CastMelee <- struct{}{}
	}
	if ok, spellType, x, y := g.combatKeys.CastSpell(); ok {
		log.Printf("casted spell: %v\n", spell.S(spellType))
		worldX, worldY := g.ScreenToWorld(x, y)
		g.client.CastSpell <- &message.CastSpell{
			X:     uint32(worldX / constants.TileSize),
			Y:     uint32(worldY / constants.TileSize),
			Spell: spellType}
	}
	if pressedPotion := g.combatKeys.PressedPotion(); pressedPotion != potion.None {
		g.client.UsePotion <- &message.UsePotion{Type: pressedPotion}
	}
	select {
	case m := <-g.client.CastMeleeOk:
		log.Printf("CastMeleeOk m: %#v\n", m)
		if !m.Ok {
			g.SoundBoard.Play(assets.MeleeAir)
			break
		}
		g.SoundBoard.Play(assets.MeleeBlood)
		g.player.Effect.NewAttackNumber(int(m.Dmg))
		g.players[m.Id].Effect.NewMeleeHit()
	case m := <-g.client.RecivedMelee:
		g.SoundBoard.Play(assets.MeleeBlood)
		g.player.Effect.NewMeleeHit()
		log.Printf("RecivedMelee m: %#v\n", m)
		g.players[m.Id].Effect.NewAttackNumber(int(m.Dmg))
		g.player.Client.HP = int(m.Hp)
		if g.player.Client.HP == 0 {
			g.player.Dead = true
		}
	case m := <-g.client.MeleeHit:
		log.Printf("MeleeHit m: %#v\n", m)
		if !m.Ok {
			g.SoundBoard.PlayFrom(assets.MeleeAir, g.player.X, g.player.Y, g.players[m.From].X, g.players[m.From].Y)
			break
		}
		g.SoundBoard.PlayFrom(assets.MeleeBlood, g.player.X, g.player.Y, g.players[m.To].X, g.players[m.To].Y)
		g.players[m.To].Effect.NewMeleeHit()
	case m := <-g.client.CastSpellOk:
		log.Printf("CastSpellOk m: %#v\n", m)
		if !m.Ok {
			break
		}
		g.player.Client.MP = int(m.Mp)
		g.player.Effect.NewAttackNumber(int(m.Dmg))
		if m.Id == g.sessionID {
			g.player.Effect.NewSpellHit(m.Spell)
			g.SoundBoard.Play(assets.SoundFromSpell(m.Spell))
		} else {
			g.players[m.Id].Effect.NewSpellHit(m.Spell)
			g.SoundBoard.PlayFrom(assets.SoundFromSpell(m.Spell), g.player.X, g.player.Y, g.players[m.Id].X, g.players[m.Id].Y)
		}

	case m := <-g.client.RecivedSpell:
		log.Printf("RecivedSpell m: %#v\n", m)
		switch m.Spell {
		case spell.Inmo:
			g.player.Inmobilized = true
		case spell.InmoRm:
			g.player.Inmobilized = false
		case spell.Revive:
			g.player.Dead = false
		}
		if m.Id == g.sessionID {
			break
		}
		g.SoundBoard.Play(assets.SoundFromSpell(m.Spell))
		g.player.Effect.NewSpellHit(m.Spell)
		g.players[m.Id].Effect.NewAttackNumber(int(m.Dmg))
		g.player.Client.HP = int(m.Hp)
		if g.player.Client.HP == 0 {
			g.player.Dead = true
		}
	case m := <-g.client.SpellHit:
		log.Printf("SpellHit m: %#v\n", m)
		g.SoundBoard.PlayFrom(assets.SoundFromSpell(m.Spell), g.player.X, g.player.Y, g.players[m.To].X, g.players[m.To].Y)
		g.players[m.To].Effect.NewSpellHit(m.Spell)
	case m := <-g.client.UsePotionOk:
		log.Printf("UsePotionOk m: %#v\n", m)
		if m.Ok {
			g.player.Client.HP += int(m.DeltaHP)
			g.player.Client.MP += int(m.DeltaMP)
			g.SoundBoard.Play(assets.Potion)
		}
	case m := <-g.client.PotionUsed:
		log.Printf("PotionUsed m: %#v\n", m)
		g.SoundBoard.PlayFrom(assets.Potion, g.player.X, g.player.Y, int(m.X), int(m.Y))
	default:
	}
}

func (g *Game) ProcessMovement() {
	if allowed, ok := conc.Check(g.client.MoveOk); ok {
		log.Println("Move:", allowed, "in", time.Since(g.moveLatency).String(),
			"dir:", direction.S(g.lastDir))
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
		vel := 3.0
		if g.leftForMove < vel {
			vel = g.leftForMove
		}
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
		g.leftForMove -= vel
	} else {
		g.player.Walking = false
	}
	g.keys.ListenMovement()
	d := g.keys.MovingTo()
	if g.leftForMove == 0 && g.lastMoveOkArrived && d != direction.Still && time.Since(g.startForStep).Milliseconds() > 90 {
		g.startForStep = time.Now()
		g.player.Direction = d
		x, y := g.client.DirToNewPos(d)
		t, ok := LocalGrid.At(int16(x), int16(y))
		if !ok || t.Range(func(u thing.Thing) error {
			if u != nil && u.Blocking() {
				return constants.Err{}
			}
			return nil
		}) != nil {
			g.client.Dir <- d
			return
		}
		if g.player.Inmobilized {
			g.client.Dir <- d
			return
		}
		if !g.player.Dead {
			if !g.player.Walking {
				g.SoundBoard.Play(assets.Walk1)
				g.soundPrevWalk = 1
			} else {
				if g.soundPrevWalk == 1 {
					g.SoundBoard.Play(assets.Walk2)
					g.soundPrevWalk = 2
				} else {
					g.SoundBoard.Play(assets.Walk1)
					g.soundPrevWalk = 1
				}
			}
		}
		g.moveLatency = time.Now()
		g.client.Move <- d
		g.lastDir = d
		g.lastMoveOkArrived = false
		g.leftForMove = constants.TileSize
		g.player.Walking = true
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
		p.Effect.Draw(g.world.Image())
	}
	g.Render(g.world.Image(), screen)

	ebitenutil.DebugPrint(screen,
		fmt.Sprintf("TPS: %0.2f\nFPS: %v\nPing: %v\nMove (WASD)\nMelee (Space)", ebiten.ActualTPS(), int(ebiten.ActualFPS()), g.latency))

	g.combatKeys.ShowSpellPicker(screen)
	g.stats.Draw(screen)
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
