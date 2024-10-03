package item

import "github.com/rywk/minigoao/pkg/client/game/assets/img/itemimg"

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
