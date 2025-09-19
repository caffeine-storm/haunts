package house

import (
	"github.com/MobRulesGames/haunts/base"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

// TODO(tmckee:#34): rename this file to 'wall_panel.go'

type WallPanel struct {
	*gui.VerticalTable
	room   *Room
	viewer *roomViewer

	decal          *Decal
	prev_decal     *Decal
	drag_anchor    struct{ X, Y float32 }
	selected_walls map[int]bool
}

func MakeWallPanel(room *Room, viewer *roomViewer) *WallPanel {
	var wp WallPanel
	wp.room = room
	wp.viewer = viewer
	wp.VerticalTable = gui.MakeVerticalTable()
	wp.selected_walls = make(map[int]bool)

	decal_table := gui.MakeVerticalTable()
	fnames := GetAllDecalNames()
	for i := range fnames {
		name := fnames[i]
		decal_table.AddChild(gui.MakeButton("standard_18", name, 300, 1, 1, 1, 1, func(ctx gui.EventHandlingContext, t int64) {
			decal := LoadDecal(name)
			if decal == nil {
				return
			}
			wp.decal = decal
			wp.decal.temporary = true
			wp.room.Decals = append(wp.room.Decals, wp.decal)
			wp.drag_anchor.X = 0
			wp.drag_anchor.Y = 0
		}))
	}
	wp.VerticalTable.AddChild(gui.MakeScrollFrame(decal_table, 300, 700))

	return &wp
}

func (w *WallPanel) decalNear(wx, wy int) *Decal {
	for _, decal := range w.room.Decals {
		var xx, yy float32
		if decal.X > float32(w.room.Size.Dx) {
			xx, yy, _ = w.viewer.modelviewToRightWall(float32(wx), float32(wy))
		} else if decal.Y > float32(w.room.Size.Dy) {
			xx, yy, _ = w.viewer.modelviewToLeftWall(float32(wx), float32(wy))
		} else {
			xx, yy, _ = w.viewer.modelviewToBoard(float32(wx), float32(wy))
		}
		dx := float32(decal.Texture.Data().Dx()) / 100 / 2
		dy := float32(decal.Texture.Data().Dy()) / 100 / 2
		if xx > decal.X-dx && xx < decal.X+dx && yy > decal.Y-dy && yy < decal.Y+dy {
			return decal
		}
	}
	return nil
}

func (w *WallPanel) onEscape() {
	if w.decal != nil {
		if w.prev_decal != nil {
			*w.decal = *w.prev_decal
		} else {
			algorithm.Choose(&w.room.Decals, func(decal *Decal) bool {
				return decal != w.decal
			})
		}
	}
	w.decal = nil
	w.prev_decal = nil
}

func (w *WallPanel) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if w.VerticalTable.Respond(ui, group) {
		return true
	}

	if group.IsPressed(gin.AnyBackspace) || group.IsPressed(gin.AnyKeyDelete) {
		algorithm.Choose(&w.room.Decals, func(decal *Decal) bool {
			return decal != w.decal
		})
		w.decal = nil
		w.prev_decal = nil
		return true
	}

	if group.IsPressed(gin.AnyEscape) {
		w.onEscape()
		return true
	}

	if group.IsPressed(base.GetDefaultKeyMap()["flip"].Id()) {
		if w.decal != nil {
			w.decal.Flip = !w.decal.Flip
		}
		return true
	}

	// Hold space and scroll to rotate the decal.
	if w.decal != nil {
		if group.IsPressed(gin.AnySpace) {
			if event, found := group.FindEvent(gin.AnyMouseWheelVertical); found {
				w.decal.Rot += float32(event.Key.CurPressAmt() / 100)
			}
		}
	}

	if group.IsPressed(gin.AnyMouseLButton) {
		if w.decal != nil {
			w.decal.temporary = false
			w.decal = nil
		} else if w.decal == nil {
			if mpos, ok := ui.UseMousePosition(group); ok {
				w.decal = w.decalNear(mpos.X, mpos.Y)
				if w.decal != nil {
					w.prev_decal = new(Decal)
					*w.prev_decal = *w.decal
					w.decal.temporary = true

					wx, wy := w.viewer.BoardToWindowf(w.decal.X, w.decal.Y)
					w.drag_anchor.X = float32(mpos.X) - wx
					w.drag_anchor.Y = float32(mpos.Y) - wy
				}
			}
		}
		return true
	}
	return false
}

func (w *WallPanel) Think(ui *gui.Gui, t int64) {
	if w.decal != nil {
		// TODO(tmckee): need to ask the gui for cursor pos
		// px, py := gin.In().GetCursor("Mouse").Point()
		px, py := 0, 0
		tx := float32(px) - w.drag_anchor.X
		ty := float32(py) - w.drag_anchor.Y
		bx, by := w.viewer.WindowToBoardf(tx, ty)
		w.decal.X = bx
		w.decal.Y = by
	}
	w.VerticalTable.Think(ui, t)
}

func (w *WallPanel) Collapse() {
	w.onEscape()
}

func (w *WallPanel) Expand() {
}

func (w *WallPanel) Reload() {
	w.onEscape()
}
