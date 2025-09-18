package house

import (
	"image"

	"github.com/MobRulesGames/haunts/house/perspective"
)

type BoardSpaceUnit = perspective.BoardSpaceUnit

// Helper to convert pairs of values to pairs of BoardSpaceUnit
func BoardSpaceUnitPair[T int | float32 | float64](x, y T) (BoardSpaceUnit, BoardSpaceUnit) {
	return BoardSpaceUnit(x), BoardSpaceUnit(y)
}

type RoomSizey interface {
	GetDx() BoardSpaceUnit
	GetDy() BoardSpaceUnit
}

func ImageRect(x1, y1, x2, y2 BoardSpaceUnit) image.Rectangle {
	return image.Rect(int(x1), int(y1), int(x2), int(y2))
}
