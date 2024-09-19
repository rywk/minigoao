package grid

import (
	"fmt"
	"log"
	"sync"

	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
)

type Grid struct {
	layers uint8
	w, h   int32
	grid   [][]Tile
	Rect   typ.Rect
}

func NewGrid(w, h int32, layers uint8) *Grid {
	// 0 <-> w-1.
	// 0 <-> h-1
	s := Grid{
		w:      w,
		h:      h,
		layers: layers,
		grid:   make([][]Tile, w),
		Rect:   typ.Rect{Min: typ.P{X: 0, Y: 0}, Max: typ.P{X: w, Y: h}},
	}

	for i := range s.grid {
		s.grid[i] = make([]Tile, h)
		for j := range s.grid[i] {
			s.grid[i][j].p.Set(int32(i), int32(j))
			s.grid[i][j].Layers = make([]uint16, layers)
			s.grid[i][j].obs = []chan Event{}
		}
	}
	return &s
}

func (s *Grid) Get(p typ.P) (*Tile, func()) {
	t := &s.grid[p.X][p.Y]
	t.m.RLock()
	return t, t.m.RUnlock
}

func (s *Grid) Update(p typ.P) (*Tile, func()) {
	t := &s.grid[p.X][p.Y]
	t.m.Lock()
	return t, t.m.Unlock
}

func (s *Grid) Notify(p typ.P, e msgs.E, ev interface{}, ids ...uint16) {
	s.grid[p.X][p.Y].NotifyTile(Event{ids, p, e, ev})
}

type Tile struct {
	p      typ.P
	m      sync.RWMutex
	Layers []uint16

	obsLock sync.RWMutex
	obs     []chan Event
}

func (l *Tile) AddObserver(c chan Event) {
	l.obsLock.Lock()
	defer l.obsLock.Unlock()
	l.obs = append(l.obs, c)
}

func (l *Tile) RemoveObserver(c chan Event) {
	l.obsLock.Lock()
	defer l.obsLock.Unlock()
	l.obs = removeObs(l.obs, c)
}

func (l *Tile) NotifyTile(ev Event) {
	l.obsLock.RLock()
	defer l.obsLock.RUnlock()
	for _, c := range l.obs {
		c <- ev
	}
}

type GridError int

const (
	ErrorOutOfBounds    = "out of bounds"
	ErrorMoveNotAllowed = "cannot move there"
	ErrorSlotOccupied   = "cannot move there, slots is in use"
)

var gridErrors = []string{
	"out of bounds",
	"cannot move there",
	"cannot move there, slots is in use",
}

func (ge GridError) Error() string {
	return gridErrors[ge]
}

func (g *Grid) GetSlot(layer int, p typ.P) uint16 {
	return g.grid[p.X][p.Y].Layers[layer]
}

func (g *Grid) Move(layer int, p typ.P, np typ.P) error {
	if !exist(np, typ.P{}, g.w, g.h) {
		return fmt.Errorf("%s: from %v to %v", ErrorOutOfBounds, p, np)
	}
	to, unlockTo := g.Update(np)
	defer unlockTo()
	if to.Layers[layer] != 0 {
		return fmt.Errorf("%s: from %v to %v", ErrorSlotOccupied, p, np)
	}
	from, unlockFrom := g.Update(p)
	defer unlockFrom()
	to.Layers[layer] = from.Layers[layer]
	from.Layers[layer] = 0
	return nil
}

func (g *Grid) Set(layer int, np typ.P, id uint16) error {
	if !exist(np, typ.P{}, g.w, g.h) {
		return fmt.Errorf("%s: on %v", ErrorOutOfBounds, np)
	}
	on, unlock := g.Update(np)
	defer unlock()
	if on.Layers[layer] != 0 {
		return fmt.Errorf("%s: on %v", ErrorSlotOccupied, np)
	}
	on.Layers[layer] = id
	return nil
}

func (g *Grid) Unset(layer int, np typ.P) error {
	if !exist(np, typ.P{}, g.w, g.h) {
		return fmt.Errorf("%s: on %v", ErrorOutOfBounds, np)
	}
	on, unlock := g.Update(np)
	defer unlock()
	on.Layers[layer] = 0
	return nil
}

func removeObs(obss []chan Event, obs chan Event) []chan Event {
	i := 0
	for _, o := range obss {
		if obs != o {
			obss[i] = o
			i++
		}
	}
	// theres no need, those addresses should still be used
	// just somewhere else..
	// what i dont know is; if its worth handling this on the last removal 🤔
	for j := i; j < len(obss); j++ {
		obss[j] = nil
	}
	return obss[:i]
}

func exist(p, pos typ.P, w, h int32) bool {
	if p.X+pos.X < 0 || p.X+pos.X > w-1 || p.Y+pos.Y < 0 || p.Y+pos.Y > h-1 {
		return false
	}
	return true
}

type Obs struct {
	s               *Grid
	Pos             typ.P // top left
	Width, WidthR   int32
	Height, HeightR int32
	View            typ.Rect
	Events          chan Event
}

func NewObserver(s *Grid, pos typ.P, w, h int32) *Obs {
	if w%2 == 0 || h%2 == 0 {
		panic("observer must have an odd width and height")
	}
	o := &Obs{
		s:       s,
		Pos:     pos,
		Width:   w,
		WidthR:  w >> 1,
		Height:  h,
		HeightR: h >> 1,
		Events:  make(chan Event, w*h), // every tile sends a message at the same time we dont block
	}
	sx, sy := pos.X-o.WidthR, pos.Y-o.HeightR
	ex, ey := pos.X+o.WidthR, pos.Y+o.HeightR
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}
	if ex >= int32(s.w) {
		ex = int32(s.w - 1)
	}
	if ey >= int32(s.h) {
		ey = int32(s.h - 1)
	}
	for x := sx; x <= ex; x++ {
		for y := sy; y <= ey; y++ {
			s.grid[x][y].AddObserver(o.Events)
		}
	}
	return o
}

func NewObserverRange(s *Grid, pos typ.P, w, h int32, fn func(*Tile)) *Obs {
	if w%2 == 0 || h%2 == 0 {
		panic("observer must have an odd width and height")
	}
	o := &Obs{
		s:       s,
		Pos:     pos,
		Width:   w,
		WidthR:  w >> 1,
		Height:  h,
		HeightR: h >> 1,
		Events:  make(chan Event, w*h), // every tile sends a message at the same time we dont block
	}
	sx, sy := pos.X-o.WidthR, pos.Y-o.HeightR
	ex, ey := pos.X+o.WidthR, pos.Y+o.HeightR
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}
	if ex >= int32(s.w) {
		ex = int32(s.w - 1)
	}
	if ey >= int32(s.h) {
		ey = int32(s.h - 1)
	}
	log.Printf("pos [%v] ", pos)
	log.Printf("start xy [%v %v] ", sx, sy)
	log.Printf("end xy [%v %v] ", ex, ey)
	o.View = typ.Rect{
		Min: typ.P{X: sx, Y: sy},
		Max: typ.P{X: ex, Y: ey},
	}
	for x := sx; x <= ex; x++ {
		for y := sy; y <= ey; y++ {
			t, unlock := s.Get(typ.P{X: x, Y: y})
			fn(t)
			unlock()
			t.AddObserver(o.Events)
		}
	}
	return o
}

func (o *Obs) MoveOne(d direction.D, fnin, fnout func(x, y int32)) {
	oldPos := o.Pos
	topX := oldPos.X + o.WidthR
	topY := oldPos.Y + o.HeightR
	botX := oldPos.X - o.WidthR
	botY := oldPos.Y - o.HeightR
	if topX >= o.s.w {
		topX = o.s.w - 1
	}
	if topY >= o.s.h {
		topY = o.s.h - 1
	}
	if botX < 0 {
		botX = 0
	}
	if botY < 0 {
		botY = 0
	}
	out, in := int32(0), int32(0)
	switch d {
	case direction.Back, direction.Front:
		if d == direction.Front {
			out, in = botY, topY+1
			o.Pos.Y++
			o.View.Move(typ.P{X: 0, Y: 1})
		} else {
			out, in = topY, botY-1
			o.Pos.Y--
			o.View.Move(typ.P{X: 0, Y: -1})
		}
		if in < 0 || in >= o.s.h {
			for x := botX; x <= topX; x++ {
				o.s.grid[x][out].RemoveObserver(o.Events)
				fnout(x, out)
			}
			return
		}
		if out < 0 || out >= o.s.h {
			for x := botX; x <= topX; x++ {
				o.s.grid[x][in].AddObserver(o.Events)
				fnin(x, in)
			}
			return
		}
		for x := botX; x <= topX; x++ {
			o.s.grid[x][out].RemoveObserver(o.Events)
			o.s.grid[x][in].AddObserver(o.Events)
			fnin(x, in)
			fnout(x, out)
		}
	case direction.Left, direction.Right:
		if d == direction.Right {
			out, in = botX, topX+1
			o.Pos.X++
			o.View.Move(typ.P{X: 1, Y: 0})
		} else {
			out, in = topX, botX-1
			o.Pos.X--
			o.View.Move(typ.P{X: -1, Y: 0})
		}
		if in < 0 || in >= o.s.w {
			for y := botY; y <= topY; y++ {
				o.s.grid[out][y].RemoveObserver(o.Events)
				fnout(out, y)
			}
			return
		}
		if out < 0 || out >= o.s.w {
			for y := botY; y <= topY; y++ {
				o.s.grid[in][y].AddObserver(o.Events)
				fnin(in, y)
			}
			return
		}
		for y := botY; y <= topY; y++ {
			o.s.grid[out][y].RemoveObserver(o.Events)
			o.s.grid[in][y].AddObserver(o.Events)
			fnin(in, y)
			fnout(out, y)
		}
	default:
		panic(fmt.Sprintf("not a direction %v", d))
	}
}

func (o *Obs) Nuke() {
	sx, sy := o.Pos.X-o.WidthR, o.Pos.Y-o.HeightR
	ex, ey := o.Pos.X+o.WidthR, o.Pos.Y+o.HeightR
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}
	if ex >= int32(o.s.w) {
		ex = int32(o.s.w - 1)
	}
	if ey >= int32(o.s.h) {
		ey = int32(o.s.h - 1)
	}

	for y := sy; y <= ey; y++ {
		for x := sx; x <= ex; x++ {
			o.s.grid[x][y].RemoveObserver(o.Events)
		}
	}
	close(o.Events)
}

type Event struct {
	ID   []uint16
	From typ.P
	E    msgs.E
	Data interface{}
}

func (e Event) HasID(id uint16) bool {
	for _, i := range e.ID {
		if i == id {
			return true
		}
	}
	return false
}
