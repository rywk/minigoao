package assets

import (
	"reflect"

	"github.com/rywk/minigoao/pkg/constants/attack"
)

type Image = uint32

const (
	Nothing Image = iota
	// Ground assets
	Grass
	Bricks
	MossBricks
	SandBricks
	PvPTeam1Tile
	PvPTeam2Tile
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

	// items

	// Effects
	MeleeHit
	SpellInmo
	SpellInmoRm
	SpellApoca
	SpellDesca
	SpellHealWounds
	SpellHealArea
	SpellResurrect

	// Mark
	// Place ever asset that is a solid block after `SolidBlocks`
	SolidBlocks
	// ---
	Shroom
	Rock
	Tree1
	// ---

	// Can be used as the total of assests
	Len
)

func CanFight(a Image) bool {
	return a == Bricks || a == SandBricks
}
func IsSolid(a Image) bool {
	return a > SolidBlocks && a < Len
}
func BattleStarter(a Image) bool {
	return a == PvPTeam1Tile || a == PvPTeam2Tile
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
	SpellResurrectSound
	SpellHealWoundsSound
	SpellInmoSound
	SpellInmoRmSound
	SpellApocaSound
	SpellDescaSound
	Death
	KillBell
)

func SoundFromSpell(s attack.Spell) Sound {
	switch s {
	case attack.SpellExplode:
		return SpellApocaSound
	case attack.SpellParalize:
		return SpellInmoSound
	case attack.SpellRemoveParalize:
		return SpellInmoRmSound
	case attack.SpellElectricDischarge:
		return SpellDescaSound
	case attack.SpellHealWounds:
		return SpellHealWoundsSound
	case attack.SpellResurrect:
		return SpellResurrectSound

	}
	return 0
}
