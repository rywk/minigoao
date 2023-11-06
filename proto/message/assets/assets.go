package assets

import "reflect"

type Asset = uint32

const (
	// Ground assets
	Grass Asset = iota

	// Skins
	DarkArmour
	WarAxe
	Head
	Tiletest

	// Mark
	// Place ever asset that is a solid block after `SolidBlocks`
	SolidBlocks
	// ---
	Shroom
	Tree1
	// ---

	// Can be used as the total of assests
	Nothing
)

func IsSolid(a Asset) bool {
	return a > SolidBlocks && a < Nothing
}

func AssetName(a Asset) string {
	return reflect.TypeOf(a).Name()
}
