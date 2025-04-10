package game_test

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"

	. "github.com/smartystreets/goconvey/convey"
)

func givenAnOnlineMenuLayout() *game.OnlineLayout {
	ret := game.OnlineLayout{}

	err := base.LoadAndProcessObject(filepath.Join(base.GetDataDir(), "ui", "start", "online", "layout.json"), "json", &ret)
	if err != nil {
		panic(fmt.Errorf("couldn't load layout: %w", err))
	}

	return &ret
}

func givenAnOnlineMenu() *game.OnlineMenu {
	parent := &gui.StandardParent{}
	// layout := givenAnOnlineMenuLayout()
	err := game.InsertOnlineMenu(parent)
	if err != nil {
		panic(fmt.Errorf("couldn't insert start menu: %w", err))
	}

	return parent.GetChildren()[0].(*game.OnlineMenu)
}

func TestUiOnline(t *testing.T) {
	Convey("UI for starting an online game", t, func() {
		base.SetDatadir("../data")

		// TODO(tmckee): this is copy-pasta from ui_start_test.go; we ought to have a
		// generic-enough "can draw this bit of UI" test harness.
		Convey("can insert to a widget parent", func() {
			windowRegion := gui.Region{
				Point: gui.Point{X: 0, Y: 0},
				Dims:  gui.Dims{Dx: 1024, Dy: 750},
			}
			onlineScreen := givenAnOnlineMenu()

			rendertest.WithGlAndHandleForTest(windowRegion.Dx, windowRegion.Dy, func(sys system.System, windowHandle system.NativeWindowHandle, queue render.RenderQueueInterface) {
				queue.Queue(func(st render.RenderQueueState) {
					globals.SetRenderQueueState(st)
				})
				queue.Purge()

				ctx := gametest.GivenADrawingContext(windowRegion.Dims)
				base.InitDictionaries(ctx)
				texture.Init(queue)

				datadir := base.GetDataDir()
				texturePaths := []string{
					"ui/arrow_lf.png",
					"ui/dialog/large.png",
					"ui/start/online/title.png",
					"ui/arrow_down.png",
					"ui/arrow_up.png",
				}
				for i, p := range texturePaths {
					texturePaths[i] = path.Join(datadir, p)
				}
				for _, p := range texturePaths {
					_, e := texture.LoadFromPath(p)
					if e != nil {
						panic(fmt.Errorf("couldn't LoadFromPath: %w", e))
					}
				}

				deadlineContext, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
				defer cancel()
				err := texture.BlockUntilLoaded(deadlineContext, texturePaths...)
				So(err, ShouldBeNil)

				queue.Queue(func(st render.RenderQueueState) {
					onlineScreen.Draw(windowRegion, ctx)
				})
				queue.Purge()

				Convey("should look like the Online Menu", func() {
					So(queue, rendertest.ShouldLookLikeFile, "online", rendertest.Threshold(8))
				})
			})
		})
	})
}
