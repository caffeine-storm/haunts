package game

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

type creditsLayout struct {
	Title struct {
		X, Y    int
		Texture texture.Object
	}
	Background texture.Object
	Credits    struct {
		Scroll ScrollingRegion
		Lines  []string
		Size   int
	}
	Back, Up, Down Button
}

type CreditsMenu struct {
	layout  creditsLayout
	region  gui.Region
	buttons []ButtonLike
	mx, my  int
	last_t  int64
	ui      gui.WidgetParent
}

func InsertCreditsMenu(ui gui.WidgetParent) error {
	var cm CreditsMenu
	datadir := base.GetDataDir()
	err := base.LoadAndProcessObject(filepath.Join(datadir, "ui", "start", "credits", "layout.json"), "json", &cm.layout)
	if err != nil {
		return err
	}
	cm.buttons = []ButtonLike{
		&cm.layout.Back,
		&cm.layout.Up,
		&cm.layout.Down,
	}
	cm.layout.Back.f = func(interface{}) {
		ui.RemoveChild(&cm)
		InsertStartMenu(ui)
	}
	d := base.GetDictionary(cm.layout.Credits.Size)
	cm.layout.Credits.Scroll.Height = len(cm.layout.Credits.Lines) * int(d.MaxHeight())
	cm.layout.Down.valid_func = func() bool {
		return cm.layout.Credits.Scroll.Height > cm.layout.Credits.Scroll.Dy
	}
	cm.layout.Up.valid_func = cm.layout.Down.valid_func
	cm.layout.Down.f = func(interface{}) {
		cm.layout.Credits.Scroll.Down()
	}
	cm.layout.Up.f = func(interface{}) {
		cm.layout.Credits.Scroll.Up()
	}
	cm.ui = ui

	ui.AddChild(&cm)
	return nil
}

func (cm *CreditsMenu) Requested() gui.Dims {
	return gui.Dims{1024, 768}
}

func (cm *CreditsMenu) Expandable() (bool, bool) {
	return false, false
}

func (cm *CreditsMenu) Rendered() gui.Region {
	return cm.region
}

func (cm *CreditsMenu) Think(g *gui.Gui, t int64) {
	if cm.last_t == 0 {
		cm.last_t = t
		return
	}
	dt := t - cm.last_t
	cm.last_t = t
	if cm.mx == 0 && cm.my == 0 {
		// TODO(tmckee): need to ask the gui for a cursor pos
		// cm.mx, cm.my = gin.In().GetCursor("Mouse").Point()
		cm.mx, cm.my = 0, 0
	}
	cm.layout.Credits.Scroll.Think(dt)
	for _, button := range cm.buttons {
		button.Think(cm.region.X, cm.region.Y, cm.mx, cm.my, dt)
	}
}

func (cm *CreditsMenu) Respond(g *gui.Gui, group gui.EventGroup) bool {
	cursor := group.Events[0].Key.Cursor()
	if cursor != nil {
		cm.mx, cm.my = cursor.Point()
	}
	if found, event := group.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
		for _, button := range cm.buttons {
			if button.handleClick(cm.mx, cm.my, nil) {
				return true
			}
		}
	}

	hit := false
	for _, button := range cm.buttons {
		if button.Respond(group, nil) {
			hit = true
		}
	}
	return hit
}

func (cm *CreditsMenu) Draw(region gui.Region, ctx gui.DrawingContext) {
	shaderBank := globals.RenderQueueState().Shaders()
	cm.region = region
	gl.Color4ub(255, 255, 255, 255)
	cm.layout.Background.Data().RenderNatural(region.X, region.Y)
	title := cm.layout.Title
	title.Texture.Data().RenderNatural(region.X+title.X, region.Y+title.Y)
	for _, button := range cm.buttons {
		button.RenderAt(cm.region.X, cm.region.Y)
	}

	d := base.GetDictionary(cm.layout.Credits.Size)
	sx := cm.layout.Credits.Scroll.X
	sy := cm.layout.Credits.Scroll.Top()
	cm.layout.Credits.Scroll.Region().PushClipPlanes()
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(255, 255, 255, 255)
	for _, line := range cm.layout.Credits.Lines {
		sy -= int(d.MaxHeight())
		d.RenderString(line, gui.Point{X: sx, Y: sy}, d.MaxHeight(), gui.Left, shaderBank)
	}
	cm.layout.Credits.Scroll.Region().PopClipPlanes()
}

func (cm *CreditsMenu) DrawFocused(region gui.Region, ctx gui.DrawingContext) {
}

func (cm *CreditsMenu) String() string {
	return "credits menu"
}
