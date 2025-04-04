package game_test

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/glop/system/systemtest"
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

func givenADrawingContext(dims gui.Dims) gui.UpdateableDrawingContext {
	return guitest.MakeStubbedGui(dims)
}

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
	windowRegion := gui.Region{
		Point: gui.Point{X: 0, Y: 0},
		Dims:  gui.Dims{Dx: 1024, Dy: 750},
	}
	menu := givenAStartMenu()

	rendertest.WithGlAndHandleForTest(windowRegion.Dx, windowRegion.Dy, func(sys system.System, windowHandle system.NativeWindowHandle, queue render.RenderQueueInterface) {
		queue.Queue(func(st render.RenderQueueState) {
			globals.SetRenderQueueState(st)
		})
		queue.Purge()

		ctx := givenADrawingContext(windowRegion.Dims)
		base.InitDictionaries(ctx)
		texture.Init(queue)

		startTexture := path.Join(base.GetDataDir(), "ui", "start", "start.png")
		menuTexture := path.Join(base.GetDataDir(), "ui", "start", "menu.png")

		var err error
		_, err = texture.LoadFromPath(startTexture)
		if err != nil {
			panic(err)
		}
		_, err = texture.LoadFromPath(menuTexture)
		if err != nil {
			panic(err)
		}

		deadlineContext, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
		defer cancel()
		err = texture.BlockUntilLoaded(deadlineContext, startTexture, menuTexture)
		So(err, ShouldBeNil)

		menu.SetOpacity(0.6)

		queue.Queue(func(st render.RenderQueueState) {
			menu.Draw(windowRegion, ctx)
		})
		queue.Purge()

		Convey("should look like the start screen", func() {
			So(queue, rendertest.ShouldLookLikeFile, "startup", rendertest.Threshold(8))
		})

		Convey("should let you click the menu", func() {
			flag := false
			menu.PatchButtonForTest("Credits", func() {
				flag = true
			})

			// TODO(tmckee): this sucks; we have to like, 'prime' the menu? It needs
			// a 'Think' with a non-zero horizon before it will try to propagate
			// Thinks to its children T_T
			menu.Think(nil, 12)

			window := systemtest.NewTestWindow(sys, windowHandle, queue)
			driver := window.NewDriver()
			x, y := getButtonLocation(menu, "Credits")
			logging.Error("got button location", "x", x, "y", y)
			driver.Click(x, y)

			// Add a gin.EventHandler to the gin.Input object so that we can dispatch
			// input events to our 'gui' which is just the startmenu r.n.
			fel := &forwardingEventListener{
				gui:     guitest.MakeStubbedGui(windowRegion.Dims),
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

			So(flag, ShouldEqual, true)
		})
	})
}

func TestDrawStartupUi(t *testing.T) {
	Convey("Startup UI", t, RunStartupSpecs)
}
