package house_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

// TODO(tmckee): rename 'WallTexture' to 'Decal' or something.
func TestWallTextureSpecs(t *testing.T) {
	base.SetDatadir("../data")
	Convey("Wall Textures", t, func() {
		SkipConvey("can be made", func() {
			rendertest.WithGlForTest(200, 200, func(sys system.System, queue render.RenderQueueInterface) {
				fmt.Printf("the test is running!\n")
				datadir := base.GetDataDir()
				registry.LoadAllRegistries()
				base.InitShaders(queue)
				texture.Init(queue)
				wt := house.MakeWallTexture("Cobweb")
				So(wt, ShouldNotBeNil)
				doneLoadingTexture := make(chan bool, 1)
				queue.Queue(func(render.RenderQueueState) {
					texture.BlockUntilLoaded(path.Join(datadir, "textures", "cobweb.png"))
					doneLoadingTexture <- true
				})
				doneRendering := make(chan bool, 1)
				queue.Queue(func(render.RenderQueueState) {
					wt.Render()
					doneRendering <- true
				})

				fmt.Printf("going to wait for texture\n")
				<-doneLoadingTexture
				fmt.Printf("going to wait for render\n")
				<-doneRendering
				fmt.Printf("done waiting\n")

				So(queue, rendertest.ShouldLookLikeFile, "cobweb-wall-texture")
			})
		})
	})
}
