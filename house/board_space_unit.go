package house

import "image"

// TODO(tmckee): does this belong in the 'perspective' package?
type BoardSpaceUnit int

type RoomSizey interface {
	GetDx() BoardSpaceUnit
	GetDy() BoardSpaceUnit
}

func ImageRect(x1, y1, x2, y2 BoardSpaceUnit) image.Rectangle {
	return image.Rect(int(x1), int(y1), int(x2), int(y2))
}
