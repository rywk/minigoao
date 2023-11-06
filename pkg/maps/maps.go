package maps

import (
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

var MapLayers = [layerTypes][][]assets.Asset{
	FillLayer(constants.WorldX, constants.WorldY, assets.Grass),
	RandomShroomLayer(constants.WorldX, constants.WorldY),
}

func RandomShroomLayer(width, height int) [][]assets.Asset {
	layer := make([][]assets.Asset, height)
	for y := 0; y < height; y++ {
		layer[y] = make([]assets.Asset, width)
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

func FillLayer(width, height int, a assets.Asset) [][]assets.Asset {
	layer := make([][]assets.Asset, height)
	for y := 0; y < height; y++ {
		layer[y] = make([]assets.Asset, width)
		for x := 0; x < width; x++ {
			layer[y][x] = a
		}
	}
	return layer
}
