package server

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/server/webpage"
	"github.com/rywk/minigoao/pkg/typ"
)

type Server struct {
	web       *http.Server
	mms       msgs.MMsgs
	mws       msgs.MMsgs
	tcpport   string
	webport   string
	connCount atomic.Int32
	newConn   chan msgs.Msgs
	game      *Game
}

func NewServer(tcpport string, webport string) *Server {
	return &Server{
		tcpport: tcpport,
		webport: webport,
		newConn: make(chan msgs.Msgs, 100),
	}
}

func (s *Server) AcceptTCPConnections() {
	log.Printf("Accepting TCP connections at %v.\n", s.mms.Address())
	for {
		conn, err := s.mms.NewConn()
		if err != nil {
			return
		}
		log.Printf("accepted plain tcp conn\n")
		s.connCount.Add(1)
		s.newConn <- conn
	}
}

func (s *Server) AcceptWSConnections() {
	log.Printf("Accepting WS connections at %v.\n", s.mws.Address())
	for {
		conn, err := s.mws.NewConn()
		if err != nil {
			return
		}
		log.Printf("accepted web socket conn\n")
		s.connCount.Add(1)
		s.newConn <- conn
	}
}

var (
	//go:embed pk_path.txt
	PKPath []byte

	//go:embed cert_path.txt
	CertPath []byte
)

func (s *Server) Start(exposed bool) error {
	PKPath = []byte(strings.Trim(string(PKPath), "\n"))
	CertPath = []byte(strings.Trim(string(CertPath), "\n"))
	log.Println(PKPath)
	log.Println(CertPath)
	address := fmt.Sprintf("127.0.0.1%s", s.tcpport)
	if exposed {
		address = fmt.Sprintf("0.0.0.0%s", s.tcpport)
	}

	var web http.Server
	shutdown := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		log.Printf("Shutting down web...")

		// Received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := web.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Error at server.Shutdown: %v", err)
		}
		close(shutdown)

		<-sigint
		// Hard exit on the second ctrl-c.
		os.Exit(0)
	}()

	mux := http.NewServeMux()
	var upgraderFunc http.HandlerFunc
	s.mws, upgraderFunc = msgs.NewUpgraderMiddleware()
	mux.HandleFunc("/", webpage.Handle(upgraderFunc))
	web.Handler = mux
	web.Addr = s.webport
	go func() {
		if exposed {
			err := web.ListenAndServeTLS(string(CertPath), string(PKPath))
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Error at server.ListenAndServe: %v", err)
			}
		} else {
			err := web.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Error at server.ListenAndServe: %v", err)
			}
		}
	}()

	var err error
	s.mms, err = msgs.ListenTCP(address)
	if err != nil {
		return err
	}
	s.game = &Game{
		newConn:      s.newConn,
		players:      []*Player{{id: 0}}, // no 0 id
		playersIndex: make([]uint16, 0),
		space:        grid.NewGrid(constants.WorldX, constants.WorldY, 2),
		incomingData: make(chan IncomingMsg, 1000),
	}

	go s.AcceptTCPConnections()
	go s.AcceptWSConnections()

	go s.game.Run()

	<-shutdown

	return nil
}

type Game struct {
	newConn      chan msgs.Msgs
	players      []*Player
	playersIndex []uint16
	space        *grid.Grid
	incomingData chan IncomingMsg
}

type IncomingMsg struct {
	ID    uint16
	Event msgs.E
	Data  interface{}
}

func (g *Game) AddPlayer(p *Player) {
	g.players = append(g.players, p)
	p.id = uint16(len(g.players) - 1)
	g.playersIndex = append(g.playersIndex, p.id)
}

func (g *Game) RemovePlayer(pid uint16) {
	index := -1
	for i, id := range g.playersIndex {
		if pid == id {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}
	g.players[pid] = nil
	g.playersIndex[index] = g.playersIndex[len(g.playersIndex)-1]
	g.playersIndex = g.playersIndex[:len(g.playersIndex)-1]
}

func getNick(m msgs.Msgs) (string, error) {
	im, err := m.Read()
	if err != nil {
		return "", err
	}
	if im.Event != msgs.ERegister {
		return "", errors.New("bad message")
	}
	reg := &msgs.EventRegister{}
	msgs.DecodeMsgpack(im.Data, reg)
	return reg.Nick, nil
}

func GetNick(m msgs.Msgs) (nick string, err error) {
	timeout := time.NewTicker(time.Second).C
	done := make(chan struct{})
	go func() {
		nick, err = getNick(m)
		done <- struct{}{}
	}()
	select {
	case <-timeout:
		return "", errors.New("nick timeout")
	case <-done:
		return nick, err
	}
}

func (g *Game) HandleLogin() {
	log.Printf("Login handler started.\n")
	for conn := range g.newConn {

		p := &Player{
			g:             g,
			m:             conn,
			pos:           typ.P{X: constants.WorldX / 2, Y: constants.WorldY / 2},
			Send:          make(chan OutMsg, 100),
			dir:           direction.Front,
			speedPxXFrame: 3,
			speedXTile:    (constants.TileSize / 3) * AverageGameFrame,
			hp:            372,
			maxHp:         372,
			mp:            2420,
			maxMp:         2420,
		}

		log.Printf("player created waiting for nick\n")
		nick, err := GetNick(p.m)
		if err != nil {
			log.Printf("Get nick error: %v\n", err)
			conn.Close()
			continue
		}
		log.Printf("got nick [%v]\n", nick)
		p.nick = nick
		g.incomingData <- IncomingMsg{
			Event: msgs.EPlayerConnect,
			Data:  p,
		}
	}
}

func (g *Game) AddObjectsToSpace() {
	arena1v1 := image.Rect(0, 0, 8, 8)
	arena2v2 := image.Rect(0, 0, 16, 12)

	arena1v1n1 := arena1v1.Add(image.Point{X: 25, Y: 29})
	arena2v2n1 := arena2v2.Add(image.Point{X: 40, Y: 29})

	for y := arena1v1n1.Min.Y; y < arena1v1n1.Max.Y; y++ {
		g.space.Set(1, typ.P{X: int32(arena1v1n1.Min.X), Y: int32(y)}, uint16(assets.Shroom))
		g.space.Set(1, typ.P{X: int32(arena1v1n1.Max.X), Y: int32(y)}, uint16(assets.Shroom))
	}
	for x := arena1v1n1.Min.X; x < arena1v1n1.Max.X; x++ {
		g.space.Set(1, typ.P{X: int32(x), Y: int32(arena1v1n1.Min.Y)}, uint16(assets.Shroom))
		g.space.Set(1, typ.P{X: int32(x), Y: int32(arena1v1n1.Max.Y)}, uint16(assets.Shroom))
	}
	g.space.Unset(1, typ.P{X: 32, Y: 37})

	for y := arena2v2n1.Min.Y; y < arena2v2n1.Max.Y; y++ {
		g.space.Set(1, typ.P{X: int32(arena2v2n1.Min.X), Y: int32(y)}, uint16(assets.Shroom))
		g.space.Set(1, typ.P{X: int32(arena2v2n1.Max.X), Y: int32(y)}, uint16(assets.Shroom))
	}
	for x := arena2v2n1.Min.X; x < arena2v2n1.Max.X; x++ {
		g.space.Set(1, typ.P{X: int32(x), Y: int32(arena2v2n1.Min.Y)}, uint16(assets.Shroom))
		g.space.Set(1, typ.P{X: int32(x), Y: int32(arena2v2n1.Max.Y)}, uint16(assets.Shroom))
	}
	g.space.Unset(1, typ.P{X: 55, Y: 41})
	g.space.Unset(1, typ.P{X: 54, Y: 41})
	g.space.Unset(1, typ.P{X: 40, Y: 41})
	g.space.Unset(1, typ.P{X: 41, Y: 41})

	for y := 0; y < constants.WorldY; y++ {
		g.space.Set(1, typ.P{X: int32(0), Y: int32(y)}, uint16(assets.Shroom))
		g.space.Set(1, typ.P{X: int32(constants.WorldX - 1), Y: int32(y)}, uint16(assets.Shroom))
		for x := 0; x < constants.WorldX; x++ {
			g.space.Set(1, typ.P{X: int32(x), Y: int32(0)}, uint16(assets.Shroom))
			g.space.Set(1, typ.P{X: int32(x), Y: int32(constants.WorldY - 1)}, uint16(assets.Shroom))
			if x%25 == 0 && y%25 == 0 {
				g.space.Set(1, typ.P{X: int32(x), Y: int32(y)}, uint16(assets.Shroom))
			}
		}
	}
}

func (g *Game) Run() {
	g.AddObjectsToSpace()
	go g.HandleLogin()
	g.consumeIncomingData()

}

func (g *Game) consumeIncomingData() {
	log.Printf("Game started.\n")
	online := 0
	for incomingData := range g.incomingData {
		player := g.players[incomingData.ID]
		switch incomingData.Event {
		case msgs.EPlayerConnect:
			online++
			player = incomingData.Data.(*Player)
			g.AddPlayer(player)
			player.Login()
			log.Printf("LOG IN: %v  [%v] [%v]\n", player.m.IP(), player.nick, player.id)
		case msgs.EPing:
			player.Send <- OutMsg{Event: msgs.EPingOk, Data: uint16(online)}
		case msgs.EPlayerLogout:
			online--
			g.RemovePlayer(player.id)
			player.Logout()
			log.Printf("LOG OUT: %v  [%v] [%v]\n", player.m.IP(), player.nick, player.id)
		case msgs.EMove:
			g.playerMove(player, incomingData)
		case msgs.ECastSpell:
			g.playerCastSpell(player, incomingData)
		case msgs.EMelee:
			g.playerMelee(player, incomingData.Data.(direction.D))
		case msgs.EUseItem:
			item := incomingData.Data.(msgs.Item)
			//log.Printf("[%v][%v] USE ITEM %v\n", player.id, player.nick, item)
			changed := UseItem(item, player)
			player.Send <- OutMsg{Event: msgs.EUseItemOk, Data: &msgs.EventUseItemOk{
				Item:   msgs.Item(item),
				Change: changed,
			}}
		case msgs.ESendChat:
			chat := incomingData.Data.(*msgs.EventSendChat)
			log.Printf("[%v][%v]: %v", player.id, player.nick, chat.Msg)
			g.space.Notify(player.pos, msgs.EBroadcastChat, &msgs.EventBroadcastChat{
				ID:  player.id,
				Msg: chat.Msg,
			}, player.id)
		}
	}
}

const AverageGameFrame = time.Duration((time.Millisecond * 16) + (6 * (time.Millisecond / 10)))

func (g *Game) playerMove(player *Player, incomingData IncomingMsg) {
	player.dir = incomingData.Data.(direction.D)
	np := player.pos
	switch player.dir {
	case direction.Front:
		np.Y++
	case direction.Back:
		np.Y--
	case direction.Left:
		np.X--
	case direction.Right:
		np.X++
	}
	var err error
	if np.Out(g.space.Rect) {
		err = errors.New("map edge")
	} else if player.paralized {
		err = errors.New("player paralized")
	} else if block := g.space.GetSlot(1, np); block != 0 {
		err = errors.New("map object blocking")
	} else {
		err = g.space.Move(0, player.pos, np)
	}
	if err != nil {
		player.Send <- OutMsg{Event: msgs.EMoveOk, Data: []byte{msgs.BoolByte(false), player.dir}}
		g.space.Notify(player.pos, msgs.EPlayerMoved, &msgs.EventPlayerMoved{
			ID:  player.id,
			Pos: player.pos,
			Dir: player.dir,
		}, player.id)
		//log.Printf("[%v][%v] %v -X-> %v: %v\n", player.id, player.nick, player.pos, np, err)
		return
	}
	player.lastMove = time.Now()
	//log.Printf("[%v][%v] MOVE %v->%v\n", player.id, player.nick, player.pos, np)
	player.obs.MoveOne(player.dir, func(x, y int32) {
		newPlayerInSight := g.space.GetSlot(0, typ.P{X: x, Y: y})
		if newPlayerInSight == 0 {
			return
		}
		newPlayer := g.players[newPlayerInSight]
		newPlayer.Send <- OutMsg{Event: msgs.EPlayerEnterViewport, Data: &msgs.EventPlayerEnterViewport{
			ID:    player.id,
			Nick:  player.nick,
			Pos:   player.pos,
			Dir:   player.dir,
			Dead:  player.dead,
			Speed: uint8(player.speedPxXFrame),
		}}
		player.Send <- OutMsg{Event: msgs.EPlayerEnterViewport, Data: &msgs.EventPlayerEnterViewport{
			ID:    uint16(newPlayer.id),
			Nick:  newPlayer.nick,
			Pos:   newPlayer.pos,
			Dir:   newPlayer.dir,
			Dead:  newPlayer.dead,
			Speed: uint8(newPlayer.speedPxXFrame),
		}}
	}, func(x, y int32) {
		newPlayerOutSight := g.space.GetSlot(0, typ.P{X: x, Y: y})
		if newPlayerOutSight == 0 {
			return
		}
		newPlayerOut := g.players[newPlayerOutSight]
		newPlayerOut.Send <- OutMsg{Event: msgs.EPlayerLeaveViewport, Data: player.id}
		player.Send <- OutMsg{Event: msgs.EPlayerLeaveViewport, Data: uint16(newPlayerOut.id)}
	})
	g.space.Notify(np, msgs.EPlayerMoved, &msgs.EventPlayerMoved{
		ID:  player.id,
		Pos: np,
		Dir: player.dir,
	}, player.id)
	player.pos = np
	player.Send <- OutMsg{Event: msgs.EMoveOk, Data: []byte{msgs.BoolByte(true), player.dir}}
}

func (g *Game) playerCastSpell(player *Player, incomingData IncomingMsg) {
	ev := incomingData.Data.(*msgs.EventCastSpell)
	defer log.Printf("[%v][%v] SPELL %v at [%v %v]\n", player.id, player.nick, ev.Spell.String(), ev.PX, ev.PY)
	hitPlayer := g.CheckSpellTargets(typ.P{X: int32(ev.PX), Y: int32(ev.PY)})
	if hitPlayer == 0 {
		log.Printf("missed all hitboxs\n")
		return
	}
	targetPlayer := g.players[hitPlayer]
	dmg, err := Cast(GetSpellProp(ev.Spell), player, targetPlayer)
	if err != nil {
		return
	}
	if dmg < 0 {
		dmg = -dmg
	}
	g.space.Notify(targetPlayer.pos, msgs.EPlayerSpell, &msgs.EventPlayerSpell{
		ID:     uint16(hitPlayer),
		Spell:  ev.Spell,
		Killed: targetPlayer.dead,
	}, player.id, uint16(targetPlayer.id))
	player.Send <- OutMsg{Event: msgs.ECastSpellOk, Data: &msgs.EventCastSpellOk{
		ID:     uint16(hitPlayer),
		Damage: uint32(dmg),
		NewMP:  uint32(player.mp),
		Spell:  ev.Spell,
		Killed: targetPlayer.dead,
	}}
	targetPlayer.Send <- OutMsg{Event: msgs.EPlayerSpellRecieved, Data: &msgs.EventPlayerSpellRecieved{
		ID:     player.id,
		Spell:  ev.Spell,
		Damage: uint32(dmg),
		NewHP:  uint32(targetPlayer.hp),
	}}
}

func (g *Game) playerMelee(player *Player, d direction.D) {
	np := player.pos
	if d == 0 {
		d = player.dir
	}
	player.dir = d

	switch d {
	case direction.Front:
		np.Y++
	case direction.Back:
		np.Y--
	case direction.Left:
		np.X--
	case direction.Right:
		np.X++
	}
	log.Printf("[%v][%v] MELEE looking %v at %v to %v\n", player.id, player.nick, direction.S(d), player.pos, np)
	if player.dead {
		log.Printf("dead?")

		player.Send <- OutMsg{Event: msgs.EMeleeOk, Data: &msgs.EventMeleeOk{}}
		return
	}
	targetId := g.space.GetSlot(0, np)
	dmg := int32(0)
	killed := false
	if targetId != 0 {
		targetPlayer := g.players[targetId]
		if !targetPlayer.dead {
			dmg = Melee(player, targetPlayer)
			targetPlayer.Send <- OutMsg{Event: msgs.EPlayerMeleeRecieved, Data: &msgs.EventPlayerMeleeRecieved{
				ID:     player.id,
				Damage: uint32(dmg),
				NewHP:  uint32(targetPlayer.hp),
				Dir:    player.dir,
			}}
		} else {
			targetId = 0
		}
		killed = targetPlayer.dead
	}
	plMele := &msgs.EventPlayerMelee{
		From:   player.id,
		ID:     targetId,
		Hit:    targetId != 0,
		Killed: killed,
		Dir:    player.dir,
	}
	log.Printf("%#v", *plMele)
	g.space.Notify(player.pos, msgs.EPlayerMelee, plMele, player.id, targetId)
	meleOk := &msgs.EventMeleeOk{
		ID:     targetId,
		Damage: uint32(dmg),
		Hit:    targetId != 0,
		Killed: killed,
		Dir:    player.dir,
	}
	log.Printf("%#v", *meleOk)

	player.Send <- OutMsg{Event: msgs.EMeleeOk, Data: meleOk}
}

type Player struct {
	g    *Game
	obs  *grid.Obs
	m    msgs.Msgs
	Send chan OutMsg
	id   uint16
	nick string
	pos  typ.P
	dir  direction.D

	lastMove      time.Time
	speedXTile    time.Duration
	speedPxXFrame int32

	paralized bool
	dead      bool
	hp        int32
	maxHp     int32
	mp        int32
	maxMp     int32

	spellCD Cooldown
	meleeCD Cooldown
	itemCD  Cooldown
}

type OutMsg struct {
	Event msgs.E
	Data  interface{}
}

func (p *Player) HandleOutgoingMessages() {
	for {
		select {
		case m := <-p.Send:
			p.m.EncodeAndWrite(m.Event, m.Data)
		case ev, ok := <-p.obs.Events:
			if !ok {
				return
			}
			if ev.HasID(uint16(p.id)) {
				continue
			}
			p.m.EncodeAndWrite(ev.E, ev.Data)
		}
	}
}

func (p *Player) HandleIncomingMessages() {
	for {

		im, err := p.m.Read()
		if err != nil {
			p.g.incomingData <- IncomingMsg{
				ID:    uint16(p.id),
				Event: msgs.EPlayerLogout,
			}
			return
		}
		msg := IncomingMsg{
			ID:    uint16(p.id),
			Event: im.Event,
		}
		//log.Printf("recieved %v from %v", im.Event.String(), p.nick)
		switch im.Event {
		case msgs.EPing:
		case msgs.EMove:
			msg.Data = direction.D(im.Data[0])
		case msgs.ECastSpell:
			msg.Data = msgs.DecodeEventCastSpell(im.Data)
		case msgs.EMelee:
			msg.Data = direction.D(im.Data[0])
		case msgs.EUseItem:
			msg.Data = msgs.Item(im.Data[0])
		case msgs.ESendChat:
			d := msgs.DecodeMsgpack(im.Data, &msgs.EventSendChat{})
			msg.Data = d
		default:
			log.Printf("HandleIncomingMessages unknown event\n")
			continue
		}
		p.g.incomingData <- msg
	}
}

func checkSpawn(g *grid.Grid, spawn typ.P) typ.P {
	empty := func(p typ.P) bool {
		return g.GetSlot(0, p)+g.GetSlot(1, p) == 0
	}
	sign := int32(1)
	for {
		if empty(spawn) {
			return spawn
		}
		spawn.X += sign
		if spawn.X >= g.Rect.Max.X || spawn.X < 0 {
			spawn.X += -sign
			spawn.Y++
			sign = -sign
		}
	}
}

func (p *Player) Login() {
	p.pos = checkSpawn(p.g.space, p.pos)
	loginEvent := &msgs.EventPlayerLogin{
		ID:    uint16(p.id),
		Nick:  p.nick,
		Pos:   p.pos,
		Dir:   p.dir,
		HP:    p.hp,
		MaxHP: p.maxHp,
		MP:    p.mp,
		MaxMP: p.maxMp,
		Speed: uint8(p.speedPxXFrame),
	}
	log.Printf("login %#v", *loginEvent)

	p.obs = grid.NewObserverRange(p.g.space, p.pos,
		constants.GridViewportX, constants.GridViewportY,
		func(t *grid.Tile) {
			if t.Layers[0] != 0 {
				log.Print(t.Layers[0])
				vp := p.g.players[t.Layers[0]]
				loginEvent.VisiblePlayers = append(loginEvent.VisiblePlayers, msgs.EventNewPlayer{
					ID:    uint16(vp.id),
					Nick:  vp.nick,
					Pos:   vp.pos,
					Dir:   vp.dir,
					Speed: uint8(vp.speedPxXFrame),
					Dead:  vp.dead,
				})
			}
		},
	)
	p.g.space.Set(0, p.pos, uint16(p.id))
	go p.HandleIncomingMessages()
	go p.HandleOutgoingMessages()
	encoded := msgs.EncodeMsgpack(loginEvent)
	testLogin := msgs.DecodeMsgpack(encoded, &msgs.EventPlayerLogin{})
	log.Printf("login %#v", *testLogin)

	p.m.WriteWithLen(msgs.EPlayerLogin, encoded)
	p.g.space.Notify(p.pos, msgs.EPlayerSpawned, &msgs.EventPlayerSpawned{
		ID:    uint16(p.id),
		Nick:  p.nick,
		Pos:   p.pos,
		Dir:   p.dir,
		Speed: uint8(p.speedPxXFrame),
	}, uint16(p.id))
}

func (p *Player) Logout() {
	p.obs.Nuke()
	p.g.space.Unset(0, p.pos)
	p.g.space.Notify(p.pos, msgs.EPlayerDespawned, uint16(p.id), uint16(p.id))
}

func (p *Player) TakeDamage(dmg int32) {
	p.hp = p.hp - dmg
	if p.hp <= 0 {
		p.hp = 0
		p.dead = true
		p.paralized = false
	}
}

func (p *Player) Heal(heal int32) {
	p.hp = p.hp + heal
	if p.hp > p.maxHp {
		p.hp = p.maxHp
	}
}
