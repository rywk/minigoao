package server

import (
	"errors"
	"math/rand"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
)

type SpellProp struct {
	Spell      spell.Spell
	BaseDamage int32
	RNGRange   int32
	ManaCost   int32
	Cast       func(from, to *Player, calc int32)
}

var ErrorNoMana = errors.New("no mana")
var ErrorTargetDead = errors.New("target dead")

var spellProps = [spell.None]SpellProp{
	{
		Spell:    spell.Paralize,
		ManaCost: 200,
		Cast: func(from, to *Player, calc int32) {
			to.paralized = true
		},
	},
	{
		Spell:    spell.RemoveParalize,
		ManaCost: 300,
		Cast: func(from, to *Player, calc int32) {
			to.paralized = false
		},
	},
	{
		Spell:      spell.HealWounds,
		ManaCost:   400,
		BaseDamage: 50,
		RNGRange:   10,
		Cast: func(from, to *Player, calc int32) {
			to.hp = to.hp + calc
			if to.hp > to.maxHp {
				to.hp = to.maxHp
			}
		},
	},
	{
		Spell:      spell.Revive,
		ManaCost:   700,
		BaseDamage: 0,
		RNGRange:   0,
		Cast: func(from, to *Player, calc int32) {
			if !to.dead {
				return
			}
			to.dead = false
			to.hp = to.maxHp
		},
	},
	{
		Spell:      spell.ElectricDischarge,
		ManaCost:   500,
		BaseDamage: 60,
		RNGRange:   10,
		Cast: func(from, to *Player, calc int32) {
			to.hp = to.hp - calc
			if to.hp <= 0 {
				to.hp = 0
				to.dead = true
			}
		},
	},
	{
		Spell:      spell.Explode,
		ManaCost:   1000,
		BaseDamage: 170,
		RNGRange:   10,
		Cast: func(from, to *Player, calc int32) {
			to.hp = to.hp - calc
			if to.hp <= 0 {
				to.hp = 0
				to.dead = true
			}
		},
	},
}

func Cast(s *SpellProp, from, to *Player) (int32, error) {
	if to.dead && s.Spell != spell.Revive {
		return 0, ErrorTargetDead
	}
	if from.mp < s.ManaCost {
		return 0, ErrorNoMana
	}
	from.mp = from.mp - s.ManaCost
	calc := s.BaseDamage
	if s.RNGRange != 0 {
		calc = calc + int32(rand.Intn(int(s.RNGRange)))
	}
	s.Cast(from, to, calc)
	return calc, nil
}

func GetSpellProp(s spell.Spell) *SpellProp {
	return &spellProps[s]
}

// Melee
const (
	MeleeBaseDamage = 100
	MeleeRNGRange   = 30
)

func Melee(from, to *Player) int32 {
	calc := MeleeBaseDamage + int32(rand.Intn(int(MeleeRNGRange)))
	to.hp = to.hp - calc
	if to.hp <= 0 {
		to.hp = 0
		to.dead = true
	}
	return calc
}

// Potions
type Item struct {
	Type ItemType
	Use  func(p *Player) uint32
}
type ItemType msgs.Item

func (i ItemType) Item() *Item {
	return &items[i]
}

const (
	ItemManaPotion ItemType = iota
	ItemHealthPotion

	ItemLen
)

var items = [ItemLen]Item{
	{
		Type: ItemManaPotion,
		Use: func(p *Player) uint32 {
			p.mp = p.mp + int32(float32(p.maxMp)*0.05)
			if p.mp > p.maxMp {
				p.mp = p.maxMp
			}
			return uint32(p.mp)
		},
	}, {
		Type: ItemHealthPotion,
		Use: func(p *Player) uint32 {
			p.hp = p.hp + 30
			if p.hp > p.maxHp {
				p.hp = p.maxHp
			}
			return uint32(p.hp)
		},
	},
}

func UseItem(item ItemType, p *Player) uint32 {
	return item.Item().Use(p)
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
