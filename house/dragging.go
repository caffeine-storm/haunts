package house

import (
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

type dragState struct {
	on     bool
	refpos gui.Point
	focus  struct {
		X, Y float32
	}
}

type draggable interface {
	GetFocus() (float32, float32)
	SetFocusTarget(float64, float64)
	modelviewToBoard(float32, float32) (x, y, dist float32)
}

func (ds *dragState) dragToggle(drag draggable, group gui.EventGroup) {
	if ds.on {
		if group.PrimaryEvent().IsPress() {
			// continue dragging
		} else {
			// done dragging
			ds.on = false
		}
	} else {
		if group.PrimaryEvent().IsPress() {
			// start dragging
			ds.on = true
			ds.refpos = group.GetMousePosition()
			ds.focus.X, ds.focus.Y = drag.GetFocus()
		} else {
			// still not dragging
		}
	}
}

func (ds *dragState) dragUpdate(drag draggable, group gui.EventGroup) {
	if !ds.on {
		return
	}

	screenMousePos := group.GetMousePosition()

	// Move the focus so that the on-board drag-start position aligns with the
	// current mouse location.
	startx, starty, _ := drag.modelviewToBoard(float32(ds.refpos.X), float32(ds.refpos.Y))
	curx, cury, _ := drag.modelviewToBoard(float32(screenMousePos.X), float32(screenMousePos.Y))

	deltax := curx - startx
	deltay := cury - starty

	drag.SetFocusTarget(float64(ds.focus.X-deltax), float64(ds.focus.Y-deltay))
}

func (ds *dragState) HandleEventGroup(drag draggable, group gui.EventGroup) bool {
	rightButtonId := gin.KeyId{
		Index: gin.MouseRButton,
		Device: gin.DeviceId{
			Index: gin.DeviceIndexAny,
			Type:  gin.DeviceTypeMouse,
		},
	}

	ret := false
	if rightButtonId.Contains(group.PrimaryEvent().Key.Id()) {
		ds.dragToggle(drag, group)
		ret = true
	}
	if group.IsMouseMove() {
		ds.dragUpdate(drag, group)
		ret = true
	}
	return ret
}
