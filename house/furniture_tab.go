package house

import (
	"image"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

type FurniturePanel struct {
	*gui.VerticalTable
	name       *gui.TextEditLine
	room_size  *gui.ComboBox
	floor_path *gui.FileWidget
	wall_path  *gui.FileWidget

	Room       *Room
	RoomViewer *roomViewer

	// If we're in the middle of moving an object and this widget gets collapsed
	// we want to put the object back where it was before we started dragging it.
	prev_object *Furniture

	// Distance from the mouse to the center of the object, in board coordinates
	drag_anchor struct{ x, y float32 }

	// The piece of furniture that we are currently dragging around
	furniture *Furniture

	key_map base.KeyMap
}

func (w *FurniturePanel) Collapse() {
	w.onEscape()
}
func (w *FurniturePanel) Expand() {
	w.RoomViewer.SetEditMode(editFurniture)
}

func makeFurniturePanel(room *Room, viewer *roomViewer) *FurniturePanel {
	var fp FurniturePanel
	fp.Room = room
	fp.RoomViewer = viewer
	fp.key_map = base.GetDefaultKeyMap()
	if room.Name == "" {
		room.Name = "name"
	}
	fp.name = gui.MakeTextEditLine("standard_18", room.Name, 300, 1, 1, 1, 1)

	fp.floor_path = gui.MakeFileWidget(room.Floor.GetPath(), imagePathFilter)
	// TODO(#39): we don't seem to be getting the right thing here?
	fp.wall_path = gui.MakeFileWidget(room.Wall.GetPath(), imagePathFilter)

	var args []string
	algorithm.Map(tags.RoomSizes, &args, func(a RoomSize) string { return a.String() })
	fp.room_size = gui.MakeComboTextBox(args, 300)
	for i := range tags.RoomSizes {
		if tags.RoomSizes[i].String() == room.Size.String() {
			fp.room_size.SetSelectedIndex(i)
			break
		}
	}
	fp.VerticalTable = gui.MakeVerticalTable()
	fp.VerticalTable.Params().Spacing = 3
	fp.VerticalTable.Params().Background.R = 0.3
	fp.VerticalTable.Params().Background.B = 1
	fp.VerticalTable.AddChild(fp.name)
	fp.VerticalTable.AddChild(fp.floor_path)
	fp.VerticalTable.AddChild(fp.wall_path)
	fp.VerticalTable.AddChild(fp.room_size)

	furn_table := gui.MakeVerticalTable()
	fnames := GetAllFurnitureNames()
	for _, fname := range fnames {
		furn_table.AddChild(gui.MakeButton("standard_18", fname, 300, 1, 1, 1, 1, func(ctx gui.EventHandlingContext, t int64) {
			f := MakeFurniture(fname)
			if f == nil {
				logging.Error("makeFurniturePanel>MakeFurniture(fname) failed", "fnames", fnames)
				return
			}
			fp.furniture = f
			fp.furniture.temporary = true
			fp.Room.Furniture = append(fp.Room.Furniture, fp.furniture)
			dx, dy := fp.furniture.Dims()
			fp.drag_anchor.x = float32(dx) / 2
			fp.drag_anchor.y = float32(dy) / 2
		}))
	}
	fp.VerticalTable.AddChild(gui.MakeScrollFrame(furn_table, 300, 600))

	return &fp
}

func (w *FurniturePanel) onEscape() {
	if w.furniture != nil {
		if w.prev_object != nil {
			*w.furniture = *w.prev_object
			w.prev_object = nil
		} else {
			algorithm.Choose(&w.Room.Furniture, func(f *Furniture) bool {
				return f != w.furniture
			})
		}
		w.furniture = nil
	}
}

func (w *FurniturePanel) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if w.VerticalTable.Respond(ui, group) {
		return true
	}

	// On escape we want to revert the furniture we're moving back to where it
	// was and what state it was in before we selected it.
	if group.IsPressed(gin.AnyEscape) {
		w.onEscape()
		return true
	}

	// If we hit delete then we want to remove the furniture we're moving around
	// from the room.
	if group.IsPressed(gin.AnyBackspace) || group.IsPressed(gin.AnyKeyDelete) {
		algorithm.Choose(&w.Room.Furniture, func(f *Furniture) bool {
			return f != w.furniture
		})
		w.furniture = nil
		w.prev_object = nil
		return true
	}

	if f := w.furniture; f != nil {
		if group.IsPressed(w.key_map["rotate left"].Id()) {
			f.RotateLeft()
		}
		if group.IsPressed(w.key_map["rotate right"].Id()) {
			f.RotateRight()
		}
		if group.IsPressed(w.key_map["flip"].Id()) {
			f.Flip = !f.Flip
		}
		if group.IsPressed(gin.AnyMouseLButton) {
			if !f.invalid {
				w.furniture.temporary = false
				w.furniture = nil
			}
		}
	} else {
		// w.furniture == nil
		if group.IsPressed(gin.AnyMouseLButton) {
			if mpos, ok := ui.UseMousePosition(group); ok {
				mx, my := mpos.X, mpos.Y
				bx, by := w.RoomViewer.WindowToBoard(mx, my)
				for i := range w.Room.Furniture {
					x, y := w.Room.Furniture[i].Pos()
					dx, dy := w.Room.Furniture[i].Dims()
					if int(bx) >= x && int(bx) < x+dx && int(by) >= y && int(by) < y+dy {
						w.furniture = w.Room.Furniture[i]
						w.prev_object = new(Furniture)
						*w.prev_object = *w.furniture
						w.furniture.temporary = true
						px, py := w.furniture.Pos()
						w.drag_anchor.x = bx - float32(px)
						w.drag_anchor.y = by - float32(py)
						break
					}
				}
			}
		}

		return true
	}

	return false
}

func (w *FurniturePanel) Reload() {
	for i := range tags.RoomSizes {
		if tags.RoomSizes[i].String() == w.Room.Size.String() {
			w.room_size.SetSelectedIndex(i)
			break
		}
	}
	w.name.SetText(w.Room.Name)
	w.floor_path.SetPath(w.Room.Floor.GetPath())
	w.wall_path.SetPath(w.Room.Wall.GetPath())
	w.onEscape()
}

func (w *FurniturePanel) Think(ui *gui.Gui, t int64) {
	if w.furniture != nil {
		// TODO(tmckee): need to ask the gui where the mouse is.
		// Typically, that means this stuff should be in Respond() instead of
		// Think(); Think() means handle end-of-frame but does not really specify a
		// mouse position because the position might have been changing during the
		// frame.
		// mx, my := gin.In().GetCursor("Mouse").Point()
		mx, my := 0, 0
		bx, by := w.RoomViewer.WindowToBoard(mx, my)
		f := w.furniture
		f.X = roundDown(bx - w.drag_anchor.x + 0.5)
		f.Y = roundDown(by - w.drag_anchor.y + 0.5)
		fdx, fdy := f.Dims()
		f.invalid = false
		if f.X < 0 {
			f.invalid = true
		}
		if f.Y < 0 {
			f.invalid = true
		}
		if f.X+fdx > w.Room.Size.Dx {
			f.invalid = true
		}
		if f.Y+fdy > w.Room.Size.Dy {
			f.invalid = true
		}
		for _, t := range w.Room.Furniture {
			if t == f {
				continue
			}
			tdx, tdy := t.Dims()
			r1 := image.Rect(t.X, t.Y, t.X+tdx, t.Y+tdy)
			r2 := image.Rect(f.X, f.Y, f.X+fdx, f.Y+fdy)
			if r1.Overlaps(r2) {
				f.invalid = true
			}
		}
	}

	w.VerticalTable.Think(ui, t)
	logging.Debug("FurniturePanel.Think", "room sizes", tags.RoomSizes, "selectedidx", w.room_size.GetComboedIndex())
	w.Room.Resize(tags.RoomSizes[w.room_size.GetComboedIndex()])
	w.Room.Name = w.name.GetText()
	w.Room.Floor.ResetPath(base.Path(w.floor_path.GetPath()))
	w.Room.Wall.ResetPath(base.Path(w.wall_path.GetPath()))
}
