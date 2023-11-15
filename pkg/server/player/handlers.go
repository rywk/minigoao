package player

import (
	"log"
	"math/rand"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/potion"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/server/net"
	"github.com/rywk/minigoao/pkg/server/world"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/actions"
	"github.com/rywk/minigoao/proto/message/events"
	"github.com/rywk/tile"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Handler struct {
	p *Player
	c *net.Conn
	h [events.Len]func(*message.Event)
}

func NewHandlers(c *net.Conn) *Handler {
	h := &Handler{}
	h.c = c
	list := [events.Len]func(*message.Event){}
	list[events.Ping] = h.Ping
	list[events.Register] = h.Register
	list[events.Move] = h.Move
	list[events.Dir] = h.Dir
	list[events.CastMelee] = h.CastMelee
	list[events.CastSpell] = h.CastSpell
	list[events.UsePotion] = h.UsePotion
	h.h = list
	return h
}

func (h *Handler) SetPlayer(p *Player) { h.p = p }

func (h *Handler) Handle() {
	defer func() {
		// connection with client closed we need to delete from grid
		t, _ := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y))
		t.Despawn(h.p, h.p.ID, h.p.ToProto(actions.Despawn))
		h.p.View.Close()
		close(h.p.View.Inbox)
		close(h.c.Send())
		log.Printf("Client [%v, %v, %v] disconnected.\n", h.p.ID, h.p.Nick, h.c.Addr)
	}()
	// Handle connection for this player
	// The first message has to always be the registration
	h.h[events.Register](<-h.c.Recive())
	for {
		select {
		case e, ok := <-h.c.Recive():
			if !ok {
				return
			}
			h.h[e.Type](e)
		case ev := <-h.p.View.Inbox:
			if ev.Emmiter == h.p.ID {
				continue
			}
			switch m := ev.Data.(type) {
			case *message.PlayerAction:
				log.Println(m.Action)
				if m.Action == actions.Revive {
					log.Println("revive sent")
				}
				h.c.Send() <- &message.Event{
					Id:   ev.Emmiter,
					Type: events.PlayerAction,
					E:    events.Bytes(m),
				}
			case *message.MeleeHit:
				if m.To != h.p.ID {
					h.c.Send() <- &message.Event{
						Id:   ev.Emmiter,
						Type: events.MeleeHit,
						E:    events.Bytes(m),
					}
				}
			case *message.SpellHit:
				if m.To != h.p.ID {
					h.c.Send() <- &message.Event{
						Id:   ev.Emmiter,
						Type: events.SpellHit,
						E:    events.Bytes(m),
					}
				}
			}
		}
	}
}

func (h *Handler) Register(e *message.Event) {
	h.p.Nick = events.Proto(e.E, &message.Register{}).Nick
	playersVisible := h.p.Spawn()
	h.c.Send() <- events.New(events.RegisterOk, 0, events.Bytes(&message.RegisterOk{
		Id:     h.c.ID,
		MaxHP:  uint32(h.p.MaxHP),
		MaxMP:  uint32(h.p.MaxMP),
		HP:     uint32(h.p.HP),
		MP:     uint32(h.p.MP),
		FovX:   DefaultScreenWidth,
		FovY:   DefaultScreenHeight,
		Self:   h.p.ToProto(actions.Spawn),
		Spawns: playersVisible,
	}))
	log.Printf("Client [%v, %v, %v] connected.\n", h.p.ID, h.p.Nick, h.c.Addr)
}

func (h *Handler) Dir(e *message.Event) {
	d := events.Proto(e.E, &message.Dir{}).Dir
	h.p.SetDir(d)
	t, _ := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y))
	t.ChangeDirection(d, h.p.ID, h.p.ToProto(actions.Dir))
}

func (h *Handler) Move(e *message.Event) {
	d := events.Proto(e.E, &message.Move{}).Dir
	changeDir := h.p.D != d
	moved := false
	defer func() {
		// If in the end the player changed direction but didnt move
		// notify the direction change
		if changeDir && !moved {
			h.p.SetDir(d)
			t, _ := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y))
			t.ChangeDirection(d, h.p.ID, h.p.ToProto(actions.Dir))
		}
	}()
	nx, ny := h.p.X, h.p.Y
	switch d {
	case direction.Front:
		ny++
	case direction.Back:
		ny--
	case direction.Left:
		nx--
	case direction.Right:
		nx++
	}
	t, ok := world.PlayerGrid.At(int16(nx), int16(ny))
	if !ok {
		h.c.Send() <- &message.Event{Type: events.MoveOk, E: events.Bytes(&message.MoveOk{Ok: false})}
		return
	}
	pt, _ := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y))
	log.Printf("%v: %v,%v --> %v,%v\n", h.p.Nick, h.p.X, h.p.Y, nx, ny)
	if CanWalkTo(t) && !h.p.Inmobilized || h.p.Dead {
		h.p.MovePos(d, nx, ny)
		moved = MovePlayer(h.p, pt, t)
		h.c.Send() <- &message.Event{
			Id:   e.Id,
			Type: events.MoveOk,
			E: events.Bytes(&message.MoveOk{
				Ok: true,
			})}
		newPlayersInRange := []*message.PlayerAction{}
		myDespawnForOthers := h.p.ToProto(actions.Despawn)
		checkPlayers := func(a actions.A) func(tile.Point, tile.Tile[thing.Thing]) {
			return func(p tile.Point, t tile.Tile[thing.Thing]) {
				if t.Count() == 0 {
					return
				}
				t.Range(func(th thing.Thing) error {
					if pl, ok := th.(*Player); ok && !th.Is(h.p.ID) {
						msg := pl.ToProto(a)
						newPlayersInRange = append(newPlayersInRange, msg)
						if a == actions.Despawn {
							go func() {
								pl.Handler.SendDespawnFrom(h.p.ID, myDespawnForOthers)
							}()
						}
					}
					return nil
				})
			}
		}
		h.p.View.Resize(h.p.ViewRect, checkPlayers(actions.Spawn), checkPlayers(actions.Despawn))

		if len(newPlayersInRange) != 0 {
			h.SendPlayerActions(newPlayersInRange)
		}
		return
	}
	h.c.Send() <- &message.Event{
		Id:   e.Id,
		Type: events.MoveOk,
		E: events.Bytes(&message.MoveOk{
			Ok: false,
		})}

}

func (h *Handler) CastMelee(e *message.Event) {
	if h.p.Dead {
		h.Send(events.CastMeleeOk, &message.CastMeleeOk{Ok: false})
		return
	}
	px, py := h.p.GetPos()
	nx, ny := px, py
	switch h.p.D {
	case direction.Front:
		ny++
	case direction.Back:
		ny--
	case direction.Left:
		nx--
	case direction.Right:
		nx++
	}
	pt, _ := world.PlayerGrid.At(int16(px), int16(py))
	t, ok := world.PlayerGrid.At(int16(nx), int16(ny))
	if !ok {
		h.Send(events.CastMeleeOk, &message.CastMeleeOk{Ok: false})
		pt.TriggerEvent(h.p.ID, &message.MeleeHit{Ok: false, From: h.p.ID})
		return
	}
	var p *Player
	t.Range(func(th thing.Thing) error {
		if p, ok = th.(*Player); ok {
			return constants.Err{}
		}
		return nil
	})
	if p != nil && !p.Dead {
		// we hit someone alive
		damage := 70 + rand.Intn(60)
		hpLeft := p.DamagePlayer(damage)
		h.Send(events.CastMeleeOk, &message.CastMeleeOk{Ok: true, Id: p.ID, Dmg: uint32(damage)})
		p.Handler.Send(events.RecivedMelee, &message.RecivedMelee{Id: h.p.ID, Dmg: uint32(damage), Hp: uint32(hpLeft)})
		pt.TriggerEvent(h.p.ID, &message.MeleeHit{Ok: true, From: h.p.ID, To: p.ID})
		if p.IsDead() {
			t.TriggerEvent(p.ID, p.ToProto(actions.Died))
		}
		return
	}
	// no one is there
	h.Send(events.CastMeleeOk, &message.CastMeleeOk{Ok: false})
	pt.TriggerEvent(h.p.ID, &message.MeleeHit{Ok: false, From: h.p.ID})
}

var (
	spellConfig = map[spell.Spell]Spell{
		spell.Revive:     NewRevive(1300),
		spell.HealWounds: NewHealWounds(650, 40, 10),
		spell.InmoRm:     NewInmoRm(370),
		spell.Inmo:       NewInmo(300, time.Second*6),
		spell.Apoca:      NewDamageSpell(1100, 170, 20),
		spell.Desca:      NewDamageSpell(700, 90, 15),
	}
)

func SpellHandle(s spell.Spell, caster, reciver *Player) (ok bool, dmg int) {
	sp, ok := spellConfig[s]
	if !ok {
		return false, 0
	}
	if caster.ID == reciver.ID && !sp.CanSelfCast() {
		return false, 0
	}
	return sp.Cast(caster, reciver)
}

func (h *Handler) CastSpell(e *message.Event) {
	cs := events.Proto(e.E, &message.CastSpell{})
	isDead := h.p.IsDead()
	if isDead {
		h.Send(events.CastSpellOk, &message.CastSpellOk{Ok: false})
		return
	}
	log.Println(cs.X, cs.Y)
	t, ok := world.PlayerGrid.At(int16(cs.X), int16(cs.Y))
	if !ok {
		h.Send(events.CastSpellOk, &message.CastSpellOk{Ok: false})
		return
	}
	var p *Player
	t.Range(func(th thing.Thing) error {
		if p, ok = th.(*Player); ok {
			return constants.Err{}
		}
		return nil
	})
	if p != nil {
		if !p.IsDead() {
			// we hit someone alive, we had mana for the spell
			if cs.Spell == spell.InmoRm && !p.Inmobilized {
				h.Send(events.CastSpellOk, &message.CastSpellOk{Ok: false})
				return
			}
			if ok, dmg := SpellHandle(cs.Spell, h.p, p); ok {
				h.Send(events.CastSpellOk, &message.CastSpellOk{Ok: true, Id: p.ID, Dmg: uint32(dmg), Mp: uint32(h.p.MP), Spell: cs.Spell})
				p.Handler.Send(events.RecivedSpell, &message.RecivedSpell{Id: h.p.ID, Dmg: uint32(dmg), Hp: uint32(p.HP), Spell: cs.Spell})
				t.TriggerEvent(h.p.ID, &message.SpellHit{From: h.p.ID, To: p.ID, Spell: cs.Spell})
				if p.IsDead() {
					t.TriggerEvent(p.ID, p.ToProto(actions.Died))
				}
				return
			}
		} else if cs.Spell == spell.Revive {
			if ok, dmg := SpellHandle(cs.Spell, h.p, p); ok {
				h.Send(events.CastSpellOk, &message.CastSpellOk{Ok: true, Id: p.ID, Dmg: uint32(dmg), Mp: uint32(h.p.MP), Spell: cs.Spell})
				p.Handler.Send(events.RecivedSpell, &message.RecivedSpell{Id: h.p.ID, Dmg: uint32(dmg), Hp: uint32(p.HP), Spell: cs.Spell})
				t.TriggerEvent(h.p.ID, &message.SpellHit{From: h.p.ID, To: p.ID, Spell: cs.Spell})
				t.TriggerEvent(p.ID, p.ToProto(actions.Revive))
				return
			}
		}
	}
	// no one is there
	h.Send(events.CastSpellOk, &message.CastSpellOk{Ok: false})
}

const (
	HealthPotionRestoreValue = 30
	ManaPotionRestoreValue   = .05 // of total
)

func (h *Handler) UsePotion(e *message.Event) {
	h.p.PotionLock.Lock()
	defer h.p.PotionLock.Unlock()
	pot := events.Proto(e.E, &message.UsePotion{})
	if h.p.Dead {
		h.Send(events.UsePotionOk, &message.UsePotionOk{Ok: false})
		return
	}
	switch pot.Type {
	case potion.Red:
		val := HealthPotionRestoreValue
		h.p.HealPlayer(val)
		h.Send(events.UsePotionOk, &message.UsePotionOk{Ok: true, NewHP: uint32(h.p.GetHP())})
		if t, ok := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y)); ok {
			t.TriggerEvent(h.p.ID, &message.PotionUsed{X: uint32(h.p.X), Y: uint32(h.p.Y)})
		}
	case potion.Blue:
		val := int(float64(h.p.MaxMP) * ManaPotionRestoreValue)
		h.p.AddMana(val)
		h.Send(events.UsePotionOk, &message.UsePotionOk{Ok: true, NewMP: uint32(h.p.GetMP())})
		if t, ok := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y)); ok {
			t.TriggerEvent(h.p.ID, &message.PotionUsed{X: uint32(h.p.X), Y: uint32(h.p.Y)})
		}
	default:
		log.Println("NON EXISTENT POTION", pot.Type)
	}
}

func (h *Handler) Ping(e *message.Event) {
	h.c.Send() <- &message.Event{Type: events.Ping, Id: h.p.ID,
		E: events.Bytes(&message.Ping{}),
	}
}

func (h *Handler) SendDespawnFrom(id uint32, e *message.PlayerAction) {
	h.c.Send() <- &message.Event{Id: id, Type: events.PlayerAction,
		E: events.Bytes(e)}
}

func (h *Handler) SendPlayerActions(e []*message.PlayerAction) {
	h.c.Send() <- &message.Event{Id: h.p.ID, Type: events.PlayerActions,
		E: events.Bytes(&message.PlayerActions{PlayerActions: e})}
}

func (h *Handler) Send(e events.E, data protoreflect.ProtoMessage) {
	h.c.Send() <- &message.Event{Type: e, Id: h.p.ID,
		E: events.Bytes(data),
	}
}
