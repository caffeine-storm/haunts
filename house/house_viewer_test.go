package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	. "github.com/smartystreets/goconvey/convey"
)

var _ gametest.Drawer = (*house.HouseViewer)(nil)

func givenAHouseViewer() gametest.Drawer {
	ret := house.MakeHouseViewer(housetest.GivenAHouseDef(), 62)
	ret.Zoom(10)
	return ret
}

func TestHouseViewer(t *testing.T) {
	Convey("HouseViewer", t, func() {
		base.SetDatadir("../data")

		Convey("can draw houseviewer", func() {
			gametest.RunDrawingTest(givenAHouseViewer, "house-viewer")
		})
	})
}
