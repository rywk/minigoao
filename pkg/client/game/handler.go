package game

import (
	"fmt"
	"log"
	"time"

	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/events"
)

type Handler struct {
	g      *Game
	h      [events.Len]func(*message.Event)
	Events chan *message.Event
	ping   time.Time
}

func NewHandler(g *Game) *Handler {
	h := &Handler{g: g}
	h.Events = make(chan *message.Event, 10)
	h.h = [events.Len]func(*message.Event){}
	h.h[events.Ping] = h.Ping

	h.h[events.RegisterOk] = h.RegisterOk
	h.h[events.MoveOk] = h.MoveOk

	h.h[events.CastMeleeOk] = h.CastMeleeOk
	h.h[events.RecivedMelee] = h.RecivedMelee
	h.h[events.MeleeHit] = h.MeleeHit

	h.h[events.CastSpellOk] = h.CastSpellOk
	h.h[events.RecivedSpell] = h.RecivedSpell
	h.h[events.SpellHit] = h.SpellHit

	h.h[events.UsePotionOk] = h.UsePotionOk
	h.h[events.PotionUsed] = h.PotionUsed

	h.h[events.PlayerAction] = h.PlayerAction
	h.h[events.PlayerActions] = h.PlayerActions
	return h
}

func (h *Handler) TCP() {
	var e *message.Event
	var err error
	for {
		e, err = h.g.m.Read()
		if err != nil {
			break
		}
		h.Events <- e
	}
	close(h.Events)
	log.Println("Stopped reading tcp messages", err)
}

func (h *Handler) SendRegister(nick string) {
	h.g.m.Write(&message.Event{
		Type: events.Register,
		E: events.Bytes(&message.Register{
			Nick: nick,
		}),
	})
}

func (h *Handler) Start() {
	for e := range h.Events {
		h.h[e.Type](e)
	}
}
func (h *Handler) RegisterOk(e *message.Event) {
	log.Println("RegisterOk")
	rok := events.Proto(e.E, &message.RegisterOk{})
	h.g.sessionID = rok.Id
	h.g.world = NewMap(MapConfigFromRegisterOk(rok))
	h.g.player, h.g.client = player.NewRegisterOk(rok)
	players := player.NewFromLogIn(h.g.player, rok, h.g.SoundBoard)
	for id, p := range players {
		h.g.players[id] = p
		h.g.playersY = append(h.g.playersY, p)
	}
	log.Println("RegisterOk finished")
}

// Event responses

func (h *Handler) MoveOk(e *message.Event) {
	h.g.client.MoveOk <- events.Proto(e.E, &message.MoveOk{}).Ok
}

func (h *Handler) CastMeleeOk(e *message.Event) {
	h.g.client.CastMeleeOk <- events.Proto(e.E, &message.CastMeleeOk{})
}

func (h *Handler) MeleeHit(e *message.Event) {
	h.g.client.MeleeHit <- events.Proto(e.E, &message.MeleeHit{})
}

func (h *Handler) RecivedMelee(e *message.Event) {
	h.g.client.RecivedMelee <- events.Proto(e.E, &message.RecivedMelee{})
}

func (h *Handler) CastSpellOk(e *message.Event) {
	h.g.client.CastSpellOk <- events.Proto(e.E, &message.CastSpellOk{})
}

func (h *Handler) SpellHit(e *message.Event) {
	h.g.client.SpellHit <- events.Proto(e.E, &message.SpellHit{})
}

func (h *Handler) RecivedSpell(e *message.Event) {
	h.g.client.RecivedSpell <- events.Proto(e.E, &message.RecivedSpell{})
}

func (h *Handler) UsePotionOk(e *message.Event) {
	h.g.client.UsePotionOk <- events.Proto(e.E, &message.UsePotionOk{})
}

func (h *Handler) PotionUsed(e *message.Event) {
	h.g.client.PotionUsed <- events.Proto(e.E, &message.PotionUsed{})
}

// Actions

func (h *Handler) PlayerAction(e *message.Event) {
	h.playerAction(events.Proto(e.E, &message.PlayerAction{}))
}

func (h *Handler) PlayerActions(e *message.Event) {
	pas := events.Proto(e.E, &message.PlayerActions{})
	for _, pa := range pas.PlayerActions {
		h.playerAction(pa)
	}
}

func (h *Handler) playerAction(a *message.PlayerAction) {
	p := h.g.players[a.Id]
	if !p.Nil() {
		p.Process(a)
		return
	}
	h.g.players[a.Id] = player.ProcessNew(h.g.player, a, h.g.SoundBoard)
	h.g.playersY = append(h.g.playersY, h.g.players[a.Id])
	h.g.players[a.Id].Tile = MustAt(h.g.players[a.Id].X, h.g.players[a.Id].Y)
	h.g.players[a.Id].Tile.SimpleSpawn(h.g.players[a.Id])
}

func (h *Handler) HandleClient() {
	for {
		select {
		case msg := <-h.g.client.Dir:
			h.g.m.Write(&message.Event{Type: events.Dir, Id: h.g.sessionID,
				E: events.Bytes(&message.Dir{Dir: msg}),
			})
		case msg := <-h.g.client.Move:
			h.g.m.Write(&message.Event{Type: events.Move, Id: h.g.sessionID,
				E: events.Bytes(&message.Move{Dir: msg}),
			})
		case <-h.g.client.CastMelee:
			h.g.m.Write(&message.Event{Type: events.CastMelee, Id: h.g.sessionID})

		case msg := <-h.g.client.CastSpell:
			h.g.m.Write(&message.Event{Type: events.CastSpell, Id: h.g.sessionID, E: events.Bytes(msg)})

		case msg := <-h.g.client.UsePotion:
			h.g.m.Write(&message.Event{Type: events.UsePotion, Id: h.g.sessionID, E: events.Bytes(msg)})
		}
	}
}

func (h *Handler) SendPing() {
	for range time.Tick(time.Second * 2) {
		h.ping = time.Now()
		h.g.m.Write(&message.Event{Type: events.Ping, Id: h.g.sessionID, E: events.Bytes(&message.Ping{})})
	}
}

func (h *Handler) Ping(e *message.Event) {
	//log.Printf("ping response arrived %v", time.Since(h.ping).String())
	h.g.latency = fmt.Sprintf("%vms", time.Since(h.ping).Milliseconds())
}
