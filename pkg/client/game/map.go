package game

import (
	"image/color"

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
	g           *Game
	cfg         *MapConfig
	mapBgChunks [][]*ebiten.Image
	Space       *grid.Grid
	drawOp      *ebiten.DrawImageOptions
}

const imageChunckSize = 256

func NewMap(g *Game, c *MapConfig) *Map {
	m := &Map{
		g:      g,
		cfg:    c,
		drawOp: &ebiten.DrawImageOptions{},
		Space:  grid.NewGrid(int32(constants.WorldX), int32(constants.WorldY), 3),
	}
	// chunks

	// init empty images what will store the map in chunks
	mapXChunks := constants.WorldX*constants.TileSize/imageChunckSize + 1

	mapYChunks := constants.WorldY*constants.TileSize/imageChunckSize + 1
	tilePerChunk := imageChunckSize / constants.TileSize

	m.mapBgChunks = make([][]*ebiten.Image, mapXChunks)

	for x := range m.mapBgChunks {
		m.mapBgChunks[x] = make([]*ebiten.Image, mapYChunks)
		for y := range m.mapBgChunks[x] {
			m.mapBgChunks[x][y] = ebiten.NewImage(imageChunckSize, imageChunckSize)
			m.mapBgChunks[x][y].Fill(color.Black)
			startX, startY := x*tilePerChunk, y*tilePerChunk
			for tx := range tilePerChunk {
				for ty := range tilePerChunk {
					tileX, tileY := startX+tx, startY+ty
					if tileX < 0 || tileX > len(c.GroundMapTextures)-1 || tileY < 0 || tileY > len(c.GroundMapTextures[tileX])-1 {
						continue
					}
					floorT := c.GroundMapTextures[tileX][tileY]
					stuffT := c.StuffMapTextures[tileX][tileY]
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(tx*constants.TileSize), float64(ty*constants.TileSize))
					texture.LoadFloorTexture(floorT, tileX%4, tileY%4).Draw(m.mapBgChunks[x][y], op)
					if assets.IsSolid(stuffT) {
						m.Space.Set(1, typ.P{X: int32(tileX), Y: int32(tileY)}, uint16(stuffT))
						texture.LoadTexture(stuffT).Draw(m.mapBgChunks[x][y], op)
					}
				}
			}
		}
	}
	return m
}

func (m *Map) RenderWorld(screen *ebiten.Image, render func(wi *ebiten.Image, offset ebiten.GeoM)) {
	minX, minY := (m.g.player.Pos[0])-HalfScreenX+16, (m.g.player.Pos[1])-HalfScreenY+48
	maxX, maxY := minX+ScreenWidth, minY+ScreenHeight-64

	if minX < 0 || minY < 0 || maxX > constants.WorldX*constants.TileSize || maxY > constants.WorldY*constants.TileSize {
		screen.Fill(color.Black)
	}

	stChnkX := int(minX) / imageChunckSize
	stChnkY := int(minY) / imageChunckSize

	chunksX := ScreenWidth/imageChunckSize + 1
	chunksY := ScreenHeight/imageChunckSize + 1

	endChnkX := stChnkX + chunksX
	endChnkY := stChnkY + chunksY

	chnkOffX := int(minX) - stChnkX*imageChunckSize
	chnkOffY := int(minY) - stChnkY*imageChunckSize

	ge := ebiten.GeoM{}
	ge.Translate(-float64(chnkOffX), -float64(chnkOffY))
	cx, cy := 0, 0
	cop := &ebiten.DrawImageOptions{}
	for chx := stChnkX; chx < endChnkX+1; chx++ {
		if chx < 0 || chx > len(m.mapBgChunks)-1 {
			cx += imageChunckSize
			continue
		}

		cy = 0
		for chy := stChnkY; chy < endChnkY+1; chy++ {
			if chy < 0 || chy > len(m.mapBgChunks[chx])-1 {
				cy += imageChunckSize
				continue
			}

			cop.GeoM.Reset()
			cop.GeoM.Concat(ge)
			cop.GeoM.Translate(float64(cx), float64(cy))
			screen.DrawImage(m.mapBgChunks[chx][chy], cop)
			cy += imageChunckSize
		}
		cx += imageChunckSize
	}
	off := ebiten.GeoM{}
	off.Translate(-float64(minX), -float64(minY))
	render(screen, off)
}

func MapSoundToPlayer(p *player.P, x, y int) (float64, float64) {
	diffX, diffY := p.X-int32(x), p.Y-int32(y)
	return float64(diffX) * 0.08, float64(diffY) * 0.08
}
