package game

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/conc"
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

type Login struct {
	data *msgs.EventPlayerLogin
	err  error
}

type Game struct {
	secureConn   bool
	debug        bool
	start        time.Time
	memStats     *runtime.MemStats
	totalRunTime time.Duration

	// reduce mem ftp
	worldImgOp *ebiten.DrawImageOptions

	web  bool
	mode Mode

	// register stuff
	connecting                 bool
	connected                  chan Login
	adressEnteringPasteTooltip string
	serverAddress              string
	serverTyper                *typing.Typer
	typingServer               bool
	nickTyper                  *typing.Typer
	fsBtn                      *Checkbox
	vsyncBtn                   *Checkbox
	inputBox                   *ebiten.Image
	fullscreen                 bool
	vsync                      bool
	connErrorColorStart        int

	// game
	mouseX, mouseY int
	latency        string
	onlines        string
	counter        int
	ms             msgs.Msgs
	world          *Map
	sessionID      uint32
	players        [constants.MaxConnCount]*player.P
	playersY       []YSortable
	player         *player.P
	outQueue       chan *GameMsg
	eventQueue     []*GameMsg
	eventLock      sync.Mutex

	client *player.ClientP
	stats  *Hud

	keys *Keys

	dirOverwriteAttempt            bool
	dirOverwriteAttemptLeftForMove float64

	lastMove          time.Time
	leftForMove       float64 // pixels left to complete tile change
	lastDir           direction.D
	soundPrevWalk     int
	startForStep      time.Time
	lastMoveConfirmed bool

	lastPotionUsed msgs.Item

	steps []player.Step

	SoundBoard audio2d.AudioMixer

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

func NewGame(web bool, serverAddr string) *Game {
	start := time.Now()

	g := &Game{
		secureConn:  false,
		debug:       false,
		start:       start,
		memStats:    &runtime.MemStats{},
		vsync:       true,
		mode:        ModeRegister,
		web:         web,
		connected:   make(chan Login),
		nickTyper:   typing.NewTyper(),
		serverTyper: typing.NewTyper(serverAddr),
		worldImgOp:  &ebiten.DrawImageOptions{},
		inputBox:    texture.Decode(img.InputBox_png),
	}
	if strings.Split(serverAddr, ":")[1] == "443" {
		g.secureConn = true
	}
	g.fsBtn = NewCheckbox(g)
	g.vsyncBtn = NewCheckbox(g)
	g.vsyncBtn.On = false
	g.SoundBoard = audio2d.NewSoundBoard(web)
	return g
}

func (g *Game) Update() error {
	switch g.mode {
	case ModeRegister:
		g.updateRegister()
	case ModeGame:
		g.updateGame()
	case ModeOptions:
	}
	if !g.debug {
		return nil
	}
	if g.counter%30 == 0 {
		runtime.ReadMemStats(g.memStats)
		g.totalRunTime = time.Since(g.start)
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
	if !g.debug {
		return
	}
	ms := g.memStats
	msg := fmt.Sprintf(`TPS: %0.2f (max: %d);
Run time: %v
ticks: %d
Alloc: %s
Total: %s
Sys: %s
NextGC: %s
NumGC: %d`,
		ebiten.ActualTPS(), ebiten.TPS(),
		g.totalRunTime,
		g.counter,
		formatBytes(ms.Alloc), formatBytes(ms.TotalAlloc), formatBytes(ms.Sys),
		formatBytes(ms.NextGC), ms.NumGC,
	)
	ebitenutil.DebugPrintAt(screen, msg, 20, 64)
}

func formatBytes(b uint64) string {
	if b >= 1073741824 {
		return fmt.Sprintf("%0.2f GiB", float64(b)/1073741824)
	} else if b >= 1048576 {
		return fmt.Sprintf("%0.2f MiB", float64(b)/1048576)
	} else if b >= 1024 {
		return fmt.Sprintf("%0.2f KiB", float64(b)/1024)
	} else {
		return fmt.Sprintf("%d B", b)
	}
}
func (g *Game) updateRegister() {
	if g.connErrorColorStart > 0 {
		g.connErrorColorStart--
	}
	g.fullscreen = g.fsBtn.On
	g.fsBtn.Update()
	// g.vsync = g.vsyncBtn.On
	// g.vsyncBtn.Update()
	if g.typingServer {
		g.serverTyper.Update()
	} else {
		g.nickTyper.Update()
	}

	if g.connecting {
		login, ok := conc.Check(g.connected)
		if !ok {
			return
		}
		g.connecting = false
		if login.err != nil {
			log.Println(login.err)
			g.connErrorColorStart = 255
			return
		}
		g.StartGame(login.data)
		return
	}

	r := strings.NewReplacer("\n", "", " ", "")
	nickText := g.nickTyper.String()
	addressText := g.serverTyper.String()
	if !strings.HasSuffix(nickText, "\n") && !strings.HasSuffix(addressText, "\n") {
		g.nickTyper.Text, g.serverTyper.Text = r.Replace(nickText), r.Replace(addressText)
		return
	}
	g.nickTyper.Text, g.serverTyper.Text = r.Replace(nickText), r.Replace(addressText)
	if g.nickTyper.Text != "" && g.serverTyper.Text != "" {
		g.connecting = true
		go g.Connect(g.nickTyper.Text, g.serverTyper.Text)
	}
}

func (g *Game) drawRegister(screen *ebiten.Image) {
	g.mouseX, g.mouseY = ebiten.CursorPosition()
	text.PrintBigAt(screen, "Nick", HalfScreenX-144, HalfScreenY-95)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(HalfScreenX-150, HalfScreenY-55)
	screen.DrawImage(g.inputBox, op)
	g.nickTyper.Draw(screen, HalfScreenX-130, HalfScreenY-42)
	text.PrintBigAt(screen, "Fullscreen", HalfScreenX-95, HalfScreenY+93)
	g.fsBtn.Draw(screen, HalfScreenX+46, HalfScreenY+92)
	// text.PrintBigAt(screen, "Vsync", HalfScreenX-95, HalfScreenY+135)
	// g.vsyncBtn.Draw(screen, HalfScreenX+46, HalfScreenY+132)
	if g.connErrorColorStart > 0 {
		text.PrintBigAtCol(screen, "Server offline", HalfScreenX-90, HalfScreenY+5, color.RGBA{178, 0, 16, uint8(g.connErrorColorStart)})
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {

	g.mouseX, g.mouseY = ebiten.CursorPosition()
	g.world.Draw(typ.P{X: g.player.X, Y: g.player.Y})
	for _, p := range g.playersY {
		if p == nil {
			continue
		}
		p.Draw(g.world.Image())
	}
	g.keys.DrawChat(g.world.Image(), int(g.player.Pos[0]+16), int(g.player.Pos[1]-40))
	g.Render(g.world.Image(), screen)
	g.stats.Draw(screen)
	text.PrintAt(screen, fmt.Sprintf("%vFPS\n%v", int(ebiten.ActualFPS()), g.latency), 0, 0)
	text.PrintAt(screen, fmt.Sprintf("Online: %v", g.onlines), 50, 0)
}

func (g *Game) updateGame() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.Clear()
		g.nickTyper = typing.NewTyper()
		g.mode = ModeRegister
		ebiten.SetFullscreen(false)
		g.ms.Close()
		return errors.New("esc exit")
	}
	if err := g.ProcessEventQueue(); err != nil {
		g.Clear()
		g.nickTyper = typing.NewTyper()
		g.mode = ModeRegister
		ebiten.SetFullscreen(false)
		return err
	}
	g.pingServer()
	g.SendChat()
	g.UpdateGamePos()
	g.ListenInputs()
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

func (g *Game) Connect(nick string, address string) {
	var err error
	g.ms, err = msgs.DialServer(address, g.web, g.secureConn)
	if err != nil {
		g.connected <- Login{data: nil, err: err}
		return
	}
	register := &msgs.EventRegister{
		Nick: nick,
	}
	err = g.ms.EncodeAndWrite(msgs.ERegister, register)
	if err != nil {
		g.connected <- Login{data: nil, err: err}
		return
	}
	im, err := g.ms.Read()
	if err != nil {
		g.connected <- Login{data: nil, err: err}
		return
	}
	if im.Event != msgs.EPlayerLogin {
		g.connected <- Login{data: nil, err: fmt.Errorf("not login response")}
		return
	}
	g.connected <- Login{data: msgs.DecodeMsgpack(im.Data, &msgs.EventPlayerLogin{}), err: nil}
}

func (g *Game) StartGame(login *msgs.EventPlayerLogin) {
	if g.web {
		g.vsync = true
	}
	ebiten.SetVsyncEnabled(g.vsync)
	ebiten.SetFullscreen(g.fullscreen)
	g.mode = ModeGame
	g.Login(login)
	g.ViewPort = f64.Vec2{ScreenWidth, ScreenHeight}
	//g.ZoomFactor = 1
	g.lastMove = time.Now()
	g.lastMoveConfirmed = true
	g.keys = NewKeys(g, nil)
	g.keys.enterDown = true
	g.playersY = append(g.playersY, g.player)
	g.stats = NewHud(g)

	g.eventQueue = make([]*GameMsg, 0, 100)
	g.outQueue = make(chan *GameMsg, 100)
	g.eventLock = sync.Mutex{}
	go g.WriteEventQueue()
	go g.WriteToServer()

	g.SoundBoard.Play(assets.Spawn)
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
		case msgs.EPingOk:
			dim.Data = binary.BigEndian.Uint16(im.Data[:2])
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
	g.world.Space.SetSlot(0, event.Pos, event.ID)
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
		case msgs.EPingOk:
			g.WaitingPong = false
			g.onlines = fmt.Sprintf("%d", ev.Data.(uint16))
			g.latency = fmt.Sprintf("%dms", time.Since(g.LastPing).Milliseconds()/2)
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
				g.player.Direction = event.Dir
				break
			}
			g.player.Direction = event.Dir
			g.SoundBoard.Play(assets.MeleeBlood)
			g.player.Effect.NewAttackNumber(int(event.Damage), false)
			g.players[event.ID].Effect.NewMeleeHit()
			g.players[event.ID].Dead = event.Killed
		case msgs.EPlayerMeleeRecieved:
			event := ev.Data.(*msgs.EventPlayerMeleeRecieved)
			g.SoundBoard.Play(assets.MeleeBlood)
			g.player.Effect.NewMeleeHit()
			log.Printf("RecivedMelee m: %#v\n", event)
			g.players[event.ID].Effect.NewAttackNumber(int(event.Damage), false)
			g.players[event.ID].Direction = event.Dir
			g.player.Client.HP = int(event.NewHP)

			if g.player.Client.HP == 0 {
				g.player.Dead = true
				g.player.Inmobilized = false
			}
		case msgs.EPlayerMelee:
			event := ev.Data.(*msgs.EventPlayerMelee)
			log.Printf("EPlayerMelee m: %#v\n", event)
			if !event.Hit {
				g.players[event.From].Direction = event.Dir
				g.SoundBoard.PlayFrom(assets.MeleeAir, g.player.X, g.player.Y, g.players[event.From].X, g.players[event.From].Y)
				break
			}
			g.players[event.ID].Direction = event.Dir
			g.SoundBoard.PlayFrom(assets.MeleeBlood, g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
			g.players[event.ID].Effect.NewMeleeHit()
			g.players[event.ID].Dead = event.Killed
		case msgs.ECastSpellOk:
			event := ev.Data.(*msgs.EventCastSpellOk)
			log.Printf("CastSpellOk m: %#v\n", event)
			g.player.Client.MP = int(event.NewMP)
			if uint32(event.ID) != g.sessionID {
				g.player.Effect.NewAttackNumber(int(event.Damage), event.Spell == spell.HealWounds)
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
			caster.Effect.NewAttackNumber(int(event.Damage), event.Spell == spell.HealWounds)
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
	if g.WaitingPong || g.counter%240 != 0 {
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
	log.Printf("MovementResponse %v  %v\n", allowed, dir)
	if len(g.steps) == 0 {
		g.player.Walking = false
		log.Printf("move ok arrived without having a step sent\n")
		return
	}
	step := g.steps[0]
	g.steps = g.steps[1:]
	//log.Printf("MOVE:[%v][%v] %v %vms\n", allowed, step.Expect, direction.S(g.player.Direction), time.Since(g.startForStep).Milliseconds())

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
	g.leftForMove += constants.TileSize
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
	g.lastDir = d
	g.TryMove(d, np, confident)
}

func (g *Game) UpdateGamePos() {
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
}

func (g *Game) ListenInputs() {
	g.keys.ListenMovement()
	g.keys.ListenSpell()

	d := g.keys.MovingTo()

	if d == direction.Still {
		goto COMBAT
	}
	if g.lastMoveConfirmed && g.leftForMove == 0 && (time.Since(g.startForStep).Milliseconds() > 60) {
		g.AddGameStep(d)
	}
COMBAT:
	{
		if g.keys.MeleeHit() {
			g.outQueue <- &GameMsg{E: msgs.EMelee, Data: d}
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
}

const HalfScreenX, HalfScreenY = ScreenWidth / 2, ScreenHeight / 2

func (g *Game) updateWorldMatrix() {
	g.worldImgOp.GeoM.Reset()
	g.worldImgOp.GeoM.Translate(-g.player.Pos[0]+HalfScreenX-16, -g.player.Pos[1]+HalfScreenY-48)
}

func (g *Game) Render(world, screen *ebiten.Image) {
	g.updateWorldMatrix()

	screen.DrawImage(world, g.worldImgOp)
}

func (g *Game) ScreenToWorld(posX, posY int) (float64, float64) {
	g.updateWorldMatrix()
	if g.worldImgOp.GeoM.IsInvertible() {
		g.worldImgOp.GeoM.Invert()
		return g.worldImgOp.GeoM.Apply(float64(posX), float64(posY))
	} else {
		// When scaling it can happened that matrix is not invertable
		return math.NaN(), math.NaN()
	}
}
