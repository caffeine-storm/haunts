package game_test

import (
	"fmt"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAStartLayout() game.StartLayout {
	datadir := base.GetDataDir()

	ret, err := game.LoadStartLayoutFromDatadir(datadir)
	if err != nil {
		panic(fmt.Errorf("couldn't LoadStartLayoutFromDatadir: %w", err))
	}

	return *ret
}

func givenAStartMenu() *game.StartMenu {
	parent := &gui.StandardParent{}
	layout := givenAStartLayout()
	err := game.InsertStartMenu(parent, layout)
	if err != nil {
		panic(fmt.Errorf("couldn't insert start menu: %w", err))
	}
	return parent.GetChildren()[0].(*game.StartMenu)
}

type forwardingEventListener struct {
	gui     *gui.Gui
	wrapped gui.Widget
}

func (feh *forwardingEventListener) HandleEventGroup(group gin.EventGroup) {
	// Tell the wrapped Gui about the event group so it can prime its MouseState
	// cache.
	feh.gui.HandleEventGroup(group)

	gg := gui.EventGroup{
		EventGroup:                 group,
		DispatchedToFocussedWidget: false,
	}

	feh.wrapped.Respond(feh.gui, gg)
}

func (feh *forwardingEventListener) Think(int64) {
	// hurr-durr i am finking!
}

var _ gin.Listener = (*forwardingEventListener)(nil)

func getButtonLocation(menu *game.StartMenu, buttonLabel string) (int, int) {
	var it game.Button

	switch buttonLabel {
	case "Credits":
		it = menu.Layout.Menu.Credits
	case "Versus":
		it = menu.Layout.Menu.Versus
	case "Online":
		it = menu.Layout.Menu.Online
	case "Settings":
		it = menu.Layout.Menu.Settings
	default:
		panic(fmt.Errorf("bad label: %q", buttonLabel))
	}

	return it.X, it.Y
}

func RunStartupSpecs() {
	base.SetDatadir("../data")
	menu := givenAStartMenu()
	menu.SetOpacity(0.6)

	Convey("drawing the startup ui", func() {
		gametest.RunDrawingTest(menu, "startup", func(drawTestCtx gametest.DrawTestContext) {
			Convey("should let you click the menu", func() {
				flag := make(chan bool, 1)
				menu.PatchButtonForTest("Credits", func() {
					flag <- true
				})

				// TODO(tmckee): this sucks; we have to like, 'prime' the menu? It needs
				// a 'Think' with a non-zero horizon before it will try to propagate
				// Thinks to its children T_T
				menu.Think(nil, 12)

				window := drawTestCtx.NewWindow()
				driver := window.NewDriver()
				x, y := getButtonLocation(menu, "Credits")
				logging.Error("got button location", "x", x, "y", y)
				driver.Click(x, y)

				// Add a gin.EventHandler to the gin.Input object so that we can dispatch
				// input events to our 'gui' which is just the startmenu r.n.
				fel := &forwardingEventListener{
					gui:     guitest.MakeStubbedGui(window.GetDims()),
					wrapped: menu,
				}
				driver.AddInputListener(fel)

				// calls system.Think(); captures the next 'batch' of input events and
				// passess them to any "EventHandler" that got registered before this.
				// An example EventHandler is a gui.Gui which will consult its children
				// widgets to 'Respond' per-EventGroup, _then_ get an extra Think() call
				// at the end with a slice of all the groups that were just
				// attempt-to-respond-to'd.
				driver.ProcessFrame()

				So(<-flag, ShouldEqual, true)
			})
		})
	})
}

func TestDrawStartupUi(t *testing.T) {
	Convey("Startup UI", t, RunStartupSpecs)
}
