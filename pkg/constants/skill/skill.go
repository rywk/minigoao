package skill

import (
	"fmt"
	"time"
)

type Skill uint8

const (
	None Skill = iota
	Agility
	Intelligence
	Vitality
	Max
)

type Value float64

type Skills [Max]Value

func (sk Skills) Total() Value {
	var t Value
	for i := range Max {
		t += sk[i]
	}
	return t
}

type Buff uint8

const (
	BuffNothing Buff = iota
	BuffMagicDamage
	BuffPhysicalDamage
	BuffMagicDefense
	BuffPhysicalDefense
	BuffLen
)

var buffStr = [BuffLen]string{
	"BuffNothing",
	"MagicAtk",  // Spell Damage
	"PhysicAtk", // Melee Damage
	"MagicDef",  // Spell Resistence
	"PhysicDef", // Melee Resistence
}

func (b Buff) String() string {
	return buffStr[b]
}

type Buffs [BuffLen]Value

func (b Buffs) String() string {
	s := ""
	for i := range BuffLen {
		if b[i] <= 0 {
			continue
		}
		s += fmt.Sprintf("(%v +%d) ", i.String(), int(b[i]))
	}
	return s
}
func (s Buffs) AddValue(b Buff, v Value) Buffs {
	s[b] += v
	return s
}
func (s Buffs) SubTo(b Buff, v Value) {
	s[b] -= v
}
func (s Buffs) Add(sk Buffs) Buffs {
	for isk := range BuffLen {
		s[isk] = s[isk] + sk[isk]
	}
	return s
}

// Player Stats
type Stats struct {
	MaxHP    int32
	MaxMP    int32
	ActionCD time.Duration
}

// Flatters
const (
	//actionCDF Value = 0.016
	healthF Value = 1.9
	manaF   Value = 19
)

// Base stats
const (
	BaseHP             = 330
	BaseMP             = 1300
	BaseActionCooldown = time.Millisecond * 1000
)

func (s Skills) Stats() Stats {
	stats := Stats{
		ActionCD: BaseActionCooldown - time.Duration(s[Intelligence]*Value(time.Millisecond)*0.5),
		MaxHP:    int32(BaseHP + s[Vitality]*healthF),
		MaxMP:    int32(BaseMP + s[Intelligence]*manaF),
	}
	return stats
}

func (s Skills) Buffs() Buffs {
	b := Buffs{}

	b[BuffPhysicalDamage] += s[Vitality]
	b[BuffPhysicalDamage] -= s[Intelligence] * 0.4

	b[BuffMagicDamage] += s[Intelligence] * 0.9
	//b[BuffMagicDamage] -= s[Vitality]

	return b
}
