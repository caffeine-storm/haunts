package game

import (
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"path/filepath"
)

type startLayout struct {
	Menu struct {
		X, Y     int
		Texture  texture.Object
		Credits  Button
		Versus   Button
		Online   Button
		Settings Button
	}
	Background texture.Object
}

type StartMenu struct {
	layout startLayout
	// TODO(tmckee): clean: I don't think we need this; we're told the region to
	// render into each frame, no?
	region  gui.Region
	buttons []ButtonLike
	mx, my  int
	last_t  int64
}

func InsertStartMenu(ui gui.WidgetParent) error {
	var sm StartMenu
	datadir := base.GetDataDir()
	err := base.LoadAndProcessObject(filepath.Join(datadir, "ui", "start", "layout.json"), "json", &sm.layout)
	if err != nil {
		return err
	}
	sm.buttons = []ButtonLike{
		&sm.layout.Menu.Credits,
		&sm.layout.Menu.Versus,
		&sm.layout.Menu.Online,
		&sm.layout.Menu.Settings,
	}
	sm.layout.Menu.Credits.f = func(interface{}) {
		ui.RemoveChild(&sm)
		err := InsertCreditsMenu(ui)
		if err != nil {
			base.DeprecatedError().Printf("Unable to make Credits Menu: %v", err)
			return
		}
	}
	sm.layout.Menu.Versus.f = func(interface{}) {
		ui.RemoveChild(&sm)
		err := InsertMapChooser(
			ui,
			func(name string) {
				// TODO(tmckee): why is this MakeGamePanel? Shouldn't it be
				// game.InsertVersusMenu?
				ui.AddChild(MakeGamePanel(name, nil, nil, ""))
			},
			InsertStartMenu,
		)
		if err != nil {
			base.DeprecatedError().Printf("Unable to make Map Chooser: %v", err)
			return
		}
	}
	sm.layout.Menu.Settings.f = func(interface{}) {}
	sm.layout.Menu.Online.f = func(interface{}) {
		ui.RemoveChild(&sm)
		err := InsertOnlineMenu(ui)
		if err != nil {
			base.DeprecatedError().Printf("Unable to make Online Menu: %v", err)
			return
		}
	}
	ui.AddChild(&sm)
	return nil
}

func (sm *StartMenu) Requested() gui.Dims {
	return gui.Dims{1024, 768}
}

func (sm *StartMenu) Expandable() (bool, bool) {
	return false, false
}

func (sm *StartMenu) Rendered() gui.Region {
	return sm.region
}

func (sm *StartMenu) Think(g *gui.Gui, t int64) {
	if sm.last_t == 0 {
		sm.last_t = t
		return
	}
	dt := t - sm.last_t
	sm.last_t = t

	if sm.mx == 0 && sm.my == 0 {
		// TODO(tmckee): need to ask the gui for a cursor pos
		// sm.mx, sm.my = gin.In().GetCursor("Mouse").Point()
		sm.mx, sm.my = 0, 0
	}
	for _, button := range sm.buttons {
		button.Think(sm.region.X, sm.region.Y, sm.mx, sm.my, dt)
	}
}

func (sm *StartMenu) SetOpacity(percent float64) {
	for _, buttonLike := range sm.buttons {
		if button, ok := buttonLike.(SetOpacityer); ok {
			button.SetOpacity(percent)
		}
	}
}

func (sm *StartMenu) Respond(g *gui.Gui, group gui.EventGroup) bool {
	cursor := group.Events[0].Key.Cursor()
	if cursor != nil {
		sm.mx, sm.my = cursor.Point()
	}

	if found, event := group.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
		hit := false
		for _, button := range sm.buttons {
			if button.handleClick(sm.mx, sm.my, nil) {
				hit = true
			}
		}
		if hit {
			return true
		}
	} else {
		hit := false
		for _, button := range sm.buttons {
			if button.Respond(group, nil) {
				hit = true
			}
		}
		if hit {
			return true
		}
	}
	return false
}

func (sm *StartMenu) Draw(region gui.Region, ctx gui.DrawingContext) {
	base.DeprecatedLog().Trace("StartMenu.Draw", "region", region)
	sm.region = region
	gl.Color4ub(255, 255, 255, 255)
	// TODO(tmckee): this is racy. .Data() lazy-loads the texture through the
	// texture manager but we immediately call RenderNatural. This is okayish IRL
	// because eventually there'll be a frame where things _have_ loaded; we'll
	// actually draw then. For testing, we can use texture.BlockUntilLoaded.
	sm.layout.Background.Data().RenderNatural(sm.region.X, sm.region.Y)
	sm.layout.Menu.Texture.Data().RenderNatural(sm.region.X+sm.layout.Menu.X, sm.region.Y+sm.layout.Menu.Y)
	base.DeprecatedLog().Trace("StartMenu.Draw: about to render buttons", "numbuttons", len(sm.buttons), "sm.layout", sm.layout)
	for _, button := range sm.buttons {
		// TODO(tmckee): clean: (x,y) given to RenderAt is not a target location
		// but an offset from the button's (X,Y) fields. This does not seem clear
		// to me.
		button.RenderAt(sm.region.X, sm.region.Y)
	}
}

func (sm *StartMenu) DrawFocused(region gui.Region, ctx gui.DrawingContext) {
	panic("NIY")
}

func (sm *StartMenu) String() string {
	return "start menu"
}
