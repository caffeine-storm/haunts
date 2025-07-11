package house_test

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenARoomDef() *house.RoomDef {
	smallSizeRoom := house.RoomSize{
		Name: "Small",
		Dx:   10,
		Dy:   10,
	}

	return &house.RoomDef{
		Size: smallSizeRoom,
	}
}

func GivenARoom(defname string) *house.Room {
	roomDef := GivenARoomDef()
	return &house.Room{
		Defname: defname,
		RoomDef: roomDef,
	}
}

func canSeeEverything() *house.LosTexture {
	queue := texture.GetRenderQueue()
	ret := house.MakeLosTexture()
	queue.Purge()

	ret.Clear(255)
	ret.Remap()
	queue.Purge()

	return ret
}

func loadRoom(roomName string, queue render.RenderQueueInterface) *house.Room {
	defname, _, _ := strings.Cut(roomName, ".room")
	output := GivenARoom(defname)
	err := base.LoadAndProcessObject(fmt.Sprintf("../data/rooms/%s", roomName), "json", output)
	if err != nil {
		panic(fmt.Errorf("failed to LoadAndProcessObject %q: %w", roomName, err))
	}

	queue.Queue(func(render.RenderQueueState) {
		// TODO(#12): having to remember to call some weird init function is
		// sad making.
		output.SetupGlStuff(&house.RoomRealGl{})
		output.SetWallTransparency(false)
	})
	queue.Purge()
	output.LoadAndWaitForTexturesForTest()

	return output
}

var transparent = color.RGBA{}
var black = color.RGBA{A: 255}

func TestRoom(t *testing.T) {
	Convey("house.Room", t, func(c C) {
		base.SetDatadir("../data")

		dx, dy := 1024, 768
		opaquealpha := byte(255)

		camera := housetest.Camera().ForSize(dx, dy).AtFocus(5, 5).AtZoom(50.0)

		testbuilder.WithSize(dx, dy, func(queue render.RenderQueueInterface) {
			c.Convey("--stubbed-context--", func() {
				registry.LoadAllRegistries()
				base.InitShaders(queue)
				texture.Init(queue)
				var losTexture *house.LosTexture = nil

				doRoomTest := func(roomid string) {
					fmt.Printf("TestRoom: losTexture: %v\n", losTexture)
					room := loadRoom(roomid+".room", queue)
					if room.Wall.GetPath() == "" {
						panic(fmt.Errorf("the '%s.room' file should have specified a texture for the walls", roomid))
					}
					allMats := housetest.MakeRoomMatsForCamera(room.Size, camera)

					noDrawables := []house.Drawable{}
					noFloorDrawers := []house.RenderOnFloorer{}
					queue.Queue(func(render.RenderQueueState) {
						room.Render(allMats, camera.Zoom, opaquealpha, noDrawables, losTexture, noFloorDrawers)
					})
					queue.Purge()

					So(queue, rendertest.ShouldLookLikeFile, roomid, rendertest.Threshold(13), rendertest.BackgroundColour(black))
				}

				Convey("loading from registry", func() {
					restestRoom := loadRoom("restest.room", queue)

					So(restestRoom, ShouldNotBeNil)
					So(restestRoom.Defname, ShouldEqual, "restest")
					So(restestRoom.Doors, ShouldHaveLength, 0)
				})

				Convey("drawing walls", func() {
					room := loadRoom("restest.room", queue)
					floor := housetest.MakeRoomMatsForCamera(room.Size, camera).Floor

					queue.Queue(func(render.RenderQueueState) {
						house.WithRoomRenderGlSettings(floor, func() {
							logging.Info("about to render decals", "floor", render.Showmat(floor))
							room.RenderDecals(&floor, opaquealpha)
						})
					})
					queue.Purge()

					So(queue, rendertest.ShouldLookLikeFile, "restest-walls", rendertest.BackgroundColour(transparent))
				})

				Convey("drawing restest", func() {
					doRoomTest("restest")
					Convey("with non-nil LosTexture", func() {
						losTexture = canSeeEverything()
						doRoomTest("restest")
					})
				})

				Convey("drawing tutorial-entry", func() {
					doRoomTest("tutorial-entry")
					Convey("with non-nil LosTexture", func() {
						losTexture = canSeeEverything()
						doRoomTest("tutorial-entry")
					})
				})
			})
		})
	})
}
