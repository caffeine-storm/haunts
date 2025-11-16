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
	"github.com/caffeine-storm/glop/render"
	"github.com/caffeine-storm/glop/render/rendertest"
	"github.com/caffeine-storm/glop/render/rendertest/testbuilder"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetAllDecalNames(t *testing.T) {
	base.SetDatadir("../data")
	registry.LoadAllRegistries()

	names := house.GetAllDecalNames()
	if len(names) < 12 {
		t.Fatalf("expecting more decal names; got %d", len(names))
	}
}

func TestDecalSpecs(t *testing.T) {
	base.SetDatadir("../data")
	testbuilder.WithSize(266, 246, func(queue render.RenderQueueInterface) {
		Convey("Decals", t, func() {
			Convey("can be made", func() {
				datadir := base.GetDataDir()
				registry.LoadAllRegistries()
				base.InitShaders(queue)
				texture.Init(queue)
				decal := house.LoadDecal("Cobweb")
				So(decal, ShouldNotBeNil)

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

					So(decal.Texture.Data().Dx(), ShouldBeGreaterThan, 0)
					So(decal.Texture.Data().Dy(), ShouldBeGreaterThan, 0)

					Convey("can render", func() {
						queue.Queue(func(render.RenderQueueState) {
							decal.Render()
						})
						queue.Purge()

						So(queue, rendertest.ShouldLookLikeFile, "cobweb-rendered", rendertest.Threshold(6))
					})
				})
			})
		})
	})
}
