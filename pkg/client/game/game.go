package game

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	_ "image/png"
	"log"
	"math"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/potion"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/server"
	"github.com/rywk/minigoao/pkg/typ"
	"golang.design/x/clipboard"
	"golang.org/x/image/math/f64"
)

type Mode int

const (
	ModeRegister Mode = iota
	ModeGame
	ModeOptions
)

type YSortable interface {
	ValueY() float64
	Draw(*ebiten.Image)
}

type Game struct {
	serverAddress string
	mode          Mode
	typer         *typing.Typer
	latency       string
	counter       int
	ms            *msgs.M
	world         *Map
	sessionID     uint32
	players       [constants.MaxConnCount]*player.P
	playersY      []YSortable
	player        *player.P
	outQueue      chan *GameMsg
	eventQueue    []*GameMsg
	eventLock     sync.Mutex

	client *player.ClientP
	stats  *Stats

	keys       *Keys
	combatKeys *CombatKeys

	lastMove          time.Time
	leftForMove       float64 // pixels left to complete tile change
	lastDir           direction.D
	lastMoveOkArrived bool
	moveStartedAt     time.Time
	soundPrevWalk     int
	startForStep      time.Time

	SoundBoard *audio2d.SoundBoard

	ViewPort   f64.Vec2
	ZoomFactor float64
	Rotation   int

	LastPing    time.Time
	WaitingPong bool
}

type GameMsg struct {
	E    msgs.E
	Data interface{}
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
	g.mode = ModeRegister
	g.typer = typing.NewTyper()
	err := clipboard.Init()
	if err != nil {
		log.Fatal(err)
	}
}

func (g *Game) StartGame(nick string, address string) {
	g.SoundBoard = audio2d.NewSoundBoard()
	g.lastMoveOkArrived = true
	g.ViewPort = f64.Vec2{ScreenWidth, ScreenHeight}
	g.ZoomFactor = 1
	g.lastMove = time.Now()
	g.keys = NewKeys(nil)
	g.combatKeys = NewCombatKeys(nil)

	tcp, err := net.Dial("tcp4", address)
	if err != nil {
		log.Fatal(err)
	}

	nickMsg := make([]byte, 2, len(nick)+2)
	binary.BigEndian.PutUint16(nickMsg, uint16(len(nick)))
	tcp.Write(append(nickMsg, []byte(nick)...))
	log.Printf("sent nick %v\n", nick)
	g.ms = msgs.New(tcp)
	im, err := g.ms.Read()
	if err != nil {
		panic(err)
	}
	if im.Event != msgs.EPlayerLogin {
		panic("not login response")
	}
	login := &msgs.EventPlayerLogin{}
	msgs.DecodeMsgpack(im.Data, login)
	g.Login(login)
	g.playersY = append(g.playersY, g.player)
	g.stats = NewStats(g, 15, ScreenHeight-95)

	g.eventQueue = make([]*GameMsg, 0, 100)
	g.outQueue = make(chan *GameMsg, 100)
	g.eventLock = sync.Mutex{}

	go g.WriteEventQueue()
	go g.WriteToServer()

	g.mode = ModeGame
	g.SoundBoard.Play(assets.Spawn)
	log.Printf("Logged in as %v with id [%v] to server %v", g.player.Nick, g.player.ID, address)
}

func (g *Game) Login(e *msgs.EventPlayerLogin) {
	g.sessionID = uint32(e.ID)
	g.world = NewMap(MapConfigFromPlayerLogin(e))
	g.player = player.NewLogin(e)
	g.client = g.player.Client
	for _, p := range e.VisiblePlayers {
		g.players[p.ID] = player.CreateFromLogin(g.player, &p)
		g.players[p.ID].SetSoundboard(g.SoundBoard)
		g.playersY = append(g.playersY, g.players[p.ID])
	}
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
	if ebiten.IsMouseButtonPressed(ebiten.MouseButton2) {
		g.serverAddress = string(clipboard.Read(clipboard.FmtText))
	}
	if strings.HasSuffix(text, "\n") && g.serverAddress != "" {
		g.StartGame(strings.Trim(text, "\n"), g.serverAddress)
	}
}

func (g *Game) WriteToServer() {
	for m := range g.outQueue {
		g.ms.EncodeAndWrite(m.E, m.Data)
	}
}

func (g *Game) WriteEventQueue() {
	for {
		im, err := g.ms.Read()
		if err != nil {
			g.eventLock.Lock()
			g.eventQueue = append(g.eventQueue, &GameMsg{E: msgs.EServerDisconnect})
			g.eventLock.Unlock()
			return
		}
		dim := GameMsg{E: im.Event}
		switch im.Event {
		case msgs.EPing:
		case msgs.EMoveOk:
			dim.Data = im.Data
		case msgs.EMeleeOk:
			dim.Data = msgs.DecodeEventMeleeOk(im.Data)
		case msgs.ECastSpellOk:
			dim.Data = msgs.DecodeEventCastSpellOk(im.Data)
		case msgs.EUseItemOk:
			dim.Data = msgs.DecodeEventUseItemOk(im.Data)
		case msgs.EPlayerMoved:
			dim.Data = msgs.DecodeEventPlayerMoved(im.Data)
		case msgs.EPlayerMelee:
			dim.Data = msgs.DecodeEventPlayerMelee(im.Data)
		case msgs.EPlayerSpell:
			dim.Data = msgs.DecodeEventPlayerSpell(im.Data)
		case msgs.EPlayerMeleeRecieved:
			dim.Data = msgs.DecodeEventPlayerMeleeRecieved(im.Data)
		case msgs.EPlayerSpellRecieved:
			dim.Data = msgs.DecodeEventPlayerSpellRecieved(im.Data)
		case msgs.EPlayerSpawned:
			msg := &msgs.EventPlayerSpawned{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.EPlayerLeaveViewport:
			dim.Data = binary.BigEndian.Uint16(im.Data)
		case msgs.EPlayerEnterViewport:
			msg := &msgs.EventPlayerSpawned{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.EPlayerDespawned:
			dim.Data = binary.BigEndian.Uint16(im.Data)
		}
		g.eventLock.Lock()
		g.eventQueue = append(g.eventQueue, &dim)
		g.eventLock.Unlock()
	}
}

func (g *Game) Clear() {
	g.sessionID = 0
	g.player = nil
	g.players = [50]*player.P{}
	g.playersY = []YSortable{}
}

func (g *Game) ProcessEventQueue() error {
	g.eventLock.Lock()
	for _, ev := range g.eventQueue {
		switch ev.E {
		case msgs.EServerDisconnect:
			log.Printf("Server disconnected\n")
			return errors.New("server disconnected")
		case msgs.EPing:
			g.WaitingPong = false
			g.latency = fmt.Sprintf("%dms", time.Since(g.LastPing).Milliseconds())
		case msgs.EPlayerDespawned:
			pid := ev.Data.(uint16)
			g.DespawnPlayer(pid)
			log.Printf("Player [%v] despawned\n", pid)
		case msgs.EPlayerSpawned:
			event := ev.Data.(*msgs.EventPlayerSpawned)
			log.Printf("Player [%v] spawned %v\n", event.ID, event.Nick)
			g.players[event.ID] = player.CreatePlayerSpawned(g.player, event)
			g.players[event.ID].SetSoundboard(g.SoundBoard)
			g.playersY = append(g.playersY, g.players[event.ID])
		case msgs.EPlayerLeaveViewport:
			pid := ev.Data.(uint16)
			g.DespawnPlayer(pid)
			log.Printf("Player [%v] left viewport\n", pid)
		case msgs.EPlayerEnterViewport:
			event := ev.Data.(*msgs.EventPlayerSpawned)
			log.Printf("Player [%v] entered viewport %v\n", event.ID, event.Nick)
			g.players[event.ID] = player.CreatePlayerSpawned(g.player, event)
			g.players[event.ID].SetSoundboard(g.SoundBoard)
			g.playersY = append(g.playersY, g.players[event.ID])
		case msgs.EPlayerMoved:
			event := ev.Data.(*msgs.EventPlayerMoved)
			log.Printf("Player [%v] moved\n", event.ID)
			g.players[event.ID].AddStep(event)
		case msgs.EMoveOk:
			data := ev.Data.([]byte)
			allowed := data[0] != 0
			g.player.Direction = direction.D(data[1])
			log.Printf("Move: %v in %vms, dir: %v\n", allowed, time.Since(g.moveStartedAt).Milliseconds(), direction.S(g.lastDir))
			g.lastMoveOkArrived = true
			if allowed {
				g.lastDir = direction.D(data[1])
				fromx, fromy := g.player.X, g.player.Y
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
				g.world.Space.Move(0,
					typ.P{X: int32(fromx), Y: int32(fromy)},
					typ.P{X: int32(g.player.X), Y: int32(g.player.Y)})
				if g.player.Dead {
					continue
				}
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
			if !allowed && g.leftForMove != 0 {
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
			}

		case msgs.EMeleeOk:
			event := ev.Data.(*msgs.EventMeleeOk)
			log.Printf("CastMeleeOk m: %#v\n", event)
			if g.player.Dead {
				break
			}
			if !event.Hit {
				g.SoundBoard.Play(assets.MeleeAir)
				break
			}
			g.SoundBoard.Play(assets.MeleeBlood)
			g.player.Effect.NewAttackNumber(int(event.Damage))
			g.players[event.ID].Effect.NewMeleeHit()
			g.players[event.ID].Dead = event.Killed
		case msgs.EPlayerMeleeRecieved:
			event := ev.Data.(*msgs.EventPlayerMeleeRecieved)
			g.SoundBoard.Play(assets.MeleeBlood)
			g.player.Effect.NewMeleeHit()
			log.Printf("RecivedMelee m: %#v\n", event)
			g.players[event.ID].Effect.NewAttackNumber(int(event.Damage))
			g.player.Client.HP = int(event.NewHP)
			if g.player.Client.HP == 0 {
				g.player.Dead = true
			}
		case msgs.EPlayerMelee:
			event := ev.Data.(*msgs.EventPlayerMelee)
			log.Printf("EPlayerMelee m: %#v\n", event)
			if !event.Hit {
				g.SoundBoard.PlayFrom(assets.MeleeAir, g.player.X, g.player.Y, g.players[event.From].X, g.players[event.From].Y)
				break
			}
			g.SoundBoard.PlayFrom(assets.MeleeBlood, g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
			g.players[event.ID].Effect.NewMeleeHit()
			g.players[event.ID].Dead = event.Killed
		case msgs.ECastSpellOk:
			event := ev.Data.(*msgs.EventCastSpellOk)
			log.Printf("CastSpellOk m: %#v\n", event)
			g.player.Client.MP = int(event.NewMP)
			if uint32(event.ID) != g.sessionID {
				g.player.Effect.NewAttackNumber(int(event.Damage))
				g.players[event.ID].Effect.NewSpellHit(event.Spell)
				g.SoundBoard.PlayFrom(assets.SoundFromSpell(event.Spell), g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
				g.players[event.ID].Dead = event.Killed
			}
		case msgs.EPlayerSpellRecieved:
			event := ev.Data.(*msgs.EventPlayerSpellRecieved)
			log.Printf("RecivedSpell m: %#v\n", event)
			switch event.Spell {
			case spell.Paralize:
				g.player.Inmobilized = true
			case spell.RemoveParalize:
				g.player.Inmobilized = false
			case spell.Revive:
				g.player.Dead = false
			}
			caster := g.players[event.ID]
			if event.ID == uint16(g.sessionID) {
				caster = g.player
			}
			g.SoundBoard.Play(assets.SoundFromSpell(event.Spell))
			caster.Effect.NewAttackNumber(int(event.Damage))
			g.player.Effect.NewSpellHit(event.Spell)
			g.player.Client.HP = int(event.NewHP)
			if g.player.Client.HP == 0 {
				g.player.Dead = true
			}
		case msgs.EPlayerSpell:
			event := ev.Data.(*msgs.EventPlayerSpell)
			log.Printf("SpellHit m: %#v\n", event)
			g.SoundBoard.PlayFrom(assets.SoundFromSpell(event.Spell), g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
			g.players[event.ID].Effect.NewSpellHit(event.Spell)
			g.players[event.ID].Dead = event.Killed
		case msgs.EUseItemOk:
			event := ev.Data.(*msgs.EventUseItemOk)
			log.Printf("UsePotionOk m: %#v\n", event)
			switch event.Item {
			case msgs.Item(server.ItemManaPotion):
				g.player.Client.MP = int(event.Change)
			case msgs.Item(server.ItemHealthPotion):
				g.player.Client.HP = int(event.Change)
			}
			g.SoundBoard.Play(assets.Potion)
		}
	}
	g.eventQueue = g.eventQueue[:0]
	g.eventLock.Unlock()
	return nil
}

func (g *Game) pingServer() {
	if g.counter%120 != 0 || g.WaitingPong {
		return
	}
	g.outQueue <- &GameMsg{E: msgs.EPing}
	g.WaitingPong = true
	g.LastPing = time.Now()
}

func (g *Game) DespawnPlayer(pid uint16) {
	p := g.players[pid]
	g.players[pid] = nil
	if !p.Dead {
		g.world.Space.Set(0, typ.P{X: int32(p.X), Y: int32(p.Y)}, 0)
	}
	for iy, ys := range g.playersY {
		if py, ok := ys.(*player.P); ok && pid == uint16(py.ID) {
			g.playersY[iy] = g.playersY[len(g.playersY)-1]
			g.playersY = g.playersY[:len(g.playersY)-1]
			break
		}
	}
}

func (g *Game) updateGame() error {
	if err := g.ProcessEventQueue(); err != nil {
		g.Clear()
		g.typer = typing.NewTyper()
		g.mode = ModeRegister
		return err
	}
	g.pingServer()
	g.ProcessMovement()
	g.ProcessCombat()
	g.world.Update()
	g.player.Update(g.counter)
	g.player.Effect.Update(g.counter)
	for _, p := range g.players {
		if p == nil {
			continue
		}
		p.WalkSteps(g.world.Space)
		p.Update(g.counter)
		p.Effect.Update(g.counter)
	}
	sort.Slice(g.playersY, func(i, j int) bool {
		return g.playersY[i].ValueY() < g.playersY[j].ValueY()
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
		g.outQueue <- &GameMsg{E: msgs.EMelee}
	}
	if ok, spellType, x, y := g.combatKeys.CastSpell(); ok {
		worldX, worldY := g.ScreenToWorld(x, y)
		g.outQueue <- &GameMsg{E: msgs.ECastSpell, Data: &msgs.EventCastSpell{
			PX:    uint32(worldX),
			PY:    uint32(worldY),
			Spell: spellType,
		}}
	}
	if pressedPotion := g.combatKeys.PressedPotion(); pressedPotion != potion.None {
		g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: pressedPotion}
	}
}

func DirToNewPos(p *player.P, d direction.D) (int, int) {
	switch d {
	case direction.Front:
		return p.X, p.Y + 1
	case direction.Back:
		return p.X, p.Y - 1
	case direction.Left:
		return p.X - 1, p.Y
	case direction.Right:
		return p.X + 1, p.Y
	}
	return 0, 0
}

func (g *Game) ProcessMovement() {
	//if g.counter%2 == 0 {
	if g.leftForMove > 0 {
		vel := g.player.MoveSpeed
		if g.leftForMove <= vel {
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
	//}

	g.keys.ListenMovement()
	d := g.keys.MovingTo()

	if g.leftForMove == 0 && g.lastMoveOkArrived && d != direction.Still && time.Since(g.startForStep).Milliseconds() > 50 {
		g.startForStep = time.Now()
		g.player.Direction = d
		x, y := DirToNewPos(g.player, d)
		if x < 0 || x >= constants.WorldX || y < 0 || y >= constants.WorldY {
			return
		}
		stuffLayer := g.world.Space.GetSlot(1, typ.P{X: int32(x), Y: int32(y)})
		if stuffLayer != 0 {
			if g.player.Direction != g.lastDir {
				g.moveStartedAt = time.Now()
				g.outQueue <- &GameMsg{E: msgs.EMove, Data: d}
			}
			return
		}

		if g.player.Inmobilized {
			g.moveStartedAt = time.Now()
			g.outQueue <- &GameMsg{E: msgs.EMove, Data: d}
			return
		}

		g.moveStartedAt = time.Now()
		g.outQueue <- &GameMsg{E: msgs.EMove, Data: d}
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
	ebitenutil.DebugPrintAt(screen, "Right click to paste an IP", HalfScreenX-80, HalfScreenY/2-30)
	ebitenutil.DebugPrintAt(screen, g.serverAddress, HalfScreenX-40, HalfScreenY/2)
	ebitenutil.DebugPrintAt(screen, "Type a nickname and press ENTER", HalfScreenX-80, HalfScreenY-30)
	g.typer.Draw(screen, HalfScreenX, HalfScreenY)
}

func (g *Game) drawGame(screen *ebiten.Image) {
	g.world.Draw()
	for _, p := range g.playersY {
		if p == nil {
			continue
		}
		p.Draw(g.world.Image())
	}
	g.Render(g.world.Image(), screen)

	ebitenutil.DebugPrint(screen,
		fmt.Sprintf("FPS: %v\nTPS: %v\nPing: %v", int(ebiten.ActualFPS()), math.Round(ebiten.ActualTPS()), g.latency))

	g.combatKeys.ShowSpellPicker(screen)
	g.stats.Draw(screen)
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
