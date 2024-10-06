package constants

import "time"

const (
	Port = ":5555"
)

const (
	TileSize                 = 32
	WorldX, WorldY           = 100, 100
	PixelWorldX, PixelWorldY = WorldX * TileSize, WorldY * TileSize

	// We keep a constant to use arrays
	// and preallocate a mximun of online players
	// and use like a fast map, idk
	MaxConnCount = 50

	GridViewportX, GridViewportY = 43, 31

	ChatMsgTTL = time.Second * 10

	PotionCooldown = time.Millisecond * 275
)

type Err struct{}

func (Err) Error() string { return "" }
