package player

import (
	"math/rand"
	"time"
)

type Spell interface {
	ManaCost() int
	CanSelfCast() bool
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

type Revive struct {
	BaseSpell
}

func NewRevive(manaCost int) *Revive {
	return &Revive{
		BaseSpell: BaseSpell{ManaCost: manaCost},
	}
}

func (s *Revive) CanSelfCast() bool { return false }

func (s *Revive) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *Revive) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	if to.Dead {
		to.RevivePlayer()
		return true, 0
	}
	return false, 0
}

type HealWounds struct {
	BaseSpell
	BaseHeal  int
	CritRange int
}

func NewHealWounds(manaCost, baseHeal, critRange int) *HealWounds {
	return &HealWounds{
		BaseSpell: BaseSpell{ManaCost: manaCost},
		BaseHeal:  baseHeal,
		CritRange: critRange,
	}
}

func (s *HealWounds) CanSelfCast() bool { return true }

func (s *HealWounds) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *HealWounds) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	if !to.Dead {
		val := s.BaseHeal + rand.Intn(s.CritRange)
		to.HealPlayer(val)
		return true, val
	}
	return false, 0
}

type InmoRm struct {
	BaseSpell
}

func NewInmoRm(manaCost int) *InmoRm {
	return &InmoRm{
		BaseSpell: BaseSpell{ManaCost: manaCost},
	}
}

func (s *InmoRm) CanSelfCast() bool { return true }

func (s *InmoRm) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *InmoRm) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	if to.IsInmobilized() {
		to.ChangeInmobilized(false)
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

func (s *Inmo) CanSelfCast() bool { return false }

func (s *Inmo) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *Inmo) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	if to.IsInmobilized() {
		return false, 0
	}
	to.ChangeInmobilized(true)
	// go func() {
	// 	<-time.NewTicker(s.Duration).C
	// }()
	return true, 0
}

type DamageSpell struct {
	BaseSpell
	BaseDamage int
	CritRange  int
}

func NewDamageSpell(manaCost, baseDamage, critRange int) *DamageSpell {
	return &DamageSpell{
		BaseSpell:  BaseSpell{ManaCost: manaCost},
		BaseDamage: baseDamage,
		CritRange:  critRange,
	}
}

func (s *DamageSpell) CanSelfCast() bool { return false }

func (s *DamageSpell) ManaCost() int { return s.BaseSpell.ManaCost }

func (s *DamageSpell) Cast(from, to *Player) (bool, int) {
	if !Mana(s, from, to) {
		return false, 0
	}
	dmg := s.BaseDamage
	if s.CritRange > 0 {
		dmg += rand.Intn(s.CritRange)
	}
	to.DamagePlayer(dmg)
	return true, dmg
}
