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
	"github.com/rywk/minigoao/pkg/typ"
)

type SpellProp struct {
	Spell        attack.Spell
	MagicAff     MagicAffinityType
	AeraSpell    bool
	BaseCooldown time.Duration
	BaseDamage   int32
	BaseManaCost int32

	Cast func(from, to *Player, calc int32) error

	ExpSpell ExpSpell
}

type ExpSpell struct {
	CD       Cooldown
	Damage   int32
	ManaCost int32
}

var ErrorNoMana = errors.New("no mana")
var ErrorTargetDead = errors.New("target dead")
var ErrorTargetAlive = errors.New("target alive")
var ErrorCasterDead = errors.New("caster dead")
var ErrorSelfCast = errors.New("cant self cast")
var ErrorTooFast = errors.New("too fast")

const BaseHp float32 = 189
const BaseMp float32 = 840

var spellProps = [attack.SpellLen]SpellProp{
	{Spell: attack.SpellNone},
	{
		Spell:        attack.SpellParalize,
		MagicAff:     MagicAffinityTypeNone,
		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 400,
		Cast: func(from, to *Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.paralized = true
			return nil
		},
	},
	{
		Spell:    attack.SpellRemoveParalize,
		MagicAff: MagicAffinityTypeNone,

		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 550,
		Cast: func(from, to *Player, calc int32) error {
			to.paralized = false
			return nil
		},
	},
	{
		Spell:    attack.SpellHealWounds,
		MagicAff: MagicAffinityTypeCleric,

		BaseCooldown: time.Millisecond * 1000,
		BaseManaCost: 600,
		BaseDamage:   54,
		Cast: func(_, to *Player, calc int32) error {
			to.Heal(calc)
			return nil
		},
	},
	{
		Spell:    attack.SpellResurrect,
		MagicAff: MagicAffinityTypeCleric,

		BaseCooldown: time.Millisecond * 5000,
		BaseManaCost: 1100,
		BaseDamage:   0,
		Cast: func(from, to *Player, calc int32) error {
			if !to.dead {
				return ErrorTargetAlive
			}
			to.dead = false
			to.hp = to.exp.MaxHp
			return nil
		},
	},
	{
		Spell:        attack.SpellElectricDischarge,
		MagicAff:     MagicAffinityTypeElectric,
		BaseCooldown: time.Millisecond * 900,
		BaseManaCost: 550,
		BaseDamage:   71,
		Cast: func(from, to *Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
	{
		Spell:        attack.SpellExplode,
		MagicAff:     MagicAffinityTypeFire,
		BaseCooldown: time.Millisecond * 1100,
		BaseManaCost: 1250,
		BaseDamage:   138,
		Cast: func(from, to *Player, calc int32) error {
			if from == to {
				return ErrorSelfCast
			}
			to.TakeDamage(calc)
			return nil
		},
	},
}

func Cast(from, to *Player) (int32, error) {
	ns := &from.exp.spells[from.exp.SelectedSpell]
	if from.dead && ns.Spell != attack.SpellResurrect {
		return 0, ErrorCasterDead
	}
	if to.dead && ns.Spell != attack.SpellResurrect {
		return 0, ErrorTargetDead
	}
	if from.dead && ns.Spell == attack.SpellResurrect {
		ns.Cast(from, to, 0)
		return 0, nil
	}
	confirmAction, can := from.exp.ActionCooldown.Ask()
	if !can {
		return 0, ErrorTooFast
	}
	confirm, can := ns.ExpSpell.CD.Ask()
	if !can {
		return 0, ErrorTooFast
	}
	if from.mp < ns.ExpSpell.ManaCost {
		return 0, ErrorNoMana
	}
	from.mp = from.mp - ns.ExpSpell.ManaCost

	err := ns.Cast(from, to, ns.ExpSpell.Damage)
	if err != nil {
		from.mp = from.mp + ns.ExpSpell.ManaCost
		return 0, err
	}
	confirm()
	confirmAction()
	return ns.ExpSpell.Damage, nil
}

func GetSpellProp(s attack.Spell) *SpellProp {
	return &spellProps[s]
}

type ExpWeapon struct {
	CD        Cooldown
	Damage    int32
	CritRange int32
}

func Melee(from, to *Player) int32 {
	confirmAction, can := from.exp.ActionCooldown.Ask()
	if !can {
		log.Printf("melee error, too fast action")
		return -1
	}
	w := from.inv.GetWeapon()
	wp := from.exp.items[w].WeaponProp
	if wp == nil {
		calc := int32(from.exp.BaseMeleeDamage) + int32(rand.Intn(int(from.exp.BaseMeleeCritRange)))
		if from == to {
			return -1
		}
		to.TakeDamage(calc)
		return calc
	}
	confirm, can := wp.ExpWeapon.CD.Ask()
	log.Printf("cd : %v\ncan %v", wp.ExpWeapon.CD.CD, can)
	if !can {
		log.Printf("melee error, too fast melee by %v", time.Since(wp.ExpWeapon.CD.Last)-wp.ExpWeapon.CD.CD)
		return -1
	}
	confirm()
	confirmAction()
	log.Printf("base: %v, crit range: %v", wp.ExpWeapon.Damage, wp.ExpWeapon.CritRange)
	calc := int32(wp.ExpWeapon.Damage) + int32(rand.Intn(int(wp.ExpWeapon.CritRange)))
	wp.Cast(from, to, calc)
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

func (c *Cooldown) Ask() (func(), bool) {
	now := time.Now()
	return func() { c.Last = now }, !(now.Sub(c.Last) < c.CD-time.Millisecond*4)
}

func GetItemProp(w item.Item) ItemProp {
	return items[w]
}

func GetWeaponProp(w item.Item) WeaponProp {
	return *items[w].WeaponProp
}

type WeaponProp struct {
	Weapon        item.Item
	MeleeAff      MeleeAffinityType
	MagicAff      MagicAffinityType
	MeleeAffN     int32
	MagicAffN     int32
	BaseCooldown  time.Duration
	BaseDamage    int32
	BaseCritRange int32
	ExpWeapon     ExpWeapon
	Cast          func(from, to *Player, calc int32) error
}

type ShieldProp struct {
	PhysicalDef int32
	MagicDef    int32
}
type HeadProp struct {
	PhysicalDef int32
	MagicDef    int32
}
type ArmorProp struct {
	PhysicalDef int32
	MagicDef    int32
}
type ItemProp struct {
	Type       item.Item
	WeaponProp *WeaponProp
	ShieldProp *ShieldProp
	HeadProp   *HeadProp
	ArmorProp  *ArmorProp
	Use        func(p *Player) uint32
}

func (ip ItemProp) IsWeapon() bool {
	return ip.WeaponProp != nil
}

func (ip ItemProp) IsPotion() bool {
	return ip.Type == item.ManaPotion || ip.Type == item.HealthPotion
}

var items = [item.ItemLen]ItemProp{
	{Type: item.None, Use: func(p *Player) uint32 { return 0 }},
	{
		Type: item.ManaPotion,
		Use: func(p *Player) uint32 {
			p.mp = p.mp + int32(float32(p.exp.MaxMp)*0.05)
			if p.mp > p.exp.MaxMp {
				p.mp = p.exp.MaxMp
			}
			return uint32(p.mp)
		},
	}, {
		Type: item.HealthPotion,
		Use: func(p *Player) uint32 {
			p.hp = p.hp + 30
			if p.hp > p.exp.MaxHp {
				p.hp = p.exp.MaxHp
			}
			return uint32(p.hp)
		},
	},
	{
		Type: item.WeaponWindSword,
		Use: func(p *Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Weapon:        item.WeaponWindSword,
			MagicAff:      MagicAffinityTypeCleric,
			MagicAffN:     3,
			MeleeAff:      MeleeAffinityTypeMartialArt,
			MeleeAffN:     2,
			BaseCooldown:  time.Millisecond * 900,
			BaseDamage:    74,
			BaseCritRange: 2,
			Cast: func(from, to *Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},
	{
		Type: item.WeaponMightySword,
		Use: func(p *Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Weapon:        item.WeaponMightySword,
			MagicAff:      MagicAffinityTypeElectric,
			MagicAffN:     2,
			MeleeAff:      MeleeAffinityTypeWarrior,
			MeleeAffN:     3,
			BaseCooldown:  time.Millisecond * 1000,
			BaseDamage:    98,
			BaseCritRange: 8,
			Cast: func(from, to *Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},
	{
		Type: item.WeaponFireStaff,
		Use: func(p *Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Weapon:        item.WeaponFireStaff,
			MagicAff:      MagicAffinityTypeFire,
			MagicAffN:     5,
			MeleeAff:      MeleeAffinityTypeNone,
			BaseCooldown:  time.Millisecond * 1000,
			BaseDamage:    20,
			BaseCritRange: 20,
			Cast: func(from, to *Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},
	{
		Type: item.WeaponDarkDagger,
		Use: func(p *Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Weapon:        item.WeaponFireStaff,
			MagicAff:      MagicAffinityTypeElectric,
			MagicAffN:     3,
			MeleeAff:      MeleeAffinityTypeAssasin,
			MeleeAffN:     2,
			BaseCooldown:  time.Millisecond * 1000,
			BaseDamage:    66,
			BaseCritRange: 60,
			Cast: func(from, to *Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},

	{
		Type:       item.ShieldArcane,
		ShieldProp: &ShieldProp{},
		Use: func(p *Player) uint32 {
			return 0
		},
	},
	{
		Type:       item.ShieldTower,
		ShieldProp: &ShieldProp{},
		Use: func(p *Player) uint32 {
			return 0
		},
	},
	{
		Type:     item.HatMage,
		HeadProp: &HeadProp{},
		Use: func(p *Player) uint32 {
			return 0
		},
	},
	{
		Type:     item.HelmetPaladin,
		HeadProp: &HeadProp{},
		Use: func(p *Player) uint32 {
			return 0
		},
	},
	{
		Type:      item.ArmorShadow,
		ArmorProp: &ArmorProp{},
		Use: func(p *Player) uint32 {
			return 0
		},
	},
	{
		Type:      item.ArmorDark,
		ArmorProp: &ArmorProp{},
		Use: func(p *Player) uint32 {
			return 0
		},
	},
}

func UseItem(item item.Item, p *Player) uint32 {
	return items[item].Use(p)
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
