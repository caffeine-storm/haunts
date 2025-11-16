package gametest

import (
	"image/color"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/caffeine-storm/glop/gui"
	"github.com/caffeine-storm/glop/gui/guitest"
	"github.com/caffeine-storm/glop/render"
	"github.com/caffeine-storm/glop/render/rendertest"
	"github.com/caffeine-storm/glop/system/systemtest"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenADrawingContext(dims gui.Dims) gui.UpdateableDrawingContext {
	return guitest.MakeStubbedGui(dims)
}

type Drawer interface {
	Draw(gui.Region, gui.DrawingContext)
}

type DrawTestContext interface {
	GetTestWindow() systemtest.Window
}

type rendertestDrawTestContext struct {
	testWindow systemtest.Window
}

func (ctx *rendertestDrawTestContext) GetTestWindow() systemtest.Window {
	return ctx.testWindow
}

var _ DrawTestContext = (*rendertestDrawTestContext)(nil)

func RunDrawingTest(c C, objectCreator func() Drawer, testid rendertest.TestDataReference, andThen ...func(DrawTestContext)) {
	RunDrawingTestWithUiDriver(c, objectCreator, func(systemtest.Driver) {}, testid, andThen...)
}

func RunDrawingTestWithUiDriver(c C, objectCreator func() Drawer, driveFunc func(systemtest.Driver), testid rendertest.TestDataReference, andThen ...func(DrawTestContext)) {
	windowRegion := gui.Region{
		Point: gui.Point{X: 0, Y: 0},
		Dims:  gui.Dims{Dx: 1024, Dy: 750},
	}

	systemtest.WithTestWindow(windowRegion.Dx, windowRegion.Dy, func(w systemtest.Window) {
		queue := w.GetQueue()
		globals.SetRenderQueue(queue)
		queue.Queue(func(st render.RenderQueueState) {
			globals.SetRenderQueueState(st)
		})
		queue.Purge()

		ctx := GivenADrawingContext(windowRegion.Dims)
		registry.LoadAllRegistries()
		base.InitDictionaries(ctx)
		texture.Init(queue)
		base.InitShaders(queue)

		thingToDraw := objectCreator()

		// Draw it once to start loading textures.
		queue.Queue(func(st render.RenderQueueState) {
			thingToDraw.Draw(windowRegion, ctx)
		})
		queue.Purge()

		// TODO(#20): this should not be allowed to take more than a frame or two
		// T_T
		err := texture.BlockWithTimeboxUntilIdle(time.Millisecond * 25000)
		c.So(err, ShouldBeNil)

		driveFunc(w.NewDriver())

		// Draw it again now that we know all the textures are loaded.
		queue.Queue(func(st render.RenderQueueState) {
			// First, blank the screen, though, because all UI expects to get a black
			// background to begin with.
			rendertest.ClearScreen()
			thingToDraw.Draw(windowRegion, ctx)
		})
		queue.Purge()

		transparent := color.RGBA{}
		c.Convey("should look like the expected screen", func() {
			So(queue, rendertest.ShouldLookLikeFile, testid, rendertest.Threshold(8), rendertest.BackgroundColour(transparent))

			for _, each := range andThen {
				each(&rendertestDrawTestContext{
					testWindow: w,
				})
			}
		})
	})
}
