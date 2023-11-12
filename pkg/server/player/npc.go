package player

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/constants/direction"
	asset "github.com/rywk/minigoao/proto/message/assets"
)

type NPC = uint32

const (
	ResurrectTotem NPC = iota
)

type NpcSpecial struct {
	img        *ebiten.Image
	offx, offy int
}

type Npc struct {
	// Npc ID
	ID   uint32
	Type NPC
	// Npc position on map
	X, Y int

	D direction.D

	// Npc health points
	HP, MaxHP int

	// Inmobilized
	Inmobilized bool

	// IDs for skins
	Armor            asset.Image
	Helmet           asset.Image
	Weapon           asset.Image
	Shield           asset.Image
	TextureOverwtire *NpcSpecial

	// Internal npc
	//Handler  *NPCHandler

}
