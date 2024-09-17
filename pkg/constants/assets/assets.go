package assets

import (
	"reflect"

	"github.com/rywk/minigoao/pkg/constants/spell"
)

type Image = uint32

const (
	Nothing Image = iota
	// Ground assets
	Grass

	// Default skins
	NakedBody
	Head
	DeadBody
	DeadHead
	// Skins
	ProHat
	DarkArmour
	WarAxe
	SpecialSword
	SilverShield
	TowerShield

	// Effects
	MeleeHit
	SpellInmo
	SpellInmoRm
	SpellApoca
	SpellDesca
	SpellHealWounds
	SpellRevive
	Tiletest

	// Mark
	// Place ever asset that is a solid block after `SolidBlocks`
	SolidBlocks
	// ---
	Shroom
	Tree1
	// ---

	// Can be used as the total of assests
	Len
)

func IsSolid(a Image) bool {
	return a > SolidBlocks && a < Len
}

func AssetName(a Image) string {
	return reflect.TypeOf(a).Name()
}

type Sound = uint32

const (
	Spawn Sound = iota
	Walk1
	Walk2
	Potion
	MeleeAir
	MeleeBlood
	SpellReviveSound
	SpellHealWoundsSound
	SpellInmoSound
	SpellInmoRmSound
	SpellApocaSound
	SpellDescaSound
)

func SoundFromSpell(s spell.Spell) Sound {
	switch s {
	case spell.Explode:
		return SpellApocaSound
	case spell.Paralize:
		return SpellInmoSound
	case spell.RemoveParalize:
		return SpellInmoRmSound
	case spell.ElectricDischarge:
		return SpellDescaSound
	case spell.HealWounds:
		return SpellHealWoundsSound
	case spell.Revive:
		return SpellReviveSound

	}
	return 0
}
