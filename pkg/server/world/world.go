package world

import (
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/maps"
	"github.com/rywk/minigoao/pkg/server/world/thing"
	"github.com/rywk/minigoao/proto/message/assets"
	"github.com/rywk/tile"
)

// PlayerGrid has all the things that matter to a players mobility and interactions
// meaning:
// - where players are (they can hit/cast/block eachother)
// - where stuff is (it can block movement/create effect/create action)
var PlayerGrid = tile.NewGridOf[thing.Thing](constants.WorldX, constants.WorldY)

func init() {
	FillBlockers(maps.MapLayers[maps.Stuff], PlayerGrid)
}

func FillBlockers(aa [][]assets.Image, g *tile.Grid[thing.Thing]) {
	for y := 0; y < len(aa); y++ {
		for x := 0; x < len(aa[y]); x++ {
			if assets.IsSolid(aa[y][x]) {
				t, _ := g.At(int16(x), int16(y))
				// we dont care about this pointer because it should never be deleted
				t.Add(&thing.Solid{})
			}
		}
	}
}
