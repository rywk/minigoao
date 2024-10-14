package mapdef

import (
	"image"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/grid"
)

type MapType uint8

const (
	MapNone MapType = iota
	MapLobby
	MapPvP1v1
	MapPvP2v2
)

// Map Layers
// - Ground
// - Stuff
type Layer uint8

const (
	Ground Layer = iota
	Stuff
	TallStuff
	Players
	LayerTypes
)

func (l Layer) Int() int {
	return int(l)
}

var (
	PvPSpawner1v1 = image.Pt(28, 41)
	PvPSpawner2v2 = image.Pt(28, 43)
)

func In1v1Spawn(p image.Point) bool {
	return p.Eq(PvPSpawner1v1) || p.Eq(PvPSpawner1v1.Add(image.Pt(2, 0)))
}
func OponentPvP1(space *grid.Grid, p image.Point) uint16 {
	np := p
	if p.Eq(PvPSpawner1v1) {
		np = PvPSpawner1v1.Add(image.Pt(2, 0))
	} else {
		np = PvPSpawner1v1
	}
	oponent := space.GetPoint(Players.Int(), np)
	return oponent
}
func AllayPvP2(space *grid.Grid, p image.Point) uint16 {
	np := p
	if p.Eq(PvPSpawner2v2) {
		np = PvPSpawner2v2.Add(image.Pt(0, 1))
	} else if p.Eq(PvPSpawner2v2.Add(image.Pt(0, 1))) {
		np = PvPSpawner2v2
	} else if p.Eq(PvPSpawner2v2.Add(image.Pt(2, 0))) {
		np = PvPSpawner2v2.Add(image.Pt(2, 1))
	} else if p.Eq(PvPSpawner2v2.Add(image.Pt(2, 1))) {
		np = PvPSpawner2v2.Add(image.Pt(2, 0))
	}
	allay := space.GetPoint(Players.Int(), np)
	return allay
}

func OponentPvP2(space *grid.Grid, p image.Point) (uint16, uint16) {
	np1 := p
	np2 := p
	if p.Eq(PvPSpawner2v2) || p.Eq(PvPSpawner2v2.Add(image.Pt(0, 1))) {
		np1 = PvPSpawner2v2.Add(image.Pt(2, 0))
		np2 = PvPSpawner2v2.Add(image.Pt(2, 1))
	} else if p.Eq(PvPSpawner2v2.Add(image.Pt(2, 0))) || p.Eq(PvPSpawner2v2.Add(image.Pt(2, 1))) {
		np1 = PvPSpawner2v2
		np2 = PvPSpawner2v2.Add(image.Pt(0, 1))
	}
	op1 := space.GetPoint(Players.Int(), np1)
	op2 := space.GetPoint(Players.Int(), np2)
	return op1, op2
}
func In2v2Spawn(p image.Point) bool {
	return p.Eq(PvPSpawner2v2) ||
		p.Eq(PvPSpawner2v2.Add(image.Pt(2, 0))) ||
		p.Eq(PvPSpawner2v2.Add(image.Pt(0, 1))) ||
		p.Eq(PvPSpawner2v2.Add(image.Pt(2, 1)))
}

var LobbyMapLayers = [LayerTypes][][]assets.Image{
	NewLayer(constants.WorldX, constants.WorldY, assets.Grass).
		Fill(assets.MossBricks, arena1v1Floor.Inset(-2).Add(image.Pt(5, 29))).
		Fill(assets.SandBricks, arenaBigFloor.Add(image.Pt(22, 3))).
		Fill(assets.MossBricks, image.Rect(
			arenaBigFloor.Add(image.Pt(22, 3)).Min.X,
			arenaBigFloor.Add(image.Pt(22, 3)).Max.Y,
			arenaBigFloor.Add(image.Pt(22, 3)).Max.X,
			arenaBigFloor.Add(image.Pt(22, 3)).Max.Y+2)).
		Fill(assets.Bricks, arena1v1Floor.Add(image.Pt(25, 29))).
		Fill(assets.Bricks, arena2v2Floor.Add(image.Pt(41, 29))).
		PlaceTiles(func(tiles [][]assets.Image) {
			tiles[PvPSpawner1v1.X][PvPSpawner1v1.Y] = assets.PvPTeam1Tile
			tiles[PvPSpawner1v1.X+2][PvPSpawner1v1.Y] = assets.PvPTeam2Tile

			tiles[PvPSpawner2v2.X][PvPSpawner2v2.Y] = assets.PvPTeam1Tile
			tiles[PvPSpawner2v2.X][PvPSpawner2v2.Y+1] = assets.PvPTeam1Tile
			tiles[PvPSpawner2v2.X+2][PvPSpawner2v2.Y] = assets.PvPTeam2Tile
			tiles[PvPSpawner2v2.X+2][PvPSpawner2v2.Y+1] = assets.PvPTeam2Tile
		}).L(),
	RandomShroomLayer(constants.WorldX, constants.WorldY).L(),
}
var Onev1MapLayers = [LayerTypes][][]assets.Image{
	NewLayer(Arena1v1.Max.X, Arena1v1.Max.Y, assets.Bricks).L(),
	NewLayer(Arena1v1.Max.X, Arena1v1.Max.Y, assets.Nothing).
		Edges(assets.Rock, Arena1v1).L(),
}
var Twov2MapLayers = [LayerTypes][][]assets.Image{
	NewLayer(Arena2v2.Max.X, Arena2v2.Max.Y, assets.Bricks).L(),
	NewLayer(Arena2v2.Max.X, Arena2v2.Max.Y, assets.Nothing).
		Edges(assets.Rock, Arena2v2).L(),
}
var Arena1v1 = image.Rect(0, 0, 9, 9)
var arena1v1Floor = image.Rect(0, 0, 8, 8)
var Arena2v2 = image.Rect(0, 0, 16, 12)
var arena2v2Floor = image.Rect(0, 0, 15, 11)

var arenaBig = image.Rect(0, 0, 43, 23)
var arenaBigFloor = image.Rect(0, 0, 42, 22)

func RandomShroomLayer(width, height int) *MapLayer {
	layer := make([][]assets.Image, width)
	for x := 0; x < width; x++ {
		layer[x] = make([]assets.Image, height)
		layer[x][0] = assets.Shroom
		layer[x][height-1] = assets.Shroom
	}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			layer[0][y] = assets.Shroom
			layer[width-1][y] = assets.Shroom
		}
	}

	arena1v1n1 := Arena1v1.Add(image.Pt(24, 28))
	arena2v2n1 := Arena2v2.Add(image.Pt(40, 28))

	for y := arena1v1n1.Min.Y; y < arena1v1n1.Max.Y; y++ {
		layer[arena1v1n1.Min.X][y] = assets.Rock
		layer[arena1v1n1.Max.X][y] = assets.Rock
	}
	for x := arena1v1n1.Min.X; x < arena1v1n1.Max.X; x++ {
		layer[x][arena1v1n1.Min.Y] = assets.Rock
		layer[x][arena1v1n1.Max.Y] = assets.Rock

	}
	layer[32][37] = assets.Nothing

	for y := arena2v2n1.Min.Y; y < arena2v2n1.Max.Y; y++ {
		layer[arena2v2n1.Min.X][y] = assets.Rock
		layer[arena2v2n1.Max.X][y] = assets.Rock
	}
	for x := arena2v2n1.Min.X; x < arena2v2n1.Max.X; x++ {
		layer[x][arena2v2n1.Min.Y] = assets.Rock
		layer[x][arena2v2n1.Max.Y] = assets.Rock

	}
	layer[55][40] = assets.Nothing
	layer[54][40] = assets.Nothing
	layer[40][40] = assets.Nothing
	layer[41][40] = assets.Nothing
	layer[42][40] = assets.Nothing

	arenaBign1 := arenaBig.Add(image.Pt(21, 2))

	for y := arenaBign1.Min.Y; y < arenaBign1.Max.Y; y++ {
		layer[arenaBign1.Min.X][y] = assets.Rock
		layer[arenaBign1.Max.X][y] = assets.Rock
	}
	for x := arenaBign1.Min.X; x < arenaBign1.Max.X; x++ {
		layer[x][arenaBign1.Min.Y] = assets.Rock
		//layer[x][arenaBign1.Max.Y] = assets.Rock

	}
	return &MapLayer{l: layer}
}

type MapLayer struct {
	l [][]assets.Image
}

func NewLayer(width, height int, a assets.Image) *MapLayer {
	layer := make([][]assets.Image, width)
	for x := 0; x < width; x++ {
		layer[x] = make([]assets.Image, height)
		for y := 0; y < height; y++ {
			layer[x][y] = a
		}
	}
	return &MapLayer{l: layer}
}
func (ml *MapLayer) Fill(a assets.Image, rect image.Rectangle) *MapLayer {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			ml.l[x][y] = a
		}
	}
	return ml
}
func (ml *MapLayer) Edges(a assets.Image, rect image.Rectangle) *MapLayer {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		ml.l[x][0] = a
		ml.l[x][rect.Max.Y-1] = a
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		ml.l[0][y] = a
		ml.l[rect.Max.X-1][y] = a
	}
	return ml
}

func (ml *MapLayer) PlaceTiles(place func(tiles [][]assets.Image)) *MapLayer {
	place(ml.l)
	return ml
}

func (ml *MapLayer) L() [][]assets.Image {
	return ml.l
}
