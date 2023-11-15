package maps

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/proto/message/assets"
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
	FillLayer(constants.WorldX, constants.WorldY, assets.Grass),
	RandomShroomLayer(constants.WorldX, constants.WorldY),
}

func RandomShroomLayer(width, height int) [][]assets.Image {
	layer := make([][]assets.Image, height)
	for y := 0; y < height; y++ {
		layer[y] = make([]assets.Image, width)
		for x := 0; x < width; x++ {
			a := assets.Nothing
			if x%25 == 0 && y%25 == 0 {
				a = assets.Shroom
			}
			layer[y][x] = a
		}
	}
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

type YSortable interface {
	ValueY() float64
	Draw(*ebiten.Image)
}
