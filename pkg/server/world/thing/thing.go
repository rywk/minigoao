package thing

type Thing interface {
	What() uint32
	Blocking() bool
	Is(uint32) bool
	Who() uint32
}

type T = uint32

const (
	Nothing T = iota
	Player
	Stuff
	Block
	Npc
	Effect
)

// Jsut a solid thing on a map
// stuff that always blocks
// and is no one
type Solid struct{}

func (s *Solid) What() uint32       { return Stuff }
func (s *Solid) Who() uint32        { return 0 }
func (s *Solid) Blocking() bool     { return true }
func (s *Solid) Is(uint32) bool     { return false }
func (s *Solid) Player(uint32) bool { return false }
