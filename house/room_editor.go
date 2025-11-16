package house

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/caffeine-storm/glop/gui"
)

type RoomEditorPanel struct {
	*gui.HorizontalTable
	tab     *gui.TabFrame
	widgets []tabWidget

	panels struct {
		furniture *FurniturePanel
		wall      *WallPanel
	}

	room   Room
	viewer *roomViewer
}

// Manually pass all events to the tabs, regardless of location, since the tabs
// need to know where the user clicks.
func (w *RoomEditorPanel) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	w.viewer.Respond(ui, group)
	return w.widgets[w.tab.SelectedTab()].Respond(ui, group)
}

func (w *RoomEditorPanel) SelectTab(n int) {
	if n < 0 || n >= len(w.widgets) {
		return
	}
	if n != w.tab.SelectedTab() {
		w.widgets[w.tab.SelectedTab()].Collapse()
		w.tab.SelectTab(n)
		w.viewer.SetEditMode(editNothing)
		w.widgets[n].Expand()
	}
}

func MakeRoomEditorPanel() Editor {
	var rep RoomEditorPanel

	rep.HorizontalTable = gui.MakeHorizontalTable()
	rep.room.RoomDef = new(RoomDef)
	rep.viewer = MakeRoomViewer(&rep.room, 65)
	rep.AddChild(rep.viewer)

	var tabs []gui.Widget

	rep.panels.furniture = makeFurniturePanel(&rep.room, rep.viewer)
	tabs = append(tabs, rep.panels.furniture)
	rep.widgets = append(rep.widgets, rep.panels.furniture)

	rep.panels.wall = MakeWallPanel(&rep.room, rep.viewer)
	tabs = append(tabs, rep.panels.wall)
	rep.widgets = append(rep.widgets, rep.panels.wall)

	rep.tab = gui.MakeTabFrame(tabs)
	rep.AddChild(rep.tab)
	rep.viewer.SetEditMode(editFurniture)

	return &rep
}

func (rep *RoomEditorPanel) Load(path string) error {
	var room Room
	err := base.LoadAndProcessObject(path, "json", &room.RoomDef)
	if err == nil {
		rep.room = room
		for _, tab := range rep.widgets {
			tab.Reload()
		}
	}
	return err
}

func (rep *RoomEditorPanel) Save() (string, error) {
	path := filepath.Join(datadir, "rooms", rep.room.Name+".room")
	err := base.SaveJson(path, rep.room)
	return path, err
}

func (rep *RoomEditorPanel) Reload() {
	for _, tab := range rep.widgets {
		tab.Reload()
	}
}

func (w *RoomEditorPanel) GetViewer() Viewer {
	return w.viewer
}
