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
}

var ErrorNoMana = errors.New("no mana")
var ErrorTargetDead = errors.New("target dead")
var ErrorTargetAlive = errors.New("target alive")
var ErrorCasterDead = errors.New("caster dead")
var ErrorSelfCast = errors.New("cant self cast")
var ErrorTooFast = errors.New("too fast")

var SpellProps = [SpellLen]SpellProp{
	{Spell: SpellNone},
	{
		Spell:        SpellParalize,
		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 420,
		Cast: func(from, to Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.SetParalized(true)
			return nil
		},
	},
	{
		Spell:        SpellRemoveParalize,
		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 480,
		Cast: func(from, to Player, calc int32) error {
			to.SetParalized(false)
			return nil
		},
	},
	{
		Spell:        SpellHealWounds,
		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 600,
		BaseDamage:   54,
		Cast: func(_, to Player, calc int32) error {
			to.Heal(calc)
			return nil
		},
	},
	{
		Spell:        SpellResurrect,
		BaseCooldown: time.Millisecond * 5000,
		BaseManaCost: 1100,
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
		BaseManaCost: 420,
		BaseDamage:   71,
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
		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 999,
		BaseDamage:   174,
		Cast: func(from, to Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
}
