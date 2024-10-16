package server

import (
	"errors"
	"math/rand"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/typ"
)

type SpellProp struct {
	Spell      spell.Spell
	BaseDamage int32
	RNGRange   int32
	ManaCost   int32
	Cast       func(from, to *Player, calc int32) error
}

var ErrorNoMana = errors.New("no mana")
var ErrorTargetDead = errors.New("target dead")
var ErrorTargetAlive = errors.New("target alive")
var ErrorCasterDead = errors.New("caster dead")
var ErrorSelfCast = errors.New("cant self cast")

var spellProps = [spell.Len]SpellProp{
	{Spell: spell.None},
	{
		Spell:    spell.Paralize,
		ManaCost: 200,
		Cast: func(from, to *Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.paralized = true
			return nil
		},
	},
	{
		Spell:    spell.RemoveParalize,
		ManaCost: 450,
		Cast: func(from, to *Player, calc int32) error {
			to.paralized = false
			return nil
		},
	},
	{
		Spell:      spell.HealWounds,
		ManaCost:   400,
		BaseDamage: 50,
		RNGRange:   10,
		Cast: func(_, to *Player, calc int32) error {
			to.Heal(calc)
			return nil
		},
	},
	{
		Spell:      spell.Resurrect,
		ManaCost:   1100,
		BaseDamage: 0,
		RNGRange:   0,
		Cast: func(from, to *Player, calc int32) error {
			if !to.dead {
				return ErrorTargetAlive
			}
			to.dead = false
			to.hp = to.maxHp
			return nil
		},
	},
	{
		Spell:      spell.ElectricDischarge,
		ManaCost:   550,
		BaseDamage: 81,
		RNGRange:   6,
		Cast: func(from, to *Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
	{
		Spell:      spell.Explode,
		ManaCost:   1100,
		BaseDamage: 177,
		RNGRange:   10,
		Cast: func(from, to *Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
}

func Cast(s *SpellProp, from, to *Player) (int32, error) {
	if from.dead && s.Spell != spell.Resurrect {
		return 0, ErrorCasterDead
	}
	if to.dead && s.Spell != spell.Resurrect {
		return 0, ErrorTargetDead
	}
	if from.dead && s.Spell == spell.Resurrect {
		s.Cast(from, to, 0)
		return 0, nil
	}
	if from.mp < s.ManaCost {
		return 0, ErrorNoMana
	}
	from.mp = from.mp - s.ManaCost
	calc := s.BaseDamage
	if s.RNGRange != 0 {
		calc = calc + int32(rand.Intn(int(s.RNGRange)))
	}
	err := s.Cast(from, to, calc)
	if err != nil {
		from.mp = from.mp + s.ManaCost
		return 0, err
	}
	return calc, nil
}

func GetSpellProp(s spell.Spell) *SpellProp {
	return &spellProps[s]
}

// Melee
const (
	MeleeBaseDamage = 109
	MeleeRNGRange   = 11
)

func Melee(from, to *Player) int32 {
	calc := MeleeBaseDamage + int32(rand.Intn(int(MeleeRNGRange)))
	to.TakeDamage(calc)
	return calc
}

type Cooldown struct {
	CD   time.Duration
	Last time.Time
}

func (c *Cooldown) Try() bool {
	now := time.Now()
	if now.Sub(c.Last) < c.CD {
		return false
	}
	c.Last = now
	return true
}

var playerHitbox = typ.Rect{Min: typ.P{X: -16, Y: -48}, Max: typ.P{X: 16, Y: 16}}

func (p *Player) CalcHitbox() typ.Rect {
	tilePxCenter := typ.P{
		X: (p.pos.X * constants.TileSize) + (constants.TileSize / 2),
		Y: (p.pos.Y * constants.TileSize) + (constants.TileSize / 2),
	}
	sinceMoved := time.Since(p.lastMove)
	if sinceMoved >= p.speedXTile {
		return playerHitbox.OnPoint(tilePxCenter)
	}
	off := constants.TileSize - int32((sinceMoved/AverageGameFrame))*p.speedPxXFrame
	switch p.dir {
	case direction.Back:
		tilePxCenter.Y = tilePxCenter.Y + off
	case direction.Front:
		tilePxCenter.Y = tilePxCenter.Y - off
	case direction.Left:
		tilePxCenter.X = tilePxCenter.X + off
	case direction.Right:
		tilePxCenter.X = tilePxCenter.X - off
	}
	return playerHitbox.OnPoint(tilePxCenter)
}

func (g *Game) CheckSpellTargets(px typ.P) uint16 {
	tilePos := typ.P{
		X: int32(px.X) / constants.TileSize,
		Y: int32(px.Y) / constants.TileSize,
	}

	offR := g.space.Rect
	offR.Min.Y--
	if tilePos.Out(offR) {
		return 0
	}
	// get whats in all the slots a player could have gone into from here
	// a player looks almost 2 tiles tall so we always looks for the click in the upper part too
	// this means the tile under it has the player
	// something like this
	// [ ] [ ] [ ] [ ] [ ]
	// [ ] [ ] [x] [ ] [ ]
	// [ ] [x] [o] [x] [ ]
	// [ ] [x] [x] [x] [ ]
	// [ ] [ ] [x] [ ] [ ]
	// [ ] [ ] [ ] [ ] [ ]

	// make sure we check first for the ids that are over the other hitboxs
	// the ones over have priority
	downTilePos := tilePos
	downTilePos.Y++
	if downTilePos.In(offR) {
		downTargetId := g.space.GetSlot(0, downTilePos)
		if downTargetId != 0 {
			if px.In(g.players[downTargetId].CalcHitbox()) {
				return downTargetId
			}
		}
	}

	leftDownTilePos := downTilePos
	leftDownTilePos.X--
	if leftDownTilePos.In(g.space.Rect) {
		leftDownTargetId := g.space.GetSlot(0, leftDownTilePos)
		if leftDownTargetId != 0 {
			if px.In(g.players[leftDownTargetId].CalcHitbox()) {
				return leftDownTargetId
			}
		}
	}

	rightDownTilePos := downTilePos
	rightDownTilePos.X++
	if rightDownTilePos.In(g.space.Rect) {
		rightDownTargetId := g.space.GetSlot(0, rightDownTilePos)
		if rightDownTargetId != 0 {
			if px.In(g.players[rightDownTargetId].CalcHitbox()) {
				return rightDownTargetId
			}
		}
	}

	downDownTilePos := downTilePos
	downDownTilePos.Y++
	if downDownTilePos.In(g.space.Rect) {
		downDownTargetId := g.space.GetSlot(0, downDownTilePos)
		if downDownTargetId != 0 {
			if px.In(g.players[downDownTargetId].CalcHitbox()) {
				return downDownTargetId
			}
		}
	}

	if tilePos.In(g.space.Rect) {
		targetId := g.space.GetSlot(0, tilePos)
		if targetId != 0 {
			if px.In(g.players[targetId].CalcHitbox()) {
				return targetId
			}
		}
	}
	upTilePos := tilePos
	upTilePos.Y--
	if upTilePos.In(g.space.Rect) {
		upTargetId := g.space.GetSlot(0, upTilePos)
		if upTargetId != 0 {
			if px.In(g.players[upTargetId].CalcHitbox()) {
				return upTargetId
			}
		}
	}

	leftTilePos := tilePos
	leftTilePos.X--
	if leftTilePos.In(g.space.Rect) {

		leftTargetId := g.space.GetSlot(0, leftTilePos)
		if leftTargetId != 0 {
			if px.In(g.players[leftTargetId].CalcHitbox()) {
				return leftTargetId
			}
		}
	}

	rightTilePos := tilePos
	rightTilePos.X++
	if rightTilePos.In(g.space.Rect) {
		rightTargetId := g.space.GetSlot(0, rightTilePos)
		if rightTargetId != 0 {
			if px.In(g.players[rightTargetId].CalcHitbox()) {
				return rightTargetId
			}
		}
	}
	return 0
}
