package server

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/constants/mapdef"
	"github.com/rywk/minigoao/pkg/constants/skill"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/typ"
)

type Cooldowns struct {
	LastAction time.Time
	LastMelee  time.Time
	LastSpells [attack.SpellLen]time.Time
}

func Cast(from, to *Player) (int32, error) {
	now := time.Now()
	if now.Sub(from.cds.LastMelee) < from.exp.Stats.SwitchCD {
		return 0, attack.ErrorTooFast
	}
	if now.Sub(from.cds.LastAction) < from.exp.Stats.ActionCD {
		return 0, attack.ErrorTooFast
	}

	if from.SelectedSpell != attack.SpellRemoveParalize &&
		from.SelectedSpell != attack.SpellHealWounds &&
		from.SelectedSpell != attack.SpellResurrect {
		for _, tid := range from.team {
			if tid == to.id {
				log.Printf("spell attacking teamate")
				return 0, errors.New("cannot attack teamate")
			}
		}
	}

	sp := attack.SpellProps[from.SelectedSpell]
	if from.dead {
		return 0, attack.ErrorCasterDead
	}
	if to.dead && sp.Spell != attack.SpellResurrect {
		return 0, attack.ErrorTargetDead
	}

	if now.Sub(from.cds.LastSpells[from.SelectedSpell]) < sp.BaseCooldown {
		return 0, attack.ErrorTooFast
	}

	manaCost := sp.RealManaCost(from.exp.Stats.MaxMP)
	if from.mp < manaCost {
		return 0, attack.ErrorNoMana
	}
	from.mp = from.mp - manaCost

	itemBuff := int32(from.exp.ItemBuffs[skill.BuffMagicDamage] - to.exp.ItemBuffs[skill.BuffMagicDefense])
	skillBuff := int32(from.exp.SkillBuffs[skill.BuffMagicDamage] - to.exp.SkillBuffs[skill.BuffMagicDefense])
	if sp.Spell == attack.SpellResurrect ||
		sp.Spell == attack.SpellHealWounds {
		itemBuff = int32(from.exp.ItemBuffs[skill.BuffMagicDamage])
		skillBuff = int32(from.exp.SkillBuffs[skill.BuffMagicDamage])
	}

	damage := from.exp.Stats.BaseSpell + sp.BaseDamage + skillBuff + itemBuff
	if damage < 0 {
		damage = 0
	}

	if sp.Cast == nil {
		return 0, nil
	}

	err := sp.Cast(from, to, damage)
	if err != nil {
		from.mp = from.mp + manaCost
		return 0, err
	}
	if to.dead {
		from.kills++
		to.deaths++
	}
	if from.SelectedSpell == attack.SpellParalize ||
		from.SelectedSpell == attack.SpellRemoveParalize {
		damage = 0
	}
	from.cds.LastAction = now
	from.cds.LastSpells[from.SelectedSpell] = now
	return damage, nil
}

func Melee(from, to *Player) int32 {
	if from == to {
		return -1
	}
	now := time.Now()
	if now.Sub(from.cds.LastAction) < from.exp.Stats.SwitchCD {
		log.Printf("melee error, too fast action")
		return -1
	}
	for _, tid := range from.team {
		if tid == to.id {
			log.Printf("melee attacking teamate")

			return -1
		}
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
	damage := from.exp.Stats.BaseMelee + (wp.Damage + rand.Int31n(wp.CritRange)) + buff + itemBuff
	if damage < 0 {
		damage = 0
	}

	wp.Cast(from, to, damage)
	if to.dead {
		from.kills++
		to.deaths++
	}
	from.cds.LastMelee = now
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

func (g *Game) CheckSpellTargets(space *grid.Grid, px typ.P) uint16 {
	tilePos := typ.P{
		X: int32(px.X) / constants.TileSize,
		Y: int32(px.Y) / constants.TileSize,
	}

	offR := space.Rect
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
		downTargetId := space.GetSlot(mapdef.Players.Int(), downTilePos)
		if downTargetId != 0 {
			if px.In(g.players[downTargetId].CalcHitbox()) {
				return downTargetId
			}
		}
	}

	leftDownTilePos := downTilePos
	leftDownTilePos.X--
	if leftDownTilePos.In(space.Rect) {
		leftDownTargetId := space.GetSlot(mapdef.Players.Int(), leftDownTilePos)
		if leftDownTargetId != 0 {
			if px.In(g.players[leftDownTargetId].CalcHitbox()) {
				return leftDownTargetId
			}
		}
	}

	rightDownTilePos := downTilePos
	rightDownTilePos.X++
	if rightDownTilePos.In(space.Rect) {
		rightDownTargetId := space.GetSlot(mapdef.Players.Int(), rightDownTilePos)
		if rightDownTargetId != 0 {
			if px.In(g.players[rightDownTargetId].CalcHitbox()) {
				return rightDownTargetId
			}
		}
	}

	downDownTilePos := downTilePos
	downDownTilePos.Y++
	if downDownTilePos.In(space.Rect) {
		downDownTargetId := space.GetSlot(mapdef.Players.Int(), downDownTilePos)
		if downDownTargetId != 0 {
			if px.In(g.players[downDownTargetId].CalcHitbox()) {
				return downDownTargetId
			}
		}
	}

	if tilePos.In(space.Rect) {
		targetId := space.GetSlot(mapdef.Players.Int(), tilePos)
		if targetId != 0 {
			if px.In(g.players[targetId].CalcHitbox()) {
				return targetId
			}
		}
	}
	upTilePos := tilePos
	upTilePos.Y--
	if upTilePos.In(space.Rect) {
		upTargetId := space.GetSlot(mapdef.Players.Int(), upTilePos)
		if upTargetId != 0 {
			if px.In(g.players[upTargetId].CalcHitbox()) {
				return upTargetId
			}
		}
	}

	leftTilePos := tilePos
	leftTilePos.X--
	if leftTilePos.In(space.Rect) {

		leftTargetId := space.GetSlot(mapdef.Players.Int(), leftTilePos)
		if leftTargetId != 0 {
			if px.In(g.players[leftTargetId].CalcHitbox()) {
				return leftTargetId
			}
		}
	}

	rightTilePos := tilePos
	rightTilePos.X++
	if rightTilePos.In(space.Rect) {
		rightTargetId := space.GetSlot(mapdef.Players.Int(), rightTilePos)
		if rightTargetId != 0 {
			if px.In(g.players[rightTargetId].CalcHitbox()) {
				return rightTargetId
			}
		}
	}
	return 0
}
