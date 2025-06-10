package gametest

import (
	"context"
	"image/color"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenADrawingContext(dims gui.Dims) gui.UpdateableDrawingContext {
	return guitest.MakeStubbedGui(dims)
}

type Drawer interface {
	Draw(gui.Region, gui.DrawingContext)
}

type DrawTestContext interface {
	GetTestBuilder() *testbuilder.GlTestBuilder
}

type rendertestDrawTestContext struct {
	testBuilder *testbuilder.GlTestBuilder
}

func (ctx *rendertestDrawTestContext) GetTestBuilder() *testbuilder.GlTestBuilder {
	return ctx.testBuilder
}

var _ DrawTestContext = (*rendertestDrawTestContext)(nil)

// TODO(tmckee:#31): take a convey.C as a parameter so that callers know they need
// to be running this under a convey context.
func RunDrawingTest(objectCreator func() Drawer, testid rendertest.TestDataReference, andThen ...func(DrawTestContext)) {
	windowRegion := gui.Region{
		Point: gui.Point{X: 0, Y: 0},
		Dims:  gui.Dims{Dx: 1024, Dy: 750},
	}

	testBuilder := testbuilder.New().WithSize(windowRegion.Dx, windowRegion.Dy)
	testBuilder.WithQueue().Run(func(queue render.RenderQueueInterface) {
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
		deadlineContext, cancel := context.WithTimeout(context.Background(), time.Millisecond*25000)
		defer cancel()
		err := texture.BlockUntilIdle(deadlineContext)
		So(err, ShouldBeNil)

		// Draw it again now that we know all the textures are loaded.
		queue.Queue(func(st render.RenderQueueState) {
			// First, blank the screen, though, because all UI expects to get a black
			// background to begin with.
			rendertest.ClearScreen()
			thingToDraw.Draw(windowRegion, ctx)
		})
		queue.Purge()

		transparent := color.RGBA{}
		Convey("should look like the expected screen", func() {
			So(queue, rendertest.ShouldLookLikeFile, testid, rendertest.Threshold(8), rendertest.BackgroundColour(transparent))
		})

	})
	for _, each := range andThen {
		each(&rendertestDrawTestContext{
			testBuilder: testBuilder,
		})
	}
}
