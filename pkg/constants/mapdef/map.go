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
	layerTypes
)

var MapLayers = [layerTypes][][]assets.Image{
	NewLayer(constants.WorldX, constants.WorldY, assets.Grass).
		Fill(assets.Bricks, image.Rect(24, 28, 32, 36)).
		Fill(assets.Bricks, image.Rect(40, 29, 57, 42)).L(),
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
	arena1v1 := image.Rect(0, 0, 7, 7)
	arena2v2 := image.Rect(0, 0, 16, 12)
	arena1v1n1 := arena1v1.Add(image.Point{X: 24, Y: 28})
	arena2v2n1 := arena2v2.Add(image.Point{X: 40, Y: 29})

	for y := arena1v1n1.Min.Y; y < arena1v1n1.Max.Y; y++ {
		layer[y][arena1v1n1.Min.X] = assets.Rock
		layer[y][arena1v1n1.Max.X] = assets.Rock
	}
	for x := arena1v1n1.Min.X; x < arena1v1n1.Max.X; x++ {
		layer[arena1v1n1.Min.Y][x] = assets.Rock
		layer[arena1v1n1.Max.Y][x] = assets.Rock

	}
	layer[35][30] = assets.Nothing

	for y := arena2v2n1.Min.Y; y < arena2v2n1.Max.Y; y++ {
		layer[y][arena2v2n1.Min.X] = assets.Rock
		layer[y][arena2v2n1.Max.X] = assets.Rock
	}
	for x := arena2v2n1.Min.X; x < arena2v2n1.Max.X; x++ {
		layer[arena2v2n1.Min.Y][x] = assets.Rock
		layer[arena2v2n1.Max.Y][x] = assets.Rock

	}
	layer[41][55] = assets.Nothing
	layer[41][54] = assets.Nothing
	layer[41][40] = assets.Nothing
	layer[41][41] = assets.Nothing
	layer[41][42] = assets.Nothing
	return layer
}

type MapLayer struct {
	l [][]assets.Image
}

func NewLayer(width, height int, a assets.Image) *MapLayer {
	layer := make([][]assets.Image, height)
	for y := 0; y < height; y++ {
		layer[y] = make([]assets.Image, width)
		for x := 0; x < width; x++ {
			layer[y][x] = a
		}
	}
	return &MapLayer{l: layer}
}
func (ml *MapLayer) Fill(a assets.Image, rect image.Rectangle) *MapLayer {
	y := rect.Min.Y
	for ; y < rect.Max.Y; y++ {
		x := rect.Min.X
		for ; x < rect.Max.X; x++ {
			ml.l[y][x] = a
		}
	}
	return ml
}
func (ml *MapLayer) L() [][]assets.Image {
	return ml.l
}
