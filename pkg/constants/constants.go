package constants

const (
	Port = ":25565"
)

const (
	TileSize                 = 32
	WorldX, WorldY           = 270, 270
	PixelWorldX, PixelWorldY = WorldX * TileSize, WorldY * TileSize

	// We keep a constant to use arrays
	// and preallocate a mximun of online players
	// and use like a fast map, idk
	MaxConnCount = 50
)

const (
	EmptyTileData = 333
)

type Err struct{}

func (Err) Error() string { return "" }
