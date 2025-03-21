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
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAStartMenu() *game.StartMenu {
	parent := &gui.StandardParent{}
	err := game.InsertStartMenu(parent)
	if err != nil {
		panic(fmt.Errorf("couldn't insert start menu: %w", err))
	}
	return parent.GetChildren()[0].(*game.StartMenu)
}

func givenADrawingContext(dims gui.Dims) gui.UpdateableDrawingContext {
	return guitest.MakeStubbedGui(dims)
}

func RunStartupSpecs() {
	base.SetDatadir("../data")
	windowRegion := gui.Region{
		Point: gui.Point{X: 0, Y: 0},
		Dims:  gui.Dims{Dx: 1024, Dy: 750},
	}
	menu := givenAStartMenu()

	rendertest.WithGlForTest(windowRegion.Dx, windowRegion.Dy, func(sys system.System, queue render.RenderQueueInterface) {
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

		So(queue, rendertest.ShouldLookLikeFile, "startup", rendertest.Threshold(8))
	})
}

func TestDrawStartupUi(t *testing.T) {
	Convey("Startup UI", t, RunStartupSpecs)
}
