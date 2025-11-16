package game

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/hui"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/caffeine-storm/gl"
	"github.com/caffeine-storm/glop/gui"
)

type UiSelectMap struct {
	layout UiSelectMapLayout

	region gui.Region

	chooser *hui.RosterChooser
}

type MapOption struct {
	layout *UiSelectMapLayout

	house_def *house.HouseDef
}

func (mo *MapOption) Draw(hovered, selected, selectable bool, region gui.Region) {
	var s byte
	switch {
	case selected:
		s = 255
	case hovered && selectable:
		s = 205
	case selectable:
		s = 127
	default:
		s = 75
	}
	gl.Color4ub(s, s, s, 255)
	icon := mo.house_def.Icon.Data()
	if icon.Dx() == 0 {
		icon = mo.layout.Default_icon.Data()
	}
	icon.RenderNatural(region.X, region.Y)
	gl.Color4ub(0, 0, 0, 255)
	d := base.GetDictionary(15)
	shaderBank := globals.RenderQueueState().Shaders()
	d.RenderString(mo.house_def.Name, region.Point, d.MaxHeight(), gui.Left, shaderBank)
}

func (mo *MapOption) Think(dt int64) {
}

type UiSelectMapLayout struct {
	Default_icon texture.Object
}

func MakeUiSelectMap(gp *GamePanel) (gui.Widget, <-chan string, error) {
	var ui UiSelectMap

	datadir := base.GetDataDir()
	err := base.LoadAndProcessObject(filepath.Join(datadir, "ui", "select_map", "config.json"), "json", &ui.layout)
	if err != nil {
		return nil, nil, err
	}

	ui.region.Dx = 1024
	ui.region.Dy = 768
	var options []hui.Option
	// TODO: may want to reload the registry on this one?  If we want to pik up
	// new changes to files that is.
	for _, name := range base.GetAllNamesInRegistry("houses") {
		var mo MapOption
		mo.house_def = house.MakeHouseFromName(name)
		mo.layout = &ui.layout
		options = append(options, &mo)
	}
	out := make(chan string, 2)
	chooser := hui.MakeRosterChooser(options, hui.SelectExactlyOne, func(m map[int]bool) {
		var index int
		logging.Debug("on complete called", "selections", m)
		for index = range m {
			houseDefName := options[index].(*MapOption).house_def.Name
			out <- houseDefName
			logging.Debug("sent house name", "houseDefName", houseDefName)
			break
		}
		logging.Debug("closing")
		close(out)
	},
		nil)
	ui.chooser = chooser

	return &ui, out, nil
}

func (ui *UiSelectMap) String() string {
	return "ui select map"
}

func (ui *UiSelectMap) Requested() gui.Dims {
	return gui.Dims{Dx: 1024, Dy: 768}
}

func (ui *UiSelectMap) Expandable() (bool, bool) {
	return false, false
}

func (ui *UiSelectMap) Rendered() gui.Region {
	return ui.region
}

func (ui *UiSelectMap) Think(g *gui.Gui, dt int64) {
	ui.chooser.Think(g, dt)
}

func (ui *UiSelectMap) Respond(g *gui.Gui, group gui.EventGroup) bool {
	return ui.chooser.Respond(g, group)
}

func (ui *UiSelectMap) Draw(region gui.Region, ctx gui.DrawingContext) {
	ui.chooser.Draw(region, ctx)
}

func (ui *UiSelectMap) DrawFocused(region gui.Region, ctx gui.DrawingContext) {
	ui.chooser.Draw(region, ctx)
}
