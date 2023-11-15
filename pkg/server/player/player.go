package player

import (
	"log"
	"sync"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/server/net"
	"github.com/rywk/minigoao/pkg/server/world"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/actions"
	asset "github.com/rywk/minigoao/proto/message/assets"
	"github.com/rywk/tile"
)

const (
	MaxHP = 300
	MaxMP = 2500

	StartHP        = 300
	StartMP        = 2500
	StartX, StartY = 70, 70
	DefaultHead    = asset.ProHat
	DefaultBody    = asset.DarkArmour
	DefaultWeapon  = asset.SpecialSword
	DefaultShield  = asset.SilverShield

	DefaultScreenWidth  = 35
	DefaultScreenHeight = 25
)

type Player struct {
	*sync.RWMutex
	Online bool
	// Player stats and stuff..
	// Player ID
	ID uint32
	// Player Nick in-game
	Nick string
	// Player position on map
	X, Y int
	D    direction.D
	// Player health points
	HP, MaxHP int
	// Player mana points
	MP, MaxMP int

	// Dead
	Dead bool
	// Inmobilized
	Inmobilized bool

	// IDs for skins
	Armor  asset.Image
	Helmet asset.Image
	Weapon asset.Image
	Shield asset.Image

	PotionLock      *sync.Mutex
	WalkLock        *sync.RWMutex
	InmobilizedLock *sync.RWMutex

	// Internal player
	Handler  *Handler
	View     *tile.View[thing.Thing]
	ViewRect tile.Rect
}

func RunPlayer(c *net.Conn) {
	p := &Player{
		RWMutex:         &sync.RWMutex{},
		ID:              c.ID,
		X:               StartX,
		Y:               StartY,
		D:               direction.Front,
		HP:              StartHP,
		MP:              StartMP,
		MaxHP:           MaxHP,
		MaxMP:           MaxMP,
		Helmet:          DefaultHead,
		Armor:           DefaultBody,
		Weapon:          DefaultWeapon,
		Shield:          DefaultShield,
		Handler:         NewHandlers(c),
		WalkLock:        &sync.RWMutex{},
		InmobilizedLock: &sync.RWMutex{},
		PotionLock:      &sync.Mutex{},
	}
	p.Handler.SetPlayer(p)
	p.Handler.Handle()
	log.Printf("Player %v disconnected.\n", p.Nick)
}

func (p *Player) Blocking() bool {
	return !p.Dead
}

func (p *Player) Who() uint32 {
	return p.ID
}

func (p *Player) What() uint32 {
	return thing.Player
}

func (p *Player) Is(id uint32) bool {
	return p.ID == id
}

func spawnPlayer(p *Player) (int, int) {
	t, ok := world.PlayerGrid.At(int16(p.X), int16(p.Y))
	if !ok {
		// I think !ok means outside of the grid so no moving there
		log.Println("ERROR SPAWN !!")
		return 0, 0
	}
	if CanWalkTo(t) {
		t.Spawn(p, p.ID, p.ToProto(actions.Spawn))
		return p.X, p.Y
	}
	p.X++
	return spawnPlayer(p)
}

func (p *Player) Spawn() []*message.PlayerAction {
	p.X, p.Y = spawnPlayer(p)
	log.Println("spawning at", p.X, p.Y)
	// Create the view of the tiles for the player
	// and get all the players in view
	p.ViewRect = playerView(p.X, p.Y)
	log.Printf("Spawn view rect :%#v\n", p.ViewRect)
	playersInRange := []*message.PlayerAction{}
	p.View = world.PlayerGrid.View(p.ViewRect, func(pp tile.Point, t tile.Tile[thing.Thing]) {
		if t.Count() == 0 {
			return
		}
		t.Range(func(th thing.Thing) error {
			if pl, ok := th.(*Player); ok && !th.Is(p.ID) {
				playersInRange = append(playersInRange, pl.ToProto(actions.Spawn))
			}
			return nil
		})
	})
	return playersInRange
}

func (p *Player) ToProto(a actions.A) *message.PlayerAction {
	return &message.PlayerAction{
		Action: a,
		Id:     p.ID,
		X:      uint32(p.X),
		Y:      uint32(p.Y),
		D:      p.D,
		Nick:   p.Nick,
		Dead:   p.Dead,
		Armor:  p.Armor,
		Helmet: p.Helmet,
		Weapon: p.Weapon,
		Shield: p.Shield,
	}
}

func TryMovePlayer(p *Player, pt, t tile.Tile[thing.Thing]) bool {
	if CanWalkTo(t) {
		return MovePlayer(p, pt, t)
	}
	return false
}

func CanWalkTo(t tile.Tile[thing.Thing]) bool {
	return t.Range(func(th thing.Thing) error {
		if th != nil && th.Blocking() {
			return constants.Err{}
		}
		return nil
	}) == nil
}

func MovePlayer(p *Player, pt, t tile.Tile[thing.Thing]) bool {
	pt.MoveTo(t, p, p.ID, p.ToProto(actions.Move))
	return true
}

func (p *Player) IsDead() bool {
	p.RLock()
	defer p.RUnlock()
	return p.Dead
}

func (p *Player) SetDir(d direction.D) {
	p.WalkLock.Lock()
	p.D = d
	p.WalkLock.Unlock()
}

func (p *Player) GetDir() direction.D {
	p.WalkLock.RLock()
	defer p.WalkLock.RUnlock()
	return p.D
}

func (p *Player) GetPos() (int, int) {
	p.WalkLock.RLock()
	defer p.WalkLock.RUnlock()
	return p.X, p.Y
}

func (p *Player) MovePos(d direction.D, nx, ny int) {
	p.WalkLock.Lock()
	p.X, p.Y = nx, ny
	p.D = d
	p.ViewRect = playerView(nx, ny)
	p.WalkLock.Unlock()
}

func (p *Player) DamagePlayer(dmg int) int {
	p.Lock()
	defer p.Unlock()
	p.HP = p.HP - dmg
	if p.HP <= 0 {
		p.HP = 0
		p.Dead = true
	}
	return p.HP
}

func (p *Player) GetHP() int {
	p.RLock()
	defer p.RUnlock()
	return p.HP
}

func (p *Player) GetMP() int {
	p.RLock()
	defer p.RUnlock()
	return p.MP
}

func (p *Player) HealPlayer(heal int) int {
	p.Lock()
	defer p.Unlock()
	p.HP = p.HP + heal
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
	return p.HP
}

func (p *Player) AddMana(mana int) int {
	p.Lock()
	defer p.Unlock()
	p.MP = p.MP + mana
	if p.MP > p.MaxMP {
		p.MP = p.MaxMP
	}
	return p.MP
}

func (p *Player) RevivePlayer() {
	p.Lock()
	defer p.Unlock()
	p.HP = p.MaxHP
	p.Dead = false
}

func (p *Player) IsInmobilized() bool {
	p.InmobilizedLock.RLock()
	defer p.InmobilizedLock.RUnlock()
	return p.Inmobilized
}

func (p *Player) ChangeInmobilized(inmo bool) {
	p.InmobilizedLock.Lock()
	defer p.InmobilizedLock.Unlock()
	p.Inmobilized = inmo
}

func playerView(x, y int) tile.Rect {
	sx, sy := int16(x-DefaultScreenWidth),
		int16(y-DefaultScreenHeight)
	ex, ey := int16(x+DefaultScreenWidth),
		int16(y+DefaultScreenHeight)
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}
	if ex < 0 {
		ex = 0
	}
	if ey < 0 {
		ey = 0
	}
	if sx > constants.WorldX {
		sx = constants.WorldX
	}
	if sy > constants.WorldY {
		sy = constants.WorldY
	}
	if ex > constants.WorldX {
		ex = constants.WorldX
	}
	if ey > constants.WorldY {
		ey = constants.WorldY
	}
	r := tile.NewRect(sx, sy, ex, ey)
	return r
}
