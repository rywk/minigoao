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
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
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
	mode Mode
	// register stuff
	serverAddress string
	typer         *typing.Typer
	fsBtn         *Checkbox
	inputBox      *ebiten.Image
	fullscreen    bool

	// game
	latency    string
	counter    int
	ms         *msgs.M
	world      *Map
	sessionID  uint32
	players    [constants.MaxConnCount]*player.P
	playersY   []YSortable
	player     *player.P
	outQueue   chan *GameMsg
	eventQueue []*GameMsg
	eventLock  sync.Mutex

	client *player.ClientP
	stats  *Hud

	keys *Keys

	lastMove          time.Time
	leftForMove       float64 // pixels left to complete tile change
	lastDir           direction.D
	soundPrevWalk     int
	startForStep      time.Time
	lastMoveConfirmed bool

	lastPotion     time.Time
	lastPotionUsed msgs.Item

	steps []player.Step

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

const ScreenWidth, ScreenHeight = 1280, 720

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
	// err := clipboard.Init()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	g.fsBtn = NewCheckbox(typ.P{X: HalfScreenX + 30, Y: ScreenHeight - ScreenHeight/6},
		texture.Decode(img.CheckboxOn_png), texture.Decode(img.CheckboxOff_png))
	g.inputBox = texture.Decode(img.InputBox_png)
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

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.mode {
	case ModeRegister:
		g.drawRegister(screen)
	case ModeGame:
		g.drawGame(screen)
	case ModeOptions:
	}
}

func (g *Game) updateRegister() {
	g.fsBtn.Update()
	g.fullscreen = g.fsBtn.On
	g.typer.Update()
	text := g.typer.String()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButton2) {
		g.serverAddress = "127.0.0.1:28441" // string(clipboard.Read(clipboard.FmtText))
	}
	if !strings.HasSuffix(text, "\n") {
		return
	}
	if g.serverAddress == "" {
		g.typer = typing.NewTyper(text[:len(text)-1])
		return
	}
	nick := strings.Trim(strings.Trim(text, "\n"), " ")
	if nick == "" {
		g.typer = typing.NewTyper(text[:len(text)-1])
		return
	}
	g.StartGame(nick, g.serverAddress)
}

func (g *Game) drawRegister(screen *ebiten.Image) {
	text.PrintBigAt(screen, "Right click to paste an IP", HalfScreenX-150, HalfScreenY/2-40)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(HalfScreenX-150, HalfScreenY/2-5)
	screen.DrawImage(g.inputBox, op)
	text.PrintBigAt(screen, g.serverAddress, HalfScreenX-130, HalfScreenY/2+7)
	text.PrintBigAt(screen, "Type a nickname and press ENTER", HalfScreenX-180, HalfScreenY-45)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(HalfScreenX-150, HalfScreenY-5)
	screen.DrawImage(g.inputBox, op)
	g.typer.Draw(screen, HalfScreenX-130, HalfScreenY+7)

	text.PrintBigAt(screen, "Fullscreen", HalfScreenX-110, ScreenHeight-ScreenHeight/6)
	g.fsBtn.Draw(screen)
}

func (g *Game) drawGame(screen *ebiten.Image) {
	g.world.Draw(typ.P{X: g.player.X, Y: g.player.Y})
	for _, p := range g.playersY {
		if p == nil {
			continue
		}
		p.Draw(g.world.Image())
	}
	g.keys.DrawChat(g.world.Image(), int(g.player.Pos[0]+16), int(g.player.Pos[1]-36))
	g.Render(g.world.Image(), screen)

	ebitenutil.DebugPrint(screen,
		fmt.Sprintf("FPS: %v\nTPS: %v\nPing: %v", int(ebiten.ActualFPS()), math.Round(ebiten.ActualTPS()), g.latency))

	g.stats.Draw(screen)
}

func (g *Game) updateGame() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		g.Clear()
		g.typer = typing.NewTyper()
		g.mode = ModeRegister
		ebiten.SetFullscreen(false)
		g.ms.Close()
		return errors.New("esc exit")
	}
	if err := g.ProcessEventQueue(); err != nil {
		g.Clear()
		g.typer = typing.NewTyper()
		g.mode = ModeRegister
		ebiten.SetFullscreen(false)
		return err
	}
	g.pingServer()
	g.SendChat()
	g.ListenInputs()
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
	g.counter++
	return nil
}

func (g *Game) StartGame(nick string, address string) {
	ebiten.SetFullscreen(g.fullscreen)
	g.SoundBoard = audio2d.NewSoundBoard()
	g.ViewPort = f64.Vec2{ScreenWidth, ScreenHeight}
	g.ZoomFactor = 1
	g.lastMove = time.Now()
	g.lastMoveConfirmed = true
	g.keys = NewKeys(g, nil)

	tcp, err := net.Dial("tcp4", address)
	if err != nil {
		log.Fatal(err)
	}
	g.ms = msgs.New(tcp)
	register := &msgs.EventRegister{
		Nick: nick,
	}
	err = g.ms.EncodeAndWrite(msgs.ERegister, register)
	if err != nil {
		panic(err)
	}
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
	g.stats = NewHud(g)

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
		g.AddToGame(&p)
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
		case msgs.EBroadcastChat:
			msg := &msgs.EventBroadcastChat{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
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

func (g *Game) AddToGame(event *msgs.EventNewPlayer) {
	g.world.Space.Set(0, event.Pos, event.ID)
	g.players[event.ID] = player.CreatePlayerSpawned(g.player, event)
	g.players[event.ID].SetSoundboard(g.SoundBoard)
	g.playersY = append(g.playersY, g.players[event.ID])
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
			g.AddToGame(event)
		case msgs.EPlayerLeaveViewport:
			pid := ev.Data.(uint16)
			g.DespawnPlayer(pid)
			log.Printf("Player [%v] left viewport\n", pid)
		case msgs.EPlayerEnterViewport:
			event := ev.Data.(*msgs.EventPlayerSpawned)
			log.Printf("Player [%v] entered viewport %v\n", event.ID, event.Nick)
			g.AddToGame(event)
		case msgs.EPlayerMoved:
			event := ev.Data.(*msgs.EventPlayerMoved)
			log.Printf("Player [%v] moved\n", event.ID)
			g.players[event.ID].AddStep(event)
		case msgs.EMoveOk:
			g.MovementResponse(ev.Data.([]byte))
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
			case spell.Resurrect:
				g.player.Dead = false
			}
			caster := g.players[event.ID]
			if event.ID == uint16(g.sessionID) {
				caster = g.player
			}
			g.SoundBoard.Play(assets.SoundFromSpell(event.Spell))
			g.player.Effect.NewSpellHit(event.Spell)
			caster.Effect.NewAttackNumber(int(event.Damage))
			g.player.Client.HP = int(event.NewHP)
			if g.player.Client.HP == 0 {
				g.player.Inmobilized = false
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
			case msgs.Item(msgs.ItemManaPotion):
				g.player.Client.MP = int(event.Change)
			case msgs.Item(msgs.ItemHealthPotion):
				g.player.Client.HP = int(event.Change)
			}
			g.stats.potionAlpha = 1
			g.lastPotionUsed = event.Item
			g.SoundBoard.Play(assets.Potion)
		case msgs.EBroadcastChat:
			event := ev.Data.(*msgs.EventBroadcastChat)
			g.players[event.ID].SetChatMsg(event.Msg)
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
func (g *Game) SendChat() {
	if msg := g.keys.ChatMessage(); msg != "" {
		g.outQueue <- &GameMsg{
			E:    msgs.ESendChat,
			Data: &msgs.EventSendChat{Msg: msg},
		}
	}
}

func (g *Game) ProcessCombat() {
	if g.keys.MeleeHit() {
		g.outQueue <- &GameMsg{E: msgs.EMelee}
	}
	if ok, spellType, x, y := g.keys.CastSpell(); ok {
		worldX, worldY := g.ScreenToWorld(x, y)
		g.outQueue <- &GameMsg{E: msgs.ECastSpell, Data: &msgs.EventCastSpell{
			PX:    uint32(worldX),
			PY:    uint32(worldY),
			Spell: spellType,
		}}
	}
	if pressedPotion := g.keys.PressedPotion(); pressedPotion != msgs.ItemNone {
		g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: pressedPotion}
	}
}

func DirToNewPos(p *player.P, d direction.D) typ.P {
	switch d {
	case direction.Front:
		return typ.P{X: p.X, Y: p.Y + 1}
	case direction.Back:
		return typ.P{X: p.X, Y: p.Y - 1}
	case direction.Left:
		return typ.P{X: p.X - 1, Y: p.Y}
	case direction.Right:
		return typ.P{X: p.X + 1, Y: p.Y}
	}
	return typ.P{X: p.X, Y: p.Y}
}

func (g *Game) MovementResponse(data []byte) {
	g.lastMoveConfirmed = true
	allowed := data[0] != 0
	dir := direction.D(data[1])
	if len(g.steps) == 0 {
		g.player.Walking = false
		log.Printf("move ok arrived without having a step sent\n")
		return
	}
	step := g.steps[0]
	g.steps = g.steps[1:]
	log.Printf("MOVE:[%v][%v] %v %vms\n", allowed, step.Expect, direction.S(g.player.Direction), time.Since(g.startForStep).Milliseconds())

	if allowed == step.Expect {
		// if it was expected, we already did it
		return
	}
	if !allowed && step.Expect {
		// if it wasnt expected to fail we need to go back
		goBackPx := float64(constants.TileSize - g.leftForMove)
		log.Println("move not allowed, go back", goBackPx, "px", *g.player)
		switch dir {
		case direction.Front:
			g.player.Y--
			g.player.Pos[1] -= goBackPx
		case direction.Back:
			g.player.Y++
			g.player.Pos[1] += goBackPx
		case direction.Left:
			g.player.X++
			g.player.Pos[0] += goBackPx
		case direction.Right:
			g.player.X--
			g.player.Pos[0] -= goBackPx
		}
		g.player.Walking = false
		g.leftForMove = 0
		return
	}

	if allowed && !step.Expect {
		g.player.Inmobilized = false // if were allowed to move then we can change this already in any case
		g.lastDir = dir
		g.Move(dir, step.To)
		return
	}

}

func (g *Game) TryMove(d direction.D, np typ.P, confident bool) {
	g.lastDir = d
	g.lastMoveConfirmed = false
	g.outQueue <- &GameMsg{E: msgs.EMove, Data: d}
	g.steps = append(g.steps, player.Step{
		To:     np,
		Dir:    d,
		Expect: confident,
	})
	if confident {
		g.Move(d, np)
	}
}

func (g *Game) Move(d direction.D, np typ.P) {
	g.leftForMove = constants.TileSize
	g.player.Walking = true
	g.player.X = np.X
	g.player.Y = np.Y
	if g.player.Dead {
		return
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

func (g *Game) AddGameStep(d direction.D) {
	g.startForStep = time.Now()
	g.player.Direction = d
	np := DirToNewPos(g.player, d)
	outWorld := np.Out(g.world.Space.Rect)
	if outWorld {
		return
	}
	stuff := g.world.Space.GetSlot(1, np)
	if stuff != 0 && d == g.lastDir {
		return
	}

	playerLayer := g.world.Space.GetSlot(0, np)

	confident := !outWorld && !g.player.Inmobilized && playerLayer == 0 && stuff == 0

	g.TryMove(d, np, confident)
}

func (g *Game) ListenInputs() {
	g.keys.ListenMovement()
	g.keys.ListenSpell()
}

func (g *Game) ProcessMovement() {
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
	d := g.keys.MovingTo()
	if g.lastMoveConfirmed && g.leftForMove == 0 && d != direction.Still && time.Since(g.startForStep).Milliseconds() > 50 {
		g.AddGameStep(d)
	}
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
	m.Translate(-g.player.Pos[0]+HalfScreenX-16, -g.player.Pos[1]+HalfScreenY-48)
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
