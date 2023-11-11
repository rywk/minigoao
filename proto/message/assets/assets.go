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
	SpellApoca
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
	MeleeAir
	MeleeBlood
	SpellInmoSound
	SpellInmoRmSound
	SpellApocaSound
)

func SoundFromSpell(s spell.Spell) Sound {
	switch s {
	case spell.Apoca:
		return SpellApocaSound
	case spell.Inmo:
		return SpellInmoSound
	case spell.InmoRm:
		return SpellInmoRmSound

	}
	return 0
}
