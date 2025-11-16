package house

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/caffeine-storm/glop/gui"
)

type editMode int

const (
	editNothing editMode = iota
	editFurniture
	editDecals
	editCells
)

type Editor interface {
	gui.Widget

	Save() (string, error)
	Load(path string) error

	// Called when we tab into the editor from another editor.  It's possible that
	// a portion of what is being edited in the new editor was changed in another
	// editor, so we reload everything so we can see the up-to-date version.
	Reload()

	GetViewer() Viewer

	// TODO: Deprecate when tabs handle the switching themselves
	SelectTab(int)
}

func MakeHouseEditorPanel() Editor {
	var he HouseEditor
	he.house = *MakeHouseDef()
	he.HorizontalTable = gui.MakeHorizontalTable()
	he.viewer = MakeHouseViewer(&he.house, 62)
	he.viewer.Edit_mode = true
	he.HorizontalTable.AddChild(he.viewer)

	he.widgets = append(he.widgets, makeHouseDataTab(&he.house, he.viewer))
	he.widgets = append(he.widgets, makeHouseDoorTab(&he.house, he.viewer))
	he.widgets = append(he.widgets, makeHouseRelicsTab(&he.house, he.viewer))
	var tabs []gui.Widget
	for _, w := range he.widgets {
		tabs = append(tabs, w.(gui.Widget))
	}
	he.tab = gui.MakeTabFrame(tabs)
	he.HorizontalTable.AddChild(he.tab)

	return &he
}

// Manually pass all events to the tabs, regardless of location, since the tabs
// need to know where the user clicks.
func (he *HouseEditor) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	he.viewer.Respond(ui, group)
	return he.widgets[he.tab.SelectedTab()].Respond(ui, group)
}

func (he *HouseEditor) Load(path string) error {
	house, err := MakeHouseFromPath(path)
	if err != nil {
		return err
	}
	base.DeprecatedLog().Info("Load success", "path", path)
	house.Normalize()
	he.house = *house
	he.viewer.SetBounds()
	for _, tab := range he.widgets {
		tab.Reload()
	}
	return err
}

func (he *HouseEditor) Save() (string, error) {
	path := filepath.Join(datadir, "houses", he.house.Name+".house")
	err := base.SaveJson(path, he.house)
	return path, err
}

func (he *HouseEditor) Reload() {
	for _, floor := range he.house.Floors {
		for i := range floor.Rooms {
			base.GetObject("rooms", floor.Rooms[i])
		}
	}
	for _, tab := range he.widgets {
		tab.Reload()
	}
}
