package game

import (
	"fmt"
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

type MenuConfig struct {
	X, Y     int
	Texture  texture.Object
	Credits  Button
	Versus   Button
	Online   Button
	Settings Button
}

type StartLayout struct {
	Menu       MenuConfig
	Background texture.Object
}

func LoadStartLayoutFromDatadir(datadir string) (*StartLayout, error) {
	ret := &StartLayout{}

	err := base.LoadAndProcessObject(filepath.Join(datadir, "ui", "start", "layout.json"), "json", ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

type StartMenu struct {
	Layout StartLayout
	// TODO(tmckee): clean: I don't think we need this; we're told the region to
	// render into each frame, no?
	region  gui.Region
	buttons []ButtonLike
	mx, my  int
	last_t  int64
}

func (sm *StartMenu) PatchButtonForTest(buttonKey string, fn func()) {
	var btn *Button = nil
	switch buttonKey {
	case "Credits":
		btn = &sm.Layout.Menu.Credits
	case "Versus":
		btn = &sm.Layout.Menu.Versus
	case "Online":
		btn = &sm.Layout.Menu.Online
	case "Settings":
		btn = &sm.Layout.Menu.Settings
	default:
		panic(fmt.Errorf("unknown button key %q", buttonKey))
	}

	oldF := btn.f

	btn.f = func(arg any) {
		fn()
		oldF(arg)
	}
}

func InsertStartMenu(ui gui.WidgetParent, layout StartLayout) error {
	var sm StartMenu
	sm.Layout = layout
	sm.buttons = []ButtonLike{
		&sm.Layout.Menu.Credits,
		&sm.Layout.Menu.Versus,
		&sm.Layout.Menu.Online,
		&sm.Layout.Menu.Settings,
	}
	sm.Layout.Menu.Credits.f = func(interface{}) {
		logging.Trace("sm.Layout.Menu.f called")
		ui.RemoveChild(&sm)
		err := InsertCreditsMenu(ui)
		if err != nil {
			logging.Error("Unable to make Credits Menu", "err", err)
			return
		}
	}
	sm.Layout.Menu.Versus.f = func(interface{}) {
		ui.RemoveChild(&sm)
		err := InsertMapChooser(
			ui,
			func(name string) {
				logging.Info("MenuVersus buttonf", "'name'", name)
				ui.AddChild(MakeGamePanel(name, nil, nil, ""))
			},
			func(parent gui.WidgetParent) error {
				return InsertStartMenu(parent, sm.Layout)
			},
		)
		if err != nil {
			logging.Error("Unable to make Map Chooser", "err", err)
			return
		}
	}
	sm.Layout.Menu.Settings.f = func(interface{}) {}
	sm.Layout.Menu.Online.f = func(interface{}) {
		ui.RemoveChild(&sm)
		err := InsertOnlineMenu(ui)
		if err != nil {
			logging.Error("Unable to make Online Menu", "err", err)
			return
		}
	}
	ui.AddChild(&sm)
	return nil
}

func (sm *StartMenu) Requested() gui.Dims {
	return gui.Dims{Dx: 1024, Dy: 768}
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
	logging.Trace("StartMenu.Respond called", "events", group)

	if mpos, ok := g.UseMousePosition(group); ok {
		sm.mx, sm.my = mpos.X, mpos.Y
	}

	if group.IsPressed(gin.AnyMouseLButton) {
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
	logging.Trace("StartMenu.Draw", "region", region)
	sm.region = region
	gl.Color4ub(255, 255, 255, 255)
	// TODO(tmckee): this is racy. .Data() lazy-loads the texture through the
	// texture manager but we immediately call RenderNatural. This is okayish IRL
	// because eventually there'll be a frame where things _have_ loaded; we'll
	// actually draw then. For testing, we can use texture.BlockUntilLoaded.
	sm.Layout.Background.Data().RenderNatural(sm.region.X, sm.region.Y)
	sm.Layout.Menu.Texture.Data().RenderNatural(sm.region.X+sm.Layout.Menu.X, sm.region.Y+sm.Layout.Menu.Y)
	logging.Trace("StartMenu.Draw: about to render buttons", "numbuttons", len(sm.buttons), "sm.layout", sm.Layout)
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
