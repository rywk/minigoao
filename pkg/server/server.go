package server

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/constants/mapdef"
	"github.com/rywk/minigoao/pkg/constants/skill"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/server/db"
	"github.com/rywk/minigoao/pkg/server/webpage"
	"github.com/rywk/minigoao/pkg/typ"
	"golang.org/x/crypto/bcrypt"
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
	db        db.DB
}

func NewServer(tcpport string, webport string) *Server {

	return &Server{
		tcpport: tcpport,
		webport: webport,
		newConn: make(chan msgs.Msgs, 100),
		db:      db.NewDB(),
	}
}

func (s *Server) AcceptTCPConnections() {
	log.Printf("Accepting TCP connections at %v.\n", s.mms.Address())
	for {
		conn, err := s.mms.NewConn()
		if err != nil {
			return
		}
		log.Printf("accepted plain tcp conn %v\n", conn.IP())
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
		log.Printf("accepted web socket conn %v\n", conn.IP())
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
		chars := []msgs.Character{}
		for _, pid := range s.game.playersIndex {
			p := s.game.players[pid]
			chars = append(chars, msgs.Character{
				ID:        p.characterID,
				Px:        int(p.pos.X),
				Py:        int(p.pos.Y),
				Dir:       p.dir,
				Kills:     p.kills,
				Deaths:    p.deaths,
				Skills:    p.exp.Skills,
				KeyConfig: p.keyConfigs,
				Inventory: *p.inv,
				LoggedIn:  false,
			})

		}
		err := s.db.SaveAndLogOutAll(chars)
		if err != nil {
			log.Printf("SaveAndLogOutAll err %v", err)
		}
		err = s.db.LogOutAll()
		if err != nil {
			log.Printf("LogOutAll err %v", err)
		}
		log.Printf("Shutting down web...")

		// Received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err = web.Shutdown(ctx)
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
				log.Fatalf("Error at server.ListenAndServeTLS: %v", err)
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
		space:        grid.NewGrid(constants.WorldX, constants.WorldY, uint8(mapdef.LayerTypes)),
		incomingData: make(chan IncomingMsg, 1000),
		db:           s.db,
		killUpdater:  make(chan KillToUpdate, 20),
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
	db           db.DB

	killUpdater chan KillToUpdate
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

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func MatchPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (g *Game) HandleLogin(m msgs.Msgs, account *db.Account, characters []msgs.Character) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		m.Close()
		log.Printf("HandleLogin panic: %v", r)
	}()
	last := time.Now().Add(-time.Second)
	for {
		var err error
		var msg *msgs.IncomingData
		done := make(chan struct{})
		go func() {
			msg, err = m.Read()
			done <- struct{}{}
		}()
		select {
		case <-time.Tick(time.Second * 300):
			acc := ""
			if account != nil {
				acc = account.Account + " " + account.Email
			}
			log.Printf("Session Timeout %v   %v", m.IP(), acc)
			m.Close()
			return
		case <-done:
		}

		if err == msgs.ErrBadData {
			m.Close()
			return
		}

		if err != nil {
			log.Printf("HandleLogin:%v", err)
			return
		}

		now := time.Now()
		if now.Sub(last) < time.Millisecond*1000 {
			acc := ""
			if account != nil {
				acc = account.Account + " " + account.Email
			}
			m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "Chill..."})

			log.Printf("Loggin in too fast %v   %v", m.IP(), acc)
			continue
		}
		last = now

		handle := func(fn func() (msgs.E, interface{})) {
			var (
				e msgs.E
				d interface{}
			)
			defer func() {
				m.EncodeAndWrite(e, d)
			}()
			e, d = fn()

			if e == msgs.EError {
				log.Printf("%v err: %v\n", msg.Event, d.(*msgs.EventError).Msg)
			}

		}

		switch msg.Event {
		case msgs.ECreateAccount:
			handle(func() (msgs.E, interface{}) {
				ca := msgs.DecodeMsgpack(msg.Data, &msgs.EventCreateAccount{})
				log.Printf("CREATE ACCOUNT %v %v\n", ca.Account, ca.Email)
				err := g.db.CreateAccount(ca.Account, ca.Email, HashPassword(ca.Password))
				if err != nil {
					return msgs.EError, &msgs.EventError{Msg: "CreateAccount " + err.Error()}

				}
				account, err = g.db.GetAccount(ca.Account, "")
				if err != nil {
					return msgs.EError, &msgs.EventError{Msg: "GetAccount " + err.Error()}

				}
				characters, err = g.db.GetAccountCharacters(account.ID)
				if err != nil {
					return msgs.EError, &msgs.EventError{Msg: "GetAccountCharacters " + err.Error()}

				}
				resp := msgs.EventAccountLogin{
					ID:         uint16(account.ID),
					Account:    account.Account,
					Email:      account.Email,
					Characters: characters,
				}
				log.Printf("CREATE ACCOUNT %v\n", resp)
				return msgs.EAccountLoginOk, &resp
			})
			// ca := msgs.DecodeMsgpack(msg.Data, &msgs.EventCreateAccount{})
			// log.Printf("CREATE ACCOUNT %v\n", *ca)
			// err := g.db.CreateAccount(ca.Account, ca.Email, HashPassword(ca.Password))
			// if err != nil {
			// 	log.Printf("CreateAccount err %v\n", err)
			// 	m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "CreateAccount " + err.Error()})
			// 	continue
			// }
			// account, err = g.db.GetAccount(ca.Account, "")
			// if err != nil {
			// 	log.Printf("GetAccount err %v\n", err)
			// 	m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "GetAccount " + err.Error()})
			// 	continue
			// }
			// characters, err = g.db.GetAccountCharacters(account.ID)
			// if err != nil {
			// 	log.Printf("GetAccountCharacters err %v\n", err)
			// 	m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "GetAccountCharacters " + err.Error()})
			// 	continue
			// }
			// resp := msgs.EventAccountLogin{
			// 	ID:         uint16(account.ID),
			// 	Account:    account.Account,
			// 	Email:      account.Email,
			// 	Characters: characters,
			// }
			// log.Printf("CREATE ACCOUNT %v\n", resp)
			// m.EncodeAndWrite(msgs.EAccountLoginOk, &resp)
		case msgs.ELoginAccount:
			ca := msgs.DecodeMsgpack(msg.Data, &msgs.EventLoginAccount{})
			log.Printf("LOGIN ACCOUNT %v\n", ca.Account)
			account, err = g.db.GetAccount(ca.Account, "")
			if err != nil {
				log.Printf("LOGIN ACCOUNT ERR %v\n", err)
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "GetAccount " + err.Error()})
				continue
			}
			if !MatchPassword(ca.Password, account.Password) {
				log.Printf("LOGIN ACCOUNT ERR %v\n", "Password doesnt match")
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "Password doesnt match"})
				continue
			}
			characters, err = g.db.GetAccountCharacters(account.ID)
			if err != nil {
				log.Printf("LOGIN ACCOUNT ERR %v\n", err)
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "GetAccountCharacters " + err.Error()})
				continue
			}
			resp := msgs.EventAccountLogin{
				ID:         uint16(account.ID),
				Account:    account.Account,
				Email:      account.Email,
				Characters: characters,
			}
			m.EncodeAndWrite(msgs.EAccountLoginOk, &resp)
			log.Printf("LOGIN ACCOUNT %v\n", resp)
		case msgs.ECreateCharacter:
			ca := msgs.DecodeMsgpack(msg.Data, &msgs.EventCreateCharacter{})
			log.Printf("CREATE CHARACTER %v\n", *ca)
			err := g.db.CreateCharacter(account.ID, ca.Nick)
			if err != nil {
				log.Printf("CREATE CHARACTER err %v\n", err)
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "CreateCharacter " + err.Error()})
				continue
			}

			characters, err = g.db.GetAccountCharacters(account.ID)
			if err != nil {
				log.Printf("CREATE CHARACTER err %v\n", err)
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "GetAccountCharacters " + err.Error()})
				continue
			}

			resp := msgs.EventAccountLogin{
				ID:         uint16(account.ID),
				Account:    account.Account,
				Email:      account.Email,
				Characters: characters,
			}
			m.EncodeAndWrite(msgs.EAccountLoginOk, &resp)
			log.Printf("CREATE CHARACTER %v\n", resp)
		case msgs.ELoginCharacter:
			ca := msgs.DecodeMsgpack(msg.Data, &msgs.EventLoginCharacter{})
			log.Printf("LOGIN CHARACTER %v\n", *ca)
			var character *msgs.Character
			for _, char := range characters {
				if ca.ID == uint16(char.ID) {
					character = &char
					break
				}
			}
			if character == nil {
				log.Printf("LOGIN CHARACTER err %v\n", "Character doesnt exist")
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "Character doesnt exist"})
				continue
			}

			char, err := g.db.GetCharacter(int(ca.ID))
			if err != nil {
				log.Printf("LOGIN CHARACTER err %v\n", err)
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "GetCharacter " + err.Error()})
				continue
			}
			err = g.db.LogInCharacter(char.ID)
			if err != nil {
				log.Printf("LOGIN CHARACTER err %v\n", err)
				m.EncodeAndWrite(msgs.EError, &msgs.EventError{Msg: "LogInCharacter " + err.Error()})
				continue
			}
			*character = *char
			p := &Player{
				g:             g,
				m:             m,
				pos:           typ.P{X: int32(character.Px), Y: int32(character.Py)},
				Send:          make(chan OutMsg, 100),
				dir:           character.Dir,
				speedPxXFrame: 3,
				speedXTile:    (constants.TileSize / 3) * AverageGameFrame,
				inv:           &character.Inventory,
				cds:           &Cooldowns{},
				keyConfigs:    character.KeyConfig,
				nick:          character.Nick,
				kills:         character.Kills,
				deaths:        character.Deaths,
				account:       account,
				characters:    characters,
				characterID:   character.ID,
				space:         g.space,
				mapType:       mapdef.MapLobby,
			}
			log.Printf("LOGIN CHARACTER %v\n", *character)
			p.exp = NewExperience(p)
			p.exp.SetNewSkills(character.Skills)
			p.hp = p.exp.Stats.MaxHP
			p.mp = p.exp.Stats.MaxMP
			g.incomingData <- IncomingMsg{
				Event: msgs.EPlayerConnect,
				Data:  p,
			}
			return
		}
	}
}

func (g *Game) HandleLogins() {
	log.Printf("Login handler started.\n")
	for conn := range g.newConn {
		go g.HandleLogin(conn, nil, []msgs.Character{})
	}
}

func (g *Game) Run() {
	g.AddObjectsToSpace(g.space, mapdef.MapLobby)
	go g.HandleLogins()
	g.consumeIncomingData()

}

const (
	// internal event
	// very careful here, we start from the top
	UpdateServerRankList msgs.E = 255
)

func (g *Game) UpdateRankList(exit <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-exit:
			return
		case <-ticker.C:
			list, err := g.db.GetTopNKills(16)
			if err != nil {
				log.Printf("GetTop20Kills err: %v", err)
				return
			}
			list2, err := g.db.GetTopNPvP1v1(16)
			if err != nil {
				log.Printf("GetTopNPvP1v1 err: %v", err)
				return
			}
			list3, err := g.db.GetTopNPvP2v2(16)
			if err != nil {
				log.Printf("GetTopNPvP1v1 err: %v", err)
				return
			}
			byKills := []msgs.RankChar{}
			for _, ch := range list {
				byKills = append(byKills, ch.ToRankChar())
			}
			byArena1v1 := []msgs.RankChar{}
			for _, ch := range list2 {
				byArena1v1 = append(byArena1v1, ch.ToRankChar())
			}
			byArena2v2 := []msgs.RankChar{}
			for _, ch := range list3 {
				byArena2v2 = append(byArena2v2, ch.ToRankChar())
			}
			g.incomingData <- IncomingMsg{
				Event: UpdateServerRankList,
				Data:  [][]msgs.RankChar{byKills, byArena1v1, byArena2v2},
			}
		}
	}
}

type KillToUpdate struct {
	Kill   int
	Killed int
}

func (g *Game) KillUpdater(ku <-chan KillToUpdate) {
	for update := range ku {
		err := g.db.AddCharacterDeath(update.Killed)
		if err != nil {
			log.Printf("AddCharacterDeath err: %v", err)
			return
		}
		err = g.db.AddCharacterKill(update.Kill)
		if err != nil {
			log.Printf("AddCharacterKill err: %v", err)
			return
		}

	}
}

func (g *Game) consumeIncomingData() {
	exit := make(chan struct{})
	defer func() {
		exit <- struct{}{}
	}()
	go g.UpdateRankList(exit)

	go g.KillUpdater(g.killUpdater)
	log.Printf("Game started.\n")
	online := 0
	var rankList msgs.EventRankList
	for incomingData := range g.incomingData {
		player := g.players[incomingData.ID]
		switch incomingData.Event {
		case UpdateServerRankList:
			rankList.Kills = incomingData.Data.([][]msgs.RankChar)[0]
			rankList.Arena1v1 = incomingData.Data.([][]msgs.RankChar)[1]
			rankList.Arena2v2 = incomingData.Data.([][]msgs.RankChar)[2]
		case msgs.EGetRankList:
			player.Send <- OutMsg{Event: msgs.ERankList, Data: &rankList}
		case msgs.EPing:
			player.Send <- OutMsg{Event: msgs.EPingOk, Data: uint16(online)}
		case msgs.EMove:
			g.playerMove(player, incomingData)
		case msgs.ECastSpell:
			g.playerCastSpell(player, incomingData)
		case msgs.EMelee:
			g.playerMelee(player, incomingData.Data.(direction.D))
		case msgs.ESelectSpell:
			player.SelectedSpell = incomingData.Data.(attack.Spell)
		case msgs.EUpdateKeyConfig:
			player.keyConfigs = *incomingData.Data.(*msgs.KeyConfig)
		case msgs.EPlayerConnect:
			online++
			player = incomingData.Data.(*Player)
			g.AddPlayer(player)
			player.Login()
			log.Printf("LOG IN: %v  [%v] [%v]\n", player.m.IP(), player.nick, player.id)
		case msgs.EPlayerLogout:
			online--
			log.Printf("LOG OUT: %v  [%v] [%v]\n", player.m.IP(), player.nick, player.id)
			g.RemovePlayer(player.id)
			player.Logout()
		case msgs.EUpdateSkills:
			skills := incomingData.Data.(*skill.Skills)
			player.exp.SetNewSkills(*skills)
			nexp := player.exp.ToMsgs()
			player.Send <- OutMsg{Event: msgs.EUpdateSkillsOk, Data: &nexp}
		case msgs.ESendChat:
			chat := incomingData.Data.(*msgs.EventSendChat)
			log.Printf("[%v][%v]: %v", player.id, player.nick, chat.Msg)

			if len(chat.Msg) > 0 && chat.Msg[0] == '/' {
				player.HandleCmd(chat.Msg[1:])
				continue
			}

			player.space.Notify(player.pos, msgs.EBroadcastChat.U8(), &msgs.EventBroadcastChat{
				ID:  player.id,
				Msg: chat.Msg,
			}, player.id)

		case msgs.EUseItem:
			if player.dead {
				continue
			}
			it := incomingData.Data.(*msgs.EventUseItem)
			is := player.inv.GetSlotf(it)
			if is.Item == item.None {
				continue
			}
			// if is.Count == 0 {
			// 	continue
			// }

			//log.Printf("[%v][%v] USE ITEM %v slot %v\n", player.id, player.nick, is.Item.String(), it)

			// consumable behaviour
			if is.Item.Type() == item.TypeConsumable {
				now := time.Now()
				if now.Sub(player.lastConsumable) < constants.PotionCooldown-time.Millisecond*5 {
					log.Printf("%v is drinking potions too fast", player.nick)
					continue
				}
				player.lastConsumable = now

				is.Count--
				changed := item.UseItem(is.Item, player)
				player.Send <- OutMsg{Event: msgs.EUseItemOk, Data: &msgs.EventUseItemOk{
					Slot:   msgs.InventoryPos{X: it.X, Y: it.Y},
					Item:   is.Item,
					Change: changed,
					Count:  is.Count,
				}}
				if is.Count == 0 {
					is.Item = item.None
				}
				continue
			}
			used := msgs.InventoryPos(*it)
			//var currentItem item.Item
			var target *msgs.InventoryPos
			// equippable behaviour
			switch is.Item.Type() {
			case item.TypeArmor:
				target = &player.inv.EquippedBody

			case item.TypeHelmet:
				target = &player.inv.EquippedHead

			case item.TypeShield:
				target = &player.inv.EquippedShield

			case item.TypeWeapon:
				target = &player.inv.EquippedWeapon

			default:
				log.Print("unknown item type recived")
				continue
			}

			change := uint32(1)
			if *target == used {
				change = 0
				*target = msgs.InventoryPos{X: 255, Y: 0}
			} else {
				*target = used
			}
			player.exp.SetItemBuffs()
			nexp := player.exp.ToMsgs()
			player.Send <- OutMsg{Event: msgs.EUpdateSkillsOk, Data: &nexp}
			player.Send <- OutMsg{Event: msgs.EUseItemOk, Data: &msgs.EventUseItemOk{
				Slot:   msgs.InventoryPos{X: it.X, Y: it.Y},
				Item:   is.Item,
				Change: change,
				Count:  is.Count,
			}}
			player.space.Notify(player.pos, msgs.EPlayerChangedSkin.U8(), &msgs.EventPlayerChangedSkin{
				ID:     player.id,
				Armor:  player.inv.GetBody(),
				Weapon: player.inv.GetWeapon(),
				Shield: player.inv.GetShield(),
				Head:   player.inv.GetHead(),
			}, player.id)
		}
	}
}

const AverageGameFrame = time.Duration((time.Millisecond * 16) + (6 * (time.Millisecond / 10)))

func (g *Game) playerMove(player *Player, incomingData IncomingMsg) {
	prevDir := player.dir
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

	notOk := func() {
		player.Send <- OutMsg{Event: msgs.EMoveOk, Data: []byte{msgs.BoolByte(false), player.dir}}
		if prevDir == player.dir {
			return
		}
		player.space.Notify(player.pos, msgs.EPlayerMoved.U8(), &msgs.EventPlayerMoved{
			ID:  player.id,
			Pos: player.pos,
			Dir: player.dir,
		}, player.id)
	}

	var err error
	if np.Out(player.space.Rect) {
		notOk()
		return
	} else if player.paralized {
		notOk()
		return
	}
	stuffId := player.space.GetSlot(mapdef.Stuff.Int(), np)
	im := assets.Image(stuffId)
	if assets.IsSolid(im) {
		notOk()
		return
	}

	err = player.space.Move(mapdef.Players.Int(), player.pos, np)
	//log.Printf("[%v][%v] MOVE %v->%v err:%v\n", player.id, player.nick, player.pos, np, err)
	if err != nil {
		notOk()
		return
	}
	player.lastMove = time.Now()
	player.pos = np
	if player.meditating {
		player.meditating = false
		player.space.Notify(player.pos, msgs.EPlayerMeditating.U8(), &msgs.EventPlayerMeditating{
			ID:         player.id,
			Meditating: player.meditating,
		})
	}

	groundId := player.space.GetSlot(mapdef.Ground.Int(), np)
	switch assets.Image(groundId) {
	case assets.MossBricks:
		if player.dead {
			player.Revive()
			player.Heal(player.exp.Stats.MaxHP)
			player.Send <- OutMsg{Event: msgs.EPlayerSpellRecieved, Data: &msgs.EventPlayerSpellRecieved{
				ID:     player.id,
				Damage: uint32(player.exp.Stats.MaxHP),
				Spell:  attack.SpellResurrect,
				NewHP:  uint32(player.hp),
			}}
			player.space.Notify(player.pos, msgs.EPlayerSpell.U8(), &msgs.EventPlayerSpell{
				ID:    player.id,
				Spell: attack.SpellResurrect,
			}, player.id)
		}
	case assets.PvPTeam1Tile, assets.PvPTeam2Tile:
		if mapdef.In1v1Spawn(player.pos.Point()) {
			if oponentId := mapdef.OponentPvP1(player.space, player.pos.Point()); oponentId != 0 {
				oponent := g.players[oponentId]
				notOk()
				g.StartPvP1v1(player, oponent)
				return
			}
		} else if mapdef.In2v2Spawn(player.pos.Point()) {
			allayId := mapdef.AllayPvP2(player.space, player.pos.Point())
			enemy1, enemy2 := mapdef.OponentPvP2(player.space, player.pos.Point())
			if allayId == 0 || enemy1 == 0 || enemy2 == 0 {
				break
			}
			notOk()
			g.StartPvP2v2(player, g.players[allayId], g.players[enemy1], g.players[enemy2])
			return
		}

	}
	player.obs.MoveOne(player.dir, func(x, y int32) {
		newPlayerInSight := player.space.GetSlot(mapdef.Players.Int(), typ.P{X: x, Y: y})
		if newPlayerInSight == 0 {
			return
		}
		newPlayer := g.players[newPlayerInSight]
		newPlayer.Send <- OutMsg{Event: msgs.EPlayerEnterViewport, Data: &msgs.EventPlayerEnterViewport{
			ID:         player.id,
			Nick:       player.nick,
			Pos:        player.pos,
			Dir:        player.dir,
			Dead:       player.dead,
			Weapon:     player.inv.GetWeapon(),
			Shield:     player.inv.GetShield(),
			Head:       player.inv.GetHead(),
			Body:       player.inv.GetBody(),
			Speed:      uint8(player.speedPxXFrame),
			Meditating: player.meditating,
		}}
		player.Send <- OutMsg{Event: msgs.EPlayerEnterViewport, Data: &msgs.EventPlayerEnterViewport{
			ID:         uint16(newPlayer.id),
			Nick:       newPlayer.nick,
			Pos:        newPlayer.pos,
			Dir:        newPlayer.dir,
			Dead:       newPlayer.dead,
			Weapon:     newPlayer.inv.GetWeapon(),
			Shield:     newPlayer.inv.GetShield(),
			Head:       newPlayer.inv.GetHead(),
			Body:       newPlayer.inv.GetBody(),
			Speed:      uint8(newPlayer.speedPxXFrame),
			Meditating: newPlayer.meditating,
		}}
	}, func(x, y int32) {
		newPlayerOutSight := player.space.GetSlot(mapdef.Players.Int(), typ.P{X: x, Y: y})
		if newPlayerOutSight == 0 {
			return
		}
		newPlayerOut := g.players[newPlayerOutSight]
		newPlayerOut.Send <- OutMsg{Event: msgs.EPlayerLeaveViewport, Data: player.id}
		player.Send <- OutMsg{Event: msgs.EPlayerLeaveViewport, Data: uint16(newPlayerOut.id)}
	})
	player.space.Notify(np, msgs.EPlayerMoved.U8(), &msgs.EventPlayerMoved{
		ID:  player.id,
		Pos: np,
		Dir: player.dir,
	}, player.id)
	player.Send <- OutMsg{Event: msgs.EMoveOk, Data: []byte{msgs.BoolByte(true), player.dir}}
}

func (g *Game) playerCastSpell(player *Player, incomingData IncomingMsg) {
	groundId := player.space.GetSlot(mapdef.Ground.Int(), player.pos)
	if !assets.CanFight(assets.Image(groundId)) {
		player.cds.LastAction = time.Now()
		return
	}
	ev := incomingData.Data.(*msgs.EventCastSpell)
	hitPlayer := g.CheckSpellTargets(player.space, typ.P{X: int32(ev.PX), Y: int32(ev.PY)})
	if hitPlayer == 0 {
		log.Printf("[%v][%v] SPELL %v [%v %v] missed\n", player.id, player.nick, player.SelectedSpell.String(), ev.PX, ev.PY)
		return
	}
	defer log.Printf("[%v][%v] SPELL %v at [%v %v]\n", player.id, player.nick, player.SelectedSpell.String(), ev.PX, ev.PY)

	targetPlayer := g.players[hitPlayer]
	groundId = player.space.GetSlot(mapdef.Ground.Int(), targetPlayer.pos)
	if !assets.CanFight(assets.Image(groundId)) {
		player.cds.LastAction = time.Now()
		return
	}
	dmg, err := Cast(player, targetPlayer)
	if err != nil {
		return
	}
	if dmg < 0 {
		dmg = -dmg
	}
	player.space.Notify(targetPlayer.pos, msgs.EPlayerSpell.U8(), &msgs.EventPlayerSpell{
		ID:     uint16(hitPlayer),
		Spell:  player.SelectedSpell,
		Killed: targetPlayer.dead,
		Caster: player.id,
	}, player.id, uint16(targetPlayer.id))

	player.Send <- OutMsg{Event: msgs.ECastSpellOk, Data: &msgs.EventCastSpellOk{
		ID:     uint16(hitPlayer),
		Damage: uint32(dmg),
		NewMP:  uint32(player.mp),
		Spell:  player.SelectedSpell,
		Killed: targetPlayer.dead,
	}}
	targetPlayer.Send <- OutMsg{Event: msgs.EPlayerSpellRecieved, Data: &msgs.EventPlayerSpellRecieved{
		ID:     player.id,
		Spell:  player.SelectedSpell,
		Damage: uint32(dmg),
		NewHP:  uint32(targetPlayer.hp),
	}}
	player.SelectedSpell = attack.SpellNone

	if targetPlayer.dead {
		g.killUpdater <- KillToUpdate{Kill: player.characterID, Killed: targetPlayer.characterID}
		if player.mapType == mapdef.MapPvP1v1 {
			g.EndPvP1v1(player, targetPlayer)
		} else if player.mapType == mapdef.MapPvP2v2 {
			g.EndPvP2v2(player, targetPlayer)
		}
	}
}

func (g *Game) playerMelee(player *Player, d direction.D) {
	groundId := player.space.GetSlot(mapdef.Ground.Int(), player.pos)
	if !assets.CanFight(assets.Image(groundId)) {
		player.cds.LastMelee = time.Now()
		return
	}
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
	defer log.Printf("[%v][%v] MELEE looking %v at %v to %v\n", player.id, player.nick, direction.S(d), player.pos, np)
	if player.dead {

		player.Send <- OutMsg{Event: msgs.EMeleeOk, Data: &msgs.EventMeleeOk{}}
		return
	}
	targetId := player.space.GetSlot(mapdef.Players.Int(), np)
	dmg := int32(0)
	killed := false
	var targetPlayer *Player
	if targetId != 0 {
		targetPlayer = g.players[targetId]
		groundId = player.space.GetSlot(mapdef.Ground.Int(), targetPlayer.pos)
		if !assets.CanFight(assets.Image(groundId)) {
			player.cds.LastMelee = time.Now()
			return
		}
		if !targetPlayer.dead {

			dmg = Melee(player, targetPlayer)
			if dmg == -1 {
				log.Printf("melee error, too fast")
				return
			}

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
	player.space.Notify(player.pos, msgs.EPlayerMelee.U8(), plMele, player.id, targetId)
	meleOk := &msgs.EventMeleeOk{
		ID:     targetId,
		Damage: uint32(dmg),
		Hit:    targetId != 0,
		Killed: killed,
		Dir:    player.dir,
	}

	player.Send <- OutMsg{Event: msgs.EMeleeOk, Data: meleOk}
	if targetPlayer != nil && targetPlayer.dead {
		g.killUpdater <- KillToUpdate{Kill: player.characterID, Killed: targetPlayer.characterID}
		if player.mapType == mapdef.MapPvP1v1 {
			g.EndPvP1v1(player, targetPlayer)
		}
	}
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

	space   *grid.Grid
	mapType mapdef.MapType

	lastMove      time.Time
	speedXTile    time.Duration
	speedPxXFrame int32

	paralized bool
	dead      bool
	hp        int32
	mp        int32

	kills  int
	deaths int

	team   []uint16
	enemys []uint16

	meditating bool

	account       *db.Account
	characters    []msgs.Character
	characterID   int
	exp           *Experience
	inv           *msgs.Inventory
	cds           *Cooldowns
	SelectedSpell attack.Spell

	lastConsumable time.Time

	keyConfigs msgs.KeyConfig
}

const (
	PlayerCMDMeditar = "meditar"
)

func (p *Player) HandleCmd(cmd string) {
	cmd = strings.ToLower(cmd)
	log.Printf("handling cmd: %v", cmd)
	switch cmd {
	case PlayerCMDMeditar:
		p.meditating = !p.meditating
		p.space.Notify(p.pos, msgs.EPlayerMeditating.U8(), &msgs.EventPlayerMeditating{
			ID:         p.id,
			Meditating: p.meditating,
		})
	}
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
			p.m.EncodeAndWrite(msgs.E(ev.E), ev.Data)
		}
	}
}

func (p *Player) HandleIncomingMessages() {
	for {

		im, err := p.m.Read()
		if err == msgs.ErrBadData {
			continue
		}
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
			msg.Data = msgs.DecodeEventUseItem(im.Data)
		case msgs.ESendChat:
			d := msgs.DecodeMsgpack(im.Data, &msgs.EventSendChat{})
			msg.Data = d
		case msgs.ESelectSpell:
			msg.Data = attack.Spell(im.Data[0])
		case msgs.EUpdateSkills:
			d := msgs.DecodeMsgpack(im.Data, &skill.Skills{})
			msg.Data = d
		case msgs.EUpdateKeyConfig:
			d := msgs.DecodeMsgpack(im.Data, &msgs.KeyConfig{})
			msg.Data = d
		case msgs.EPlayerLogout:
			p.g.incomingData <- msg
			return
		case msgs.EGetRankList:
		default:
			log.Printf("HandleIncomingMessages unknown event\n")
			continue
		}
		p.g.incomingData <- msg
	}
}

func checkSpawn(g *grid.Grid, spawn typ.P) typ.P {
	empty := func(p typ.P) bool {
		return g.GetSlot(mapdef.Players.Int(), p)+g.GetSlot(mapdef.Stuff.Int(), p) == 0
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
	p.pos = checkSpawn(p.space, p.pos)
	loginEvent := &msgs.EventPlayerLogin{
		ID:        uint16(p.id),
		Nick:      p.nick,
		Pos:       p.pos,
		Dir:       p.dir,
		HP:        p.hp,
		MP:        p.mp,
		Exp:       p.exp.ToMsgs(),
		Inv:       msgs.Inventory(*p.inv),
		Speed:     uint8(p.speedPxXFrame),
		KeyConfig: p.keyConfigs,
		MapType:   p.mapType,
	}
	//log.Printf("login %#v", *loginEvent)

	p.obs = grid.NewObserverRange(p.space, p.pos,
		constants.GridViewportX, constants.GridViewportY,
		func(t *grid.Tile) {
			if t.Layers[mapdef.Players] != 0 {
				vp := p.g.players[t.Layers[mapdef.Players]]
				loginEvent.VisiblePlayers = append(loginEvent.VisiblePlayers, msgs.EventNewPlayer{
					ID:         uint16(vp.id),
					Nick:       vp.nick,
					Pos:        vp.pos,
					Dir:        vp.dir,
					Speed:      uint8(vp.speedPxXFrame),
					Weapon:     vp.inv.GetWeapon(),
					Shield:     vp.inv.GetShield(),
					Head:       vp.inv.GetHead(),
					Body:       vp.inv.GetBody(),
					Dead:       vp.dead,
					Meditating: vp.meditating,
				})
			}
		},
	)
	p.space.Set(mapdef.Players.Int(), p.pos, uint16(p.id))
	go p.HandleIncomingMessages()
	go p.HandleOutgoingMessages()
	p.m.WriteWithLen(msgs.EPlayerLogin, msgs.EncodeMsgpack(loginEvent))

	p.space.Notify(p.pos, msgs.EPlayerSpawned.U8(), &msgs.EventPlayerSpawned{
		ID:     uint16(p.id),
		Nick:   p.nick,
		Pos:    p.pos,
		Dir:    p.dir,
		Weapon: p.inv.GetWeapon(),
		Shield: p.inv.GetShield(),
		Head:   p.inv.GetHead(),
		Body:   p.inv.GetBody(),
		Speed:  uint8(p.speedPxXFrame),
	}, uint16(p.id))
}

func (p *Player) Logout() {
	p.Send <- OutMsg{Event: msgs.ECharLogoutOk, Data: nil}
	updateData := msgs.Character{
		ID:        p.characterID,
		Px:        int(p.pos.X),
		Py:        int(p.pos.Y),
		Dir:       p.dir,
		Kills:     p.kills,
		Deaths:    p.deaths,
		Skills:    p.exp.Skills,
		KeyConfig: p.keyConfigs,
		Inventory: *p.inv,
		LoggedIn:  false,
	}
	go func() {
		err := p.g.db.UpdateCharacter(updateData)
		if err != nil {
			log.Print("update character error " + err.Error())
		}
	}()
	go p.g.HandleLogin(p.m, p.account, p.characters)
	p.obs.Nuke()
	p.space.Unset(mapdef.Players.Int(), p.pos)
	p.space.Notify(p.pos, msgs.EPlayerDespawned.U8(), uint16(p.id), uint16(p.id))
}

func (p *Player) TakeDamage(dmg int32) {
	p.hp = p.hp - dmg
	if p.hp <= 0 {
		p.hp = 0
		p.dead = true
		p.paralized = false
		p.inv.UnequipAll()
	}
}

func (p *Player) Heal(heal int32) {
	p.hp = p.hp + heal
	if p.hp > p.exp.Stats.MaxHP {
		p.hp = p.exp.Stats.MaxHP
	}
}
func (p *Player) SetParalized(paralized bool) {
	p.paralized = paralized
}
func (p *Player) Dead() bool {
	return p.dead
}
func (p *Player) Revive() {
	p.dead = false
	p.hp = 1
}

func (p *Player) AddHp(heal int32) int32 {
	p.hp = p.hp + heal
	if p.hp > p.exp.Stats.MaxHP {
		p.hp = p.exp.Stats.MaxHP
	}
	return p.hp
}
func (p *Player) AddMana(mana int32) int32 {
	p.mp = p.mp + mana
	if p.mp > p.exp.Stats.MaxMP {
		p.mp = p.exp.Stats.MaxMP
	}
	return p.mp
}

func (p *Player) MultMaxMana(mana float64) int32 {
	return int32(float64(p.exp.Stats.MaxMP) * mana)
}

func (p *Player) IsParalized() bool {
	return p.paralized
}

func (g *Game) AddObjectsToSpace(space *grid.Grid, mapType mapdef.MapType) {
	var ground [][]uint32
	var stuff [][]uint32
	switch mapType {
	case mapdef.MapLobby:
		ground = mapdef.LobbyMapLayers[mapdef.Ground]
		stuff = mapdef.LobbyMapLayers[mapdef.Stuff]
	case mapdef.MapPvP1v1:
		ground = mapdef.Onev1MapLayers[mapdef.Ground]
		stuff = mapdef.Onev1MapLayers[mapdef.Stuff]
	case mapdef.MapPvP2v2:
		ground = mapdef.Twov2MapLayers[mapdef.Ground]
		stuff = mapdef.Twov2MapLayers[mapdef.Stuff]
	}
	for x := range stuff {
		for y := range stuff[x] {
			gr := ground[x][y]
			space.Set(mapdef.Ground.Int(), typ.P{X: int32(x), Y: int32(y)}, uint16(gr))
			a := stuff[x][y]
			if !assets.IsSolid(a) {
				continue
			}
			space.Set(mapdef.Stuff.Int(), typ.P{X: int32(x), Y: int32(y)}, uint16(a))
		}
	}
}
