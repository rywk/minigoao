package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/mapdef"
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
	mc.Width, mc.Height = constants.PixelWorldX, constants.PixelWorldY
	mc.StartX, mc.StartY = int(p.Pos.X)*constants.TileSize, int(p.Pos.Y)*constants.TileSize
	mc.ViewWidth, mc.ViewHeight = int(constants.GridViewportX), int(constants.GridViewportX)
	mc.GroundMapTextures = mapdef.MapLayers[mapdef.Ground]
	mc.StuffMapTextures = mapdef.MapLayers[mapdef.Stuff]
	return &mc
}

type Map struct {
	world      *ebiten.Image
	floor      *ebiten.Image
	floorTiles [][]texture.T
	stuffTiles [][]texture.T
	Space      *grid.Grid
	drawOp     *ebiten.DrawImageOptions
}

func NewMap(c *MapConfig) *Map {
	m := &Map{
		drawOp: &ebiten.DrawImageOptions{},
		Space:  grid.NewGrid(int32(constants.WorldX), int32(constants.WorldY), 3),
	}
	// Use config to load textures for floor tiles
	m.floor = ebiten.NewImage(len(c.GroundMapTextures[0])*constants.TileSize, len(c.GroundMapTextures)*constants.TileSize)

	m.floorTiles = make([][]texture.T, len(c.GroundMapTextures))
	for i := range m.floorTiles {
		m.floorTiles[i] = make([]texture.T, len(c.GroundMapTextures))
		for j := range m.floorTiles[i] {
			x := j % 4
			y := i % 4
			m.floorTiles[i][j] = texture.LoadFloorTexture(c.GroundMapTextures[i][j], x, y)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(j*constants.TileSize), float64(i*constants.TileSize))
			m.floorTiles[i][j].Draw(m.floor, op)
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
	minX, minY := pos.X*constants.TileSize-pixelXView/2, pos.Y*constants.TileSize-pixelYView/2
	maxX, maxY := pos.X*constants.TileSize+pixelXView/2, pos.Y*constants.TileSize+pixelYView/2

	if int32(minX)-pos.X*constants.TileSize < 0 {
		minX = 0
	}
	if int32(minY)-pos.Y*constants.TileSize < 0 {
		minY = 0
	}
	if int32(maxX)+pos.X*constants.TileSize > constants.WorldX*constants.TileSize {
		maxX = constants.WorldX * constants.TileSize
	}
	if int32(maxY)+pos.Y*constants.TileSize < constants.WorldY*constants.TileSize {
		maxY = constants.WorldY * constants.TileSize
	}

	visibleFloor := m.floor.SubImage(image.Rect(int(minX), int(minY), int(maxX), int(maxY))).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(minX), float64(minY))
	m.world.DrawImage(visibleFloor, op)

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

			m.drawOp.GeoM.Reset()
			m.drawOp.GeoM.Translate(float64(xpx), float64(ypx))
			m.stuffTiles[y][x].Draw(m.world, m.drawOp)
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
