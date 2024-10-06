package server

import (
	"log"
	"math/rand"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/constants/skill"
	"github.com/rywk/minigoao/pkg/typ"
)

type Cooldowns struct {
	LastAction time.Time
	LastMelee  time.Time
	LastSpells [attack.SpellLen]time.Time
}

func Cast(from, to *Player) (int32, error) {
	sp := attack.SpellProps[from.SelectedSpell]
	if from.dead && sp.Spell != attack.SpellResurrect {
		return 0, attack.ErrorCasterDead
	}
	if to.dead && sp.Spell != attack.SpellResurrect {
		return 0, attack.ErrorTargetDead
	}
	if from.dead && sp.Spell == attack.SpellResurrect {
		sp.Cast(from, to, 0)
		return 0, nil
	}

	now := time.Now()
	if now.Sub(from.cds.LastAction) < from.exp.Stats.ActionCD {
		return 0, attack.ErrorTooFast
	}
	if now.Sub(from.cds.LastSpells[from.SelectedSpell]) < from.exp.Stats.ActionCD {
		return 0, attack.ErrorTooFast
	}

	if from.mp < sp.BaseManaCost {
		return 0, attack.ErrorNoMana
	}
	from.mp = from.mp - sp.BaseManaCost

	itemBuff := int32(from.exp.ItemBuffs[skill.BuffMagicDamage] - to.exp.ItemBuffs[skill.BuffMagicDefense])
	buff := int32(from.exp.SkillBuffs[skill.BuffMagicDamage] - to.exp.SkillBuffs[skill.BuffMagicDefense])
	damage := sp.BaseDamage + buff + itemBuff
	if damage < 0 {
		damage = 0
	}

	err := sp.Cast(from, to, damage)
	if err != nil {
		from.mp = from.mp + sp.BaseManaCost
		return 0, err
	}
	if to.dead {
		from.kills++
		to.deaths--
	}
	if from.SelectedSpell == attack.SpellParalize ||
		from.SelectedSpell == attack.SpellRemoveParalize {
		damage = 0
	}
	from.cds.LastAction = now
	from.cds.LastSpells[from.SelectedSpell] = now
	return damage, nil
}

const BaseMelee = 126

func Melee(from, to *Player) int32 {
	if from == to {
		return -1
	}
	now := time.Now()
	if now.Sub(from.cds.LastAction) < from.exp.Stats.ActionCD {
		log.Printf("melee error, too fast action")
		return -1
	}

	w := from.inv.GetWeapon()
	wp := item.ItemProps[w].WeaponProp
	if wp == nil {
		return -1
	}
	if now.Sub(from.cds.LastMelee) < wp.Cooldown {
		log.Printf("melee error, too fast weapon")
		return -1
	}

	itemBuff := int32(from.exp.ItemBuffs[skill.BuffPhysicalDamage] - to.exp.ItemBuffs[skill.BuffPhysicalDefense])
	buff := int32(from.exp.SkillBuffs[skill.BuffPhysicalDamage] - to.exp.SkillBuffs[skill.BuffPhysicalDefense])
	damage := BaseMelee + (wp.Damage + rand.Int31n(wp.CritRange)) + buff + itemBuff
	if damage < 0 {
		damage = 0
	}

	wp.Cast(from, to, damage)
	if to.dead {
		from.kills++
		to.deaths--
	}
	return damage
}

///

////

///// STUF FOR CLICK SPELL

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
