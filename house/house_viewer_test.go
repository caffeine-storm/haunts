package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/render"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAHouseDef() *house.HouseDef {
	return house.MakeHouseFromName("tutorial")
}

var _ gametest.Drawer = (*house.HouseViewer)(nil)

func givenAHouseViewer(queue render.RenderQueueInterface) gametest.Drawer {
	return house.MakeHouseViewer(givenAHouseDef(), 62)
}

func TestHouseViewer(t *testing.T) {
	// TODO(#10): skipped for now while we make rooms render correctly
	SkipConvey("HouseViewer", t, func() {
		base.SetDatadir("../data")

		Convey("can draw houseviewer", func() {
			logging.DebugBracket(func() {
				gametest.RunOtherDrawingTest(givenAHouseViewer, "house-viewer", func(gametest.DrawTestContext) {})
			})
		})

	})
}
