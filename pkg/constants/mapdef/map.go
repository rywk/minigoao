package mapdef

import (
	"image"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
)

// Map Layers
// - Ground
// - Stuff
type Layer uint8

const (
	Ground Layer = iota
	Stuff
	TallStuff
	layerTypes
)

var LobbyMapLayers = [layerTypes][][]assets.Image{
	NewLayer(constants.WorldX, constants.WorldY, assets.Grass).
		Fill(assets.Bricks, image.Rect(24, 28, 32, 36)).
		Fill(assets.Bricks, image.Rect(40, 29, 57, 42)).L(),
	RandomShroomLayer(constants.WorldX, constants.WorldY),
}
var Onev1MapLayers = [layerTypes][][]assets.Image{
	NewLayer(arena1v1.Max.X, arena1v1.Max.Y, assets.Bricks).L(),
	NewLayer(arena1v1.Max.X, arena1v1.Max.Y, assets.Nothing).
		Edges(assets.Rock, arena1v1).L(),
}
var arena1v1 = image.Rect(0, 0, 7, 7)
var arena2v2 = image.Rect(0, 0, 16, 12)

func RandomShroomLayer(width, height int) [][]assets.Image {
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
			if x%25 == 0 && y%25 == 0 {
				layer[x][y] = assets.Shroom
			}

		}
	}
	arena1v1 := image.Rect(0, 0, 7, 7)
	arena2v2 := image.Rect(0, 0, 16, 12)
	arena1v1n1 := arena1v1.Add(image.Point{X: 24, Y: 28})
	arena2v2n1 := arena2v2.Add(image.Point{X: 40, Y: 29})

	for y := arena1v1n1.Min.Y; y < arena1v1n1.Max.Y; y++ {
		layer[arena1v1n1.Min.X][y] = assets.Rock
		layer[arena1v1n1.Max.X][y] = assets.Rock
	}
	for x := arena1v1n1.Min.X; x < arena1v1n1.Max.X; x++ {
		layer[x][arena1v1n1.Min.Y] = assets.Rock
		layer[x][arena1v1n1.Max.Y] = assets.Rock

	}
	layer[30][35] = assets.Nothing

	for y := arena2v2n1.Min.Y; y < arena2v2n1.Max.Y; y++ {
		layer[arena2v2n1.Min.X][y] = assets.Rock
		layer[arena2v2n1.Max.X][y] = assets.Rock
	}
	for x := arena2v2n1.Min.X; x < arena2v2n1.Max.X; x++ {
		layer[x][arena2v2n1.Min.Y] = assets.Rock
		layer[x][arena2v2n1.Max.Y] = assets.Rock

	}
	layer[55][41] = assets.Nothing
	layer[54][41] = assets.Nothing
	layer[40][41] = assets.Nothing
	layer[41][41] = assets.Nothing
	layer[42][41] = assets.Nothing
	return layer
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
func (ml *MapLayer) L() [][]assets.Image {
	return ml.l
}
