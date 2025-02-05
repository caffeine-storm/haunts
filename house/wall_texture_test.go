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
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/glog"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

func getGlTexture(wt *house.WallTexture) gl.Texture {
	return wt.Texture.Data().GetGlTexture()
}

// TODO(tmckee): rename 'WallTexture' to 'Decal' or something.
func TestWallTextureSpecs(t *testing.T) {
	base.SetDatadir("../data")
	rendertest.WithGlForTest(266, 246, func(sys system.System, queue render.RenderQueueInterface) {
		Convey("Wall Textures", t, func() {
			Convey("can be made", func() {
				logging.SetLogLevel(glog.LevelTrace)
				datadir := base.GetDataDir()
				registry.LoadAllRegistries()
				base.InitShaders(queue)
				texture.Init(queue)
				wt := house.LoadWallTexture("Cobweb")
				So(wt, ShouldNotBeNil)

				Convey("texture loads successfully", func() {
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

					So(wt.Texture.Data().Dx(), ShouldBeGreaterThan, 0)
					So(wt.Texture.Data().Dy(), ShouldBeGreaterThan, 0)

					Convey("can render", func() {
						queue.Queue(func(render.RenderQueueState) {
							wt.Render()
						})
						queue.Purge()

						So(queue, rendertest.ShouldLookLikeFile, "cobweb-rendered")
					})
				})
			})
		})
	})
}
