package player

import (
	"math/rand"
	"time"
)

type Spell interface {
	ManaCost() int
	Cast(from, to *Player) (bool, int)
}

type BaseSpell struct {
	ManaCost int
}

var _ Spell = (*Inmo)(nil)

func Mana(s Spell, from, to *Player) bool {
	if from.MP < s.ManaCost() {
		return false
	}
	from.MP = from.MP - s.ManaCost()
	return true
}

type InmoRm struct {
	BaseSpell
}

func NewInmoRm(manaCost int) *InmoRm {
	return &InmoRm{
		BaseSpell: BaseSpell{ManaCost: manaCost},
	}
}

func (s *InmoRm) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *InmoRm) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	if to.Inmobilized {
		to.Inmobilized = false
		return true, 0
	}
	return false, 0
}

type Inmo struct {
	BaseSpell
	Duration time.Duration
}

func NewInmo(manaCost int, duration time.Duration) *Inmo {
	return &Inmo{
		BaseSpell: BaseSpell{ManaCost: manaCost},
		Duration:  duration,
	}
}

func (s *Inmo) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *Inmo) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	if to.Inmobilized {
		return false, 0
	}
	to.Inmobilized = true
	// go func() {
	// 	<-time.NewTicker(s.Duration).C
	// }()
	return true, 0
}

type Apoca struct {
	BaseSpell
	BaseDamage int
	CritRange  int
}

func NewApoca(manaCost, baseDamage, critRange int) *Apoca {
	return &Apoca{
		BaseSpell:  BaseSpell{ManaCost: manaCost},
		BaseDamage: baseDamage,
		CritRange:  critRange,
	}
}

func (s *Apoca) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *Apoca) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	dmg := s.BaseDamage
	if s.CritRange > 0 {
		dmg += rand.Intn(s.CritRange)
	}
	to.HP = to.HP - dmg
	if to.HP < 0 {
		to.Dead = true
		to.HP = 0
	}
	return true, dmg
}
