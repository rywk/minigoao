package game

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
)

type MapConfig struct {
	Width, Height         int
	StartX, StartY        int
	ViewWidth, ViewHeight int
	GroundMapTextures     [][]assets.Image
	StuffMapTextures      [][]assets.Image
}

func MapConfigFromPlayerLogin(p *msgs.EventPlayerLogin) *MapConfig {
	mc := MapConfig{}
	log.Println(p.ID)
	mc.Width, mc.Height = constants.PixelWorldX, constants.PixelWorldY
	mc.StartX, mc.StartY = int(p.Pos.X)*constants.TileSize, int(p.Pos.Y)*constants.TileSize
	mc.ViewWidth, mc.ViewHeight = int(constants.GridViewportX), int(constants.GridViewportX)
	mc.GroundMapTextures = MapLayers[Ground]
	mc.StuffMapTextures = MapLayers[Stuff]
	return &mc
}

type Map struct {
	world      *ebiten.Image
	floorTiles [][]texture.T
	stuffTiles [][]texture.T
	Space      *grid.Grid
}

func NewMap(c *MapConfig) *Map {
	m := &Map{
		Space: grid.NewGrid(int32(constants.WorldX), int32(constants.WorldY), 3),
	}
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
				m.Space.Set(1, typ.P{X: int32(j), Y: int32(i)}, uint16(t))
			}
		}
	}
	m.world = ebiten.NewImage(len(m.stuffTiles[0])*constants.TileSize, len(m.stuffTiles)*constants.TileSize)
	return m
}

func (m *Map) Update() {
	// ? eventually npcs?
}

const (
	pixelXView = constants.GridViewportX * constants.TileSize
	pixelYView = constants.GridViewportY * constants.TileSize
)

func (m *Map) Draw(pos typ.P) {
	minX, minY := pos.X*constants.TileSize-pixelXView, pos.Y*constants.TileSize-pixelYView
	maxX, maxY := pos.X*constants.TileSize+pixelXView, pos.Y*constants.TileSize+pixelYView
	for y := range m.floorTiles {
		ypx := int32(y * texture.GrassTextureSize)
		if ypx < minY || ypx > maxY {
			continue
		}
		for x := range m.floorTiles[y] {
			xpx := int32(x * texture.GrassTextureSize)
			if xpx < minX || xpx > maxX {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(xpx), float64(ypx))
			if m.floorTiles[y][x] == nil {
				continue
			}
			m.floorTiles[y][x].Draw(m.world, op)
		}
	}
	for y := range m.stuffTiles {
		ypx := int32(y * constants.TileSize)
		if ypx < minY || ypx > maxY {
			continue
		}
		for x := range m.stuffTiles[y] {
			xpx := int32(x * constants.TileSize)
			if xpx < minX || xpx > maxX {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(xpx), float64(ypx))
			m.stuffTiles[y][x].Draw(m.world, op)
		}
	}
}

func (m *Map) Image() *ebiten.Image {
	return m.world
}

func MapSoundToPlayer(p *player.P, x, y int) (float64, float64) {
	diffX, diffY := p.X-int32(x), p.Y-int32(y)
	return float64(diffX) * 0.08, float64(diffY) * 0.08
}

// Map Layers
// - Ground
// - Stuff
type Layer uint8

const (
	Ground Layer = iota
	Stuff
	layerTypes
)

var MapLayers = [layerTypes][][]assets.Image{
	FillLayer(constants.WorldX, constants.WorldY, assets.Grass),
	RandomShroomLayer(constants.WorldX, constants.WorldY),
}

func RandomShroomLayer(width, height int) [][]assets.Image {
	layer := make([][]assets.Image, height)
	for y := 0; y < height; y++ {
		layer[y] = make([]assets.Image, width)
		layer[y][0] = assets.Shroom
		layer[y][width-1] = assets.Shroom
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			layer[0][x] = assets.Shroom
			layer[height-1][x] = assets.Shroom
			if x%25 == 0 && y%25 == 0 {
				layer[y][x] = assets.Shroom
			}

		}
	}
	arena1v1 := image.Rect(0, 0, 8, 8)
	arena2v2 := image.Rect(0, 0, 16, 12)
	arena1v1n1 := arena1v1.Add(image.Point{X: 25, Y: 29})
	arena2v2n1 := arena2v2.Add(image.Point{X: 40, Y: 29})

	for y := arena1v1n1.Min.Y; y < arena1v1n1.Max.Y; y++ {
		layer[y][arena1v1n1.Min.X] = assets.Shroom
		layer[y][arena1v1n1.Max.X] = assets.Shroom
	}
	for x := arena1v1n1.Min.X; x < arena1v1n1.Max.X; x++ {
		layer[arena1v1n1.Min.Y][x] = assets.Shroom
		layer[arena1v1n1.Max.Y][x] = assets.Shroom

	}
	layer[37][32] = assets.Nothing

	for y := arena2v2n1.Min.Y; y < arena2v2n1.Max.Y; y++ {
		layer[y][arena2v2n1.Min.X] = assets.Shroom
		layer[y][arena2v2n1.Max.X] = assets.Shroom
	}
	for x := arena2v2n1.Min.X; x < arena2v2n1.Max.X; x++ {
		layer[arena2v2n1.Min.Y][x] = assets.Shroom
		layer[arena2v2n1.Max.Y][x] = assets.Shroom

	}
	layer[41][55] = assets.Nothing
	layer[41][54] = assets.Nothing
	layer[41][40] = assets.Nothing
	layer[41][41] = assets.Nothing
	return layer
}

func FillLayer(width, height int, a assets.Image) [][]assets.Image {
	layer := make([][]assets.Image, height)
	for y := 0; y < height; y++ {
		layer[y] = make([]assets.Image, width)
		for x := 0; x < width; x++ {
			layer[y][x] = a
		}
	}
	return layer
}

// func FillBlockers(aa [][]assets.Image, g *tile.Grid[thing.Thing]) {
// 	for y := 0; y < len(aa); y++ {
// 		for x := 0; x < len(aa[y]); x++ {
// 			if assets.IsSolid(aa[y][x]) {
// 				t, _ := g.At(int16(x), int16(y))
// 				// we dont care about this pointer because it should never be deleted
// 				t.Add(&thing.Solid{})
// 			}
// 		}
// 	}
// }
