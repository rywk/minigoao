package player

import (
	"log"

	"github.com/rywk/minigoao/pkg/direction"
	"github.com/rywk/minigoao/pkg/server/net"
	"github.com/rywk/minigoao/pkg/server/world"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/actions"
	"github.com/rywk/minigoao/proto/message/events"
	"github.com/rywk/tile"
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
	list[events.CastSpell] = h.CastSpell
	list[events.HitMelee] = h.HitMelee
	h.h = list
	return h
}

func (h *Handler) SetPlayer(p *Player) { h.p = p }

func (h *Handler) Handle() {
	// Handle connection for this player
	for e := range h.c.Recive() {
		h.h[e.Type](e)
	}

	// connection with client closed we need to delete from grid
	t, _ := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y))
	t.Despawn(h.p, h.p.ID, h.p.ToProto(actions.Despawn))
	h.p.View.Close()
	close(h.p.View.Inbox)
	log.Printf("START despawn %v\n", h.p.Nick)
}

func (h *Handler) HandleMapEvents() {
	for ev := range h.p.View.Inbox {
		if ev.Emmiter == h.p.ID {
			continue
		}
		pa, ok := ev.Data.(*message.PlayerAction)
		if !ok {
			continue
		}
		h.c.Send() <- &message.Event{
			Id:   ev.Emmiter,
			Type: events.PlayerAction,
			E:    events.Bytes(pa),
		}
	}
	close(h.c.Send())
	log.Printf("exited %v event reciver\n", h.p.Nick)
}

func (h *Handler) Register(e *message.Event) {
	h.p.Nick = events.Proto(e.E, &message.Register{}).Nick
	log.Printf("Client [%v, %v, %v] start to register\n", h.p.ID, h.p.Nick, h.c.Addr)
	playersVisible := h.p.Spawn()
	go h.HandleMapEvents()
	rp := &message.RegisterOk{
		Id:     h.c.ID,
		FovX:   DefaultScreenWidth,
		FovY:   DefaultScreenHeight,
		Self:   h.p.ToProto(actions.Spawn),
		Spawns: playersVisible,
	}
	h.c.Send() <- &message.Event{Type: events.RegisterOk, E: events.Bytes(rp)}
	log.Printf("Client [%v, %v, %v] finish register\n", h.p.ID, h.p.Nick, h.c.Addr)
}

func (h *Handler) Dir(e *message.Event) {
	d := events.Proto(e.E, &message.Dir{}).Dir
	h.p.D = d
	t, _ := world.PlayerGrid.At(int16(h.p.X), int16(h.p.Y))
	t.ChangeDirection(d, h.p.ID, h.p.ToProto(actions.Dir))
}

func (h *Handler) Move(e *message.Event) {
	h.p.WalkLock.Lock()
	d := events.Proto(e.E, &message.Move{}).Dir
	changeDir := h.p.D != d
	moved := false
	defer func() {
		h.p.WalkLock.Unlock()
		// If in the end the player changed direction but didnt move
		// notify the direction change
		if changeDir && !moved {
			h.p.D = d
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
	log.Println(h.p.X, h.p.Y, "-->", nx, ny)
	if CanWalkTo(t) {
		h.p.X, h.p.Y = nx, ny
		h.p.D = d
		moved = MovePlayer(h.p, pt, t)
		h.c.Send() <- &message.Event{
			Id:   e.Id,
			Type: events.MoveOk,
			E: events.Bytes(&message.MoveOk{
				Ok: true,
			})}
		newPlayersInRange := []*message.PlayerAction{}
		h.p.ViewRect = playerView(nx, ny)
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

func (h *Handler) CastSpell(e *message.Event) {

}

func (h *Handler) HitMelee(e *message.Event) {

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
