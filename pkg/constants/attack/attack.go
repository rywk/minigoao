package attack

import (
	"errors"
	"time"
)

type Spell uint8

const (
	SpellNone Spell = iota
	SpellParalize
	SpellRemoveParalize
	SpellHealWounds
	SpellResurrect
	SpellElectricDischarge
	SpellExplode
	SpellLen
)

var spells = [SpellLen]string{
	"SpellNone",
	"SpellParalize",
	"SpellRemoveParalize",
	"SpellHealWounds",
	"SpellResurrect",
	"SpellElectricDischarge",
	"SpellExplode",
}

func (s Spell) String() string {
	return spells[s]
}

type SpellProp struct {
	Spell        Spell
	BaseCooldown time.Duration
	BaseDamage   int32
	BaseManaCost int32
	Cast         func(from, to Player, calc int32) error
}

type Player interface {
	Heal(int32)
	TakeDamage(int32)
	Dead() bool
	Revive()
	SetParalized(bool)
	IsParalized() bool
}

var ErrorNoMana = errors.New("no mana")
var ErrorTargetDead = errors.New("target dead")
var ErrorTargetAlive = errors.New("target alive")
var ErrorCasterDead = errors.New("caster dead")
var ErrorSelfCast = errors.New("cant self cast")
var ErrorTooFast = errors.New("too fast")
var ErrorAlreadyHasEffect = errors.New("already has effect")
var ErrorDoesNotHaveEffect = errors.New("does not have effect")

var SpellProps = [SpellLen]SpellProp{
	{Spell: SpellNone},
	{
		Spell:        SpellParalize,
		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 300,
		Cast: func(from, to Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			if to.IsParalized() {
				return ErrorAlreadyHasEffect
			}
			to.SetParalized(true)
			return nil
		},
	},
	{
		Spell:        SpellRemoveParalize,
		BaseCooldown: time.Millisecond * 940,
		BaseManaCost: 340,
		Cast: func(from, to Player, calc int32) error {
			if !to.IsParalized() {
				return ErrorDoesNotHaveEffect
			}
			to.SetParalized(false)
			return nil
		},
	},
	{
		Spell:        SpellHealWounds,
		BaseCooldown: time.Second,
		BaseManaCost: 320,
		BaseDamage:   62,
		Cast: func(_, to Player, calc int32) error {
			to.Heal(calc)
			return nil
		},
	},
	{
		Spell:        SpellResurrect,
		BaseCooldown: time.Second,
		BaseManaCost: 1200,
		BaseDamage:   0,
		Cast: func(from, to Player, calc int32) error {
			if !to.Dead() {
				return ErrorTargetAlive
			}
			to.Revive()
			return nil
		},
	},
	{
		Spell:        SpellElectricDischarge,
		BaseCooldown: time.Millisecond * 900,
		BaseManaCost: 450,
		BaseDamage:   92,
		Cast: func(from, to Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
	{
		Spell:        SpellExplode,
		BaseCooldown: time.Second,
		BaseManaCost: 999,
		BaseDamage:   149,
		Cast: func(from, to Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
}
