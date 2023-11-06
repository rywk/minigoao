package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/maps"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message"
	"github.com/rywk/minigoao/proto/message/assets"
	"github.com/rywk/tile"
)

// Hmmm, this will be heavy on ram but i think it will make the game way more fluid
// like for example, dont ask the server to move if you are seeing you cant move..
var LocalGrid = tile.NewGridOf[thing.Thing](constants.WorldX, constants.WorldY)

func MustAt(x, y int) tile.Tile[thing.Thing] {
	t, _ := LocalGrid.At(int16(x), int16(y))
	return t
}

type MapConfig struct {
	Width, Height         int
	StartX, StartY        int
	ViewWidth, ViewHeight int
	GroundMapTextures     [][]assets.Asset
	StuffMapTextures      [][]assets.Asset
}

func MapConfigFromRegisterOk(rok *message.RegisterOk) *MapConfig {
	mc := MapConfig{}
	log.Println(rok.Id)
	mc.Width, mc.Height = constants.PixelWorldX, constants.PixelWorldY
	mc.StartX, mc.StartY = int(rok.Self.X)*constants.TileSize, int(rok.Self.Y)*constants.TileSize
	mc.ViewWidth, mc.ViewHeight = int(rok.FovX), int(rok.FovY)
	mc.GroundMapTextures = maps.MapLayers[maps.Ground]
	mc.StuffMapTextures = maps.MapLayers[maps.Stuff]
	return &mc
}

type Map struct {
	world      *ebiten.Image
	floorTiles [][]texture.T
	stuffTiles [][]texture.T
}

func NewMap(c *MapConfig) *Map {
	m := &Map{}
	// Use config to load textures for floor tiles
	m.floorTiles = make([][]texture.T, constants.PixelWorldX/texture.GrassTextureSize)
	for i := range m.floorTiles {
		m.floorTiles[i] = make([]texture.T, constants.PixelWorldY/texture.GrassTextureSize)
		for j := range m.floorTiles[i] {
			m.floorTiles[i][j] = texture.LoadTexture(c.GroundMapTextures[i][j])
			if m.floorTiles[i][j] == nil {
				log.Println("asdasd", c.GroundMapTextures[i][j])
			}
		}
	}

	// Use config to load textures for stuff over floor
	m.stuffTiles = make([][]texture.T, len(c.StuffMapTextures))
	for i, y := range c.StuffMapTextures {
		m.stuffTiles[i] = make([]texture.T, len(c.StuffMapTextures))
		for j, t := range y {
			m.stuffTiles[i][j] = texture.LoadTexture(t)
			if assets.IsSolid(t) {
				tl, _ := LocalGrid.At(int16(j), int16(i))
				tl.Add(&thing.Solid{})
			}
		}
	}
	m.world = ebiten.NewImage(len(m.stuffTiles[0])*constants.TileSize, len(m.stuffTiles)*constants.TileSize)
	return m
}

func (m *Map) Update() {
	// ? eventually npcs?
}

func (m *Map) Draw() {
	for y := range m.floorTiles {
		for x := range m.floorTiles[y] {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x*texture.GrassTextureSize), float64(y*texture.GrassTextureSize))
			if m.floorTiles[y][x] == nil {
				continue
			}
			m.floorTiles[y][x].Draw(m.world, op)
		}
	}
	for y := range m.stuffTiles {
		for x := range m.stuffTiles[y] {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x*constants.TileSize), float64(y*constants.TileSize))
			m.stuffTiles[y][x].Draw(m.world, op)
		}
	}
}

func (m *Map) Image() *ebiten.Image {
	return m.world
}
