package house_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

var _ gametest.Drawer = (*house.HouseViewer)(nil)

func givenAHouseViewer() *house.HouseViewer {
	ret := house.MakeHouseViewer(housetest.GivenAHouseDef(), 62)
	ret.SetZoom(10)
	return ret
}

func centreOf(region gui.Dims) gui.Point {
	return gui.Point{
		X: region.Dx / 2,
		Y: region.Dy / 2,
	}
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

		Convey("Respond()ing to mouse wheel down zooms out", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				houseViewer := givenAHouseViewer()
				g := guitest.MakeStubbedGui(gui.Dims{Dx: 64, Dy: 64})
				wheelDown := guitest.SynthesizeEvents().WheelDown(-5)
				houseViewer.Respond(g, wheelDown)
				return houseViewer
			}, "house-viewer-zoom-out")
		})

		Convey("Respond()ing to right-click drags pans around", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				houseViewer := givenAHouseViewer()
				dimensions := gui.Dims{Dx: 64, Dy: 64}
				g := guitest.MakeStubbedGui(dimensions)

				fromPos := centreOf(dimensions)
				toPos := fromPos
				toPos.X += 2
				toPos.Y -= 2

				rightButtonId := gin.AnyMouseRButton
				rightButtonId.Device.Index = 0

				drag := guitest.SynthesizeEvents().DragGesture(rightButtonId, fromPos, toPos)
				for _, ev := range drag {
					houseViewer.Respond(g, ev)
				}
				return houseViewer
			}, "house-viewer-panned")
		})
	})
}
