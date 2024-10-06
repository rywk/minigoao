package item

import (
	"errors"
	"time"

	"github.com/rywk/minigoao/pkg/client/game/assets/img/itemimg"
	"github.com/rywk/minigoao/pkg/constants/skill"
)

type Item uint8

const (
	None Item = iota

	// Potions
	ManaPotion
	HealthPotion

	// Weapons
	WeaponWindSword
	WeaponMightySword
	WeaponFireStaff
	WeaponDarkDagger
	// Shields
	ShieldArcane
	ShieldTower

	// Hats
	HatMage
	HelmetPaladin
	// Armor
	ArmorShadow
	ArmorDark

	// len
	ItemLen
)

const ()

var items = [ItemLen]string{
	"None",
	// Potions
	"ManaPotion",
	"HealthPotion",

	// Weapons
	"WeaponWindSword",
	"WeaponMightySword",
	"WeaponFireStaff",
	"WeaponDarkDagger",
	// Shields
	"ShieldArcane",
	"ShieldTower",
	// Hats
	"HatMage",
	"HelmetPaladin",
	// Armor
	"ArmorShadow",
	"ArmorDark",
}
var itemsPretty = [ItemLen]string{
	"None",
	// Potions
	"Mana Potion",
	"Health Potion",

	// Weapons
	"Wind Sword",
	"Mighty Sword",
	"Fire Staff",
	"Dark Dagger",
	// Shields
	"Arcane Shield",
	"Tower Shield",
	// Hats
	"Mage Hat",
	"Paladin Helmet",
	// Armor
	"Shadow Armor",
	"Dark Armor",
}

type ItemType uint8

const (
	TypeUnknown ItemType = iota
	TypeArmor
	TypeHelmet
	TypeWeapon
	TypeShield
	TypeConsumable
)

var itemsTypes = [ItemLen]ItemType{
	TypeUnknown,

	TypeConsumable,
	TypeConsumable,

	TypeWeapon,
	TypeWeapon,
	TypeWeapon,
	TypeWeapon,
	TypeShield,
	TypeShield,
	TypeHelmet,
	TypeHelmet,
	TypeArmor,
	TypeArmor,
}

func (i Item) String() string {
	return items[i]
}
func (i Item) Name() string {
	return itemsPretty[i]
}
func (i Item) Type() ItemType {
	return itemsTypes[i]
}
func GetAsset(i Item, icon bool) []byte {
	switch i {
	case HealthPotion:
		return itemimg.HealthPotion_png
	case ManaPotion:
		return itemimg.ManaPotion_png
	}
	if icon {
		switch i {
		case WeaponWindSword:
			return itemimg.IconWindSword_png
		case WeaponMightySword:
			return itemimg.IconMightySword_png
		case WeaponFireStaff:
			return itemimg.IconFireStaff_png
		case WeaponDarkDagger:
			return itemimg.IconDarkDargger_png

		case ShieldArcane:
			return itemimg.IconArcaneShield_png
		case ShieldTower:
			return itemimg.IconTowerShield_png

		case HatMage:
			return itemimg.IconMageHat_png
		case HelmetPaladin:
			return itemimg.IconPaladinHelmet_png

		case ArmorDark:
			return itemimg.IconDarkArmor_png
		case ArmorShadow:
			return itemimg.IconShadowArmor_png
		}
	}
	switch i {
	case WeaponWindSword:
		return itemimg.WindSword_png
	case WeaponMightySword:
		return itemimg.MightySword_png
	case WeaponFireStaff:
		return itemimg.FireStaff_png
	case WeaponDarkDagger:
		return itemimg.DarkDargger_png

	case ShieldArcane:
		return itemimg.ArcaneShield_png
	case ShieldTower:
		return itemimg.TowerShield_png

	case HatMage:
		return itemimg.MageHat_png
	case HelmetPaladin:
		return itemimg.PaladinHelmet_png

	case ArmorDark:
		return itemimg.DarkArmor_png
	case ArmorShadow:
		return itemimg.ShadowArmor_png
	}
	return []byte{}
}

type WeaponProp struct {
	Cooldown  time.Duration
	Damage    int32
	CritRange int32
	Cast      func(from, to Player, calc int32) error
}

type ItemProp struct {
	Type       Item
	Buffs      skill.Buffs
	WeaponProp *WeaponProp
	Use        func(p Player) uint32
}

func (ip ItemProp) IsWeapon() bool {
	return ip.WeaponProp != nil
}

func (ip ItemProp) IsPotion() bool {
	return ip.Type == ManaPotion || ip.Type == HealthPotion
}

type Player interface {
	Heal(int32)
	TakeDamage(int32)
	Dead() bool
	Revive()
	SetParalized(bool)

	AddMana(int32) int32
	AddHp(int32) int32
	MultMaxMana(float64) int32
}

const HandDamage int32 = -50
const HandCritDamage int32 = 15
const HandCooldown = time.Second

var ErrorSelfCast = errors.New("cant self cast")
var ItemProps = [ItemLen]ItemProp{
	{
		Type: None,
		WeaponProp: &WeaponProp{
			Damage:    HandDamage,
			CritRange: HandCritDamage,
			Cooldown:  HandCooldown,
			Cast: func(from, to Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
		Use: func(p Player) uint32 { return 0 }},
	{
		Type: ManaPotion,
		Use: func(p Player) uint32 {
			n := p.MultMaxMana(0.05)
			return uint32(p.AddMana(n))
		},
	}, {
		Type: HealthPotion,
		Use: func(p Player) uint32 {
			return uint32(p.AddHp(27))
		},
	},
	{
		Type: WeaponWindSword,
		Buffs: skill.Buffs{}.AddValue(skill.BuffMagicDamage, 2).
			AddValue(skill.BuffPhysicalDamage, 1),
		Use: func(p Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Cooldown:  time.Millisecond * 900,
			Damage:    1,
			CritRange: 2,
			Cast: func(from, to Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},
	{
		Type:  WeaponMightySword,
		Buffs: skill.Buffs{}.AddValue(skill.BuffPhysicalDamage, 3),
		Use: func(p Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Cooldown:  time.Millisecond * 1000,
			Damage:    6,
			CritRange: 2,
			Cast: func(from, to Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},
	{
		Type:  WeaponFireStaff,
		Buffs: skill.Buffs{}.AddValue(skill.BuffMagicDamage, 3),
		Use:   func(p Player) uint32 { return 0 },
		WeaponProp: &WeaponProp{
			Cooldown:  time.Millisecond * 1000,
			Damage:    -20,
			CritRange: 6,
			Cast: func(from, to Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},
	{
		Type: WeaponDarkDagger,
		Buffs: skill.Buffs{}.AddValue(skill.BuffMagicDamage, 1).
			AddValue(skill.BuffPhysicalDamage, 2),
		Use: func(p Player) uint32 {
			// use means equip or unequip in the case of wearable items
			return 0
		},
		WeaponProp: &WeaponProp{
			Cooldown:  time.Millisecond * 1000,
			Damage:    -15,
			CritRange: 30,
			Cast: func(from, to Player, calc int32) error {
				if from == to {
					return ErrorSelfCast
				}
				to.TakeDamage(calc)
				return nil
			},
		},
	},

	{
		Type:  ShieldArcane,
		Buffs: skill.Buffs{}.AddValue(skill.BuffMagicDefense, 1),
		Use: func(p Player) uint32 {
			return 0
		},
	},
	{
		Type:  ShieldTower,
		Buffs: skill.Buffs{}.AddValue(skill.BuffPhysicalDefense, 1),
		Use: func(p Player) uint32 {
			return 0
		},
	},
	{
		Type:  HatMage,
		Buffs: skill.Buffs{}.AddValue(skill.BuffPhysicalDefense, 1),
		Use: func(p Player) uint32 {
			return 0
		},
	},
	{
		Type:  HelmetPaladin,
		Buffs: skill.Buffs{}.AddValue(skill.BuffMagicDefense, 1),
		Use: func(p Player) uint32 {
			return 0
		},
	},
	{
		Type:  ArmorShadow,
		Buffs: skill.Buffs{}.AddValue(skill.BuffMagicDefense, 1),
		Use: func(p Player) uint32 {
			return 0
		},
	},
	{
		Type:  ArmorDark,
		Buffs: skill.Buffs{}.AddValue(skill.BuffPhysicalDefense, 1),
		Use: func(p Player) uint32 {
			return 0
		},
	},
}

func UseItem(item Item, p Player) uint32 {
	return ItemProps[item].Use(p)
}
