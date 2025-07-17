package house_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

var _ gametest.Drawer = (*house.HouseViewer)(nil)

func givenAHouseViewer() *house.HouseViewer {
	ret := house.MakeHouseViewer(housetest.GivenAHouseDef(), 62)
	ret.SetZoom(10)
	return ret
}

func TestHouseViewer(t *testing.T) {
	Convey("HouseViewer", t, func() {
		base.SetDatadir("../data")

		Convey("can draw houseviewer", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				return givenAHouseViewer()
			}, "house-viewer")
		})

		Convey("can zoom in on houseviewer", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				ret := givenAHouseViewer()
				ret.SetZoom(ret.GetZoom() * 2)
				return ret
			}, "house-viewer-zoom-in")
		})

		Convey("can zoom out on houseviewer", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				ret := givenAHouseViewer()
				ret.SetZoom(ret.GetZoom() / 2)
				return ret
			}, "house-viewer-zoom-out")
		})

		Convey("has a useful stringification", func() {
			houseViewer := givenAHouseViewer()
			asString := fmt.Sprintf("%v", houseViewer)
			asLower := strings.ToLower(asString)

			// Every house should have a set of 'floors'.
			assert.Contains(t, asLower, "floors")
		})
	})
}
