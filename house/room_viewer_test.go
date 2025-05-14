package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRoomViewer(t *testing.T) {
	base.SetDatadir("../data")

	screenRegion := gui.Region{
		Point: gui.Point{
			X: 0, Y: 0,
		},
		Dims: gui.Dims{
			Dx: 256, Dy: 256,
		},
	}
	Convey("house.roomViewer", t, func() {
		rendertest.DeprecatedWithGlForTest(screenRegion.Dx, screenRegion.Dy, func(sys system.System, queue render.RenderQueueInterface) {
			registry.LoadAllRegistries()
			base.InitShaders(queue)
			texture.Init(queue)
			room := loadRoom("restest.room")

			Convey("can be made", func() {
				rv := house.MakeRoomViewer(room, 0)
				So(rv, ShouldNotBeNil)

				// TODO(#10): don't skip, fix!
				SkipConvey("can be drawn", func() {
					g := guitest.MakeStubbedGui(screenRegion.Dims)
					g.AddChild(rv)
					queue.Queue(func(render.RenderQueueState) {
						logging.TraceBracket(func() {
							g.Draw()
						})
					})
					queue.Purge()

					So(queue, rendertest.ShouldLookLikeFile, "room-viewer", rendertest.Threshold(0))
				})
			})
		})
	})
}
