package gametest

import (
	"context"
	"path"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/glop/system/systemtest"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenADrawingContext(dims gui.Dims) gui.UpdateableDrawingContext {
	return guitest.MakeStubbedGui(dims)
}

type Drawer interface {
	Draw(gui.Region, gui.DrawingContext)
}

type DrawTestContext interface {
	NewWindow() systemtest.Window
}

type rendertestDrawTestContext struct {
	sys   system.System
	hdl   system.NativeWindowHandle
	queue render.RenderQueueInterface
}

var _ DrawTestContext = (*rendertestDrawTestContext)(nil)

func (ctx *rendertestDrawTestContext) NewWindow() systemtest.Window {
	return systemtest.NewTestWindow(ctx.sys, ctx.hdl, ctx.queue)

}

func RunDrawingTest(thingToDraw Drawer, testid rendertest.TestDataReference, andThen func(DrawTestContext)) {
	windowRegion := gui.Region{
		Point: gui.Point{X: 0, Y: 0},
		Dims:  gui.Dims{Dx: 1024, Dy: 750},
	}

	rendertest.WithGlAndHandleForTest(windowRegion.Dx, windowRegion.Dy, func(sys system.System, windowHandle system.NativeWindowHandle, queue render.RenderQueueInterface) {
		queue.Queue(func(st render.RenderQueueState) {
			globals.SetRenderQueueState(st)
		})
		queue.Purge()

		ctx := GivenADrawingContext(windowRegion.Dims)
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

		queue.Queue(func(st render.RenderQueueState) {
			thingToDraw.Draw(windowRegion, ctx)
		})
		queue.Purge()

		Convey("should look like the start screen", func() {
			So(queue, rendertest.ShouldLookLikeFile, testid, rendertest.Threshold(8))
		})

		andThen(&rendertestDrawTestContext{
			sys:   sys,
			hdl:   windowHandle,
			queue: queue,
		})
	})
}
