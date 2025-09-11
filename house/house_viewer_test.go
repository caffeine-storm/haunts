package house_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/logging"

	// TODO(tmckee): T_T T_T BLECH!! THIS IS SOOOO BAD!!!
	_ "github.com/MobRulesGames/haunts/game/actions"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

var _ gametest.Drawer = (*house.HouseViewer)(nil)

func givenAHouseViewer() *house.HouseViewer {
	ret := house.MakeHouseViewer(housetest.GivenAHouseDef(), 62)
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

		Convey("can draw houseviewer including drawables", func(c C) {
			registry.LoadAllRegistries()
			game.LoadAllEntities()

			Convey("use a stub drawable as a drawable", func(c C) {
				// TODO(tmckee): BLECH!
				var hv *house.HouseViewer
				logging.DebugBracket(func() {
					gametest.RunDrawingTest(c, func() gametest.Drawer {
						if hv == nil {
							restestHouseDef := house.MakeHouseFromName("restest")
							hv = house.MakeHouseViewer(restestHouseDef, 62)
							hv.AddDrawable(&housetest.StubDraw{
								X: 5, Y: 5,
								Dx: 10, Dy: 10,
							})
						}
						return hv
					}, "house-viewer-with-stubdraw")
				})
			})

			Convey("use an entity as a drawable", func(c C) {
				// TODO(tmckee): BLECH!
				var hv *house.HouseViewer
				logging.DebugBracket(func() {
					gametest.RunDrawingTest(c, func() gametest.Drawer {
						if hv == nil {
							restestHouseDef := house.MakeHouseFromName("restest")
							hv = house.MakeHouseViewer(restestHouseDef, 62)
							hv.Think(nil, 5)
							hv.SetZoom(100)
							hv.Focus(5, 5)
							hv.Think(nil, 5000)
							ent := gametest.GivenAnEntity()
							ent.X = 5
							ent.Y = 5
							stub := &housetest.StubDraw{
								X: 5, Y: 5,
								Dx: 10, Dy: 10,
							}
							hv.AddDrawable(stub)
							hv.AddDrawable(ent)
						}
						return hv
					}, "house-viewer-with-entity")
				})
			})
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
				houseViewer.Think(g, 18)
				guitest.SynthesizeEvents(houseViewer).WheelDown(-5)
				houseViewer.Think(g, 42000)
				return houseViewer
			}, "house-viewer-zoom-out")
		})

		Convey("Respond()ing to right-click drags will pan around", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				houseViewer := givenAHouseViewer()

				dimensions := gui.Dims{Dx: 64, Dy: 64}
				g := guitest.MakeStubbedGui(dimensions)

				fromPos := gui.Point{X: 32, Y: 32}
				toPos := fromPos
				toPos.X += 128
				toPos.Y -= 128

				rightButtonId := gin.AnyMouseRButton
				rightButtonId.Device.Index = 0

				houseViewer.Think(g, 18)
				guitest.SynthesizeEvents(houseViewer).DragGesture(rightButtonId, fromPos, toPos)

				// Need to simulate a few frames going by to give the house viewer a
				// chance to run its animations.
				houseViewer.Think(g, 42000)

				return houseViewer
			}, "house-viewer-panned")
		})
	})
}
