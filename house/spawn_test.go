package house_test

import (
	"path"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/smartystreets/goconvey/convey"
)

type forwardToRenderOnFloor struct {
	it *house.SpawnPoint
}

func (self *forwardToRenderOnFloor) Draw(gui.Region, gui.DrawingContext) {
	self.it.RenderOnFloor()
}

func AsGametestDrawer(spawn *house.SpawnPoint) gametest.Drawer {
	return &forwardToRenderOnFloor{
		it: spawn,
	}
}

func GivenASpawnPoint() *house.SpawnPoint {
	path := path.Join(base.GetDataDir(), "textures/pentagram_04_large_red.png")
	return &house.SpawnPoint{
		Name: "spawn-for-test",
		Dx:   256,
		Dy:   256,
		X:    5,
		Y:    5,
		Tex: texture.Object{
			Path: base.Path(path),
		},
	}
}

func TestRenderSpawnPoint(t *testing.T) {
	// TODO(tmckee): support 'Render' inside gametest instead of with this
	// adapter thing.
	adapter := func() gametest.Drawer {
		return AsGametestDrawer(GivenASpawnPoint())
	}
	base.SetDatadir("../data")
	convey.Convey("spawn points should show up", t, func() {
		house.PushSpawnRegexp(".*")
		gametest.RunDrawingTest(adapter, "spawnpoint")
		house.PopSpawnRegexp()
	})
}
