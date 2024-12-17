package house_test

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/glog"
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
			rendertest.WithGlForTest(266, 246, func(sys system.System, queue render.RenderQueueInterface) {
				logging.SetLogLevel(glog.LevelTrace)
				datadir := base.GetDataDir()
				registry.LoadAllRegistries()
				base.InitShaders(queue)
				texture.Init(queue)
				wt := house.MakeWallTexture("Cobweb")
				So(wt, ShouldNotBeNil)
				queue.Purge()

				texpath := path.Join(datadir, "textures", "cobweb.png")
				var err error

				queue.Queue(func(render.RenderQueueState) {
					_, err = texture.LoadFromPath(texpath)
				})
				logging.Debug("going to wait for texture")
				queue.Purge()
				So(err, ShouldBeNil)

				deadline, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
				defer cancel()
				err = texture.BlockUntilLoaded(deadline, texpath)
				So(err, ShouldBeNil)

				doneRendering := make(chan bool, 1)
				queue.Queue(func(render.RenderQueueState) {
					wt.Render()
					doneRendering <- true
				})

				So(err, ShouldBeNil)

				logging.Debug("going to wait for render")
				<-doneRendering
				logging.Debug("done waiting\n")

				So(queue, rendertest.ShouldLookLikeFile, "cobweb-wall-texture")
			})
		})
	})
}
