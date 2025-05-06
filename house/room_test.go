package house_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
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

func TestRoom(t *testing.T) {
	Convey("house.Room", t, RoomSpecs)
}

func loadRoom(roomName string) *house.Room {
	defname, _, _ := strings.Cut(roomName, ".room")
	output := GivenARoom(defname)
	err := base.LoadAndProcessObject(fmt.Sprintf("../data/rooms/%s", roomName), "json", output)
	if err != nil {
		panic(fmt.Errorf("failed to LoadAndProcessObject %q: %w", roomName, err))
	}
	return output
}

func RoomSpecs() {
	base.SetDatadir("../data")

	dx, dy := 1024, 768
	opaquealpha := byte(255)

	camera := housetest.Camera().ForSize(dx, dy).At(256, 256)

	rendertest.WithGlForTest(dx, dy, func(sys system.System, queue render.RenderQueueInterface) {
		registry.LoadAllRegistries()
		base.InitShaders(queue)
		texture.Init(queue)

		Convey("loading from registry", func() {
			restestRoom := loadRoom("restest.room")

			So(restestRoom, ShouldNotBeNil)
			So(restestRoom.Defname, ShouldEqual, "restest")
			So(restestRoom.Doors, ShouldHaveLength, 0)
		})

		Convey("drawing walls", func() {
			room := loadRoom("restest.room")
			floor := housetest.MakeRoomMatsForCamera(room.Size, camera).Floor

			queue.Queue(func(render.RenderQueueState) {
				// TODO(#12): having to remember to call some weird init function is
				// sad making.
				room.SetupGlStuff(&house.RoomRealGl{})
				room.SetWallTransparency(false)
			})
			queue.Purge()

			room.LoadAndWaitForTexturesForTest()

			queue.Queue(func(render.RenderQueueState) {
				house.WithRoomRenderGlSettings(floor, func() {
					room.RenderWallTextures(&floor, opaquealpha)
				})
			})
			queue.Purge()

			So(queue, rendertest.ShouldLookLikeFile, "restest-walls")
		})

		Convey("drawing restest", func() {
			restestRoom := loadRoom("restest.room")
			if restestRoom.Wall.GetPath() == "" {
				panic("the 'restest.room' file should have specified a texture for the walls")
			}
			allMats := housetest.MakeRoomMatsForCamera(restestRoom.Size, camera)

			queue.Queue(func(render.RenderQueueState) {
				restestRoom.SetupGlStuff(&house.RoomRealGl{})
				restestRoom.SetWallTransparency(false)
			})
			queue.Purge()

			restestRoom.LoadAndWaitForTexturesForTest()

			noDrawables := []house.Drawable{}
			var nilLos *house.LosTexture = nil
			noFloorDrawers := []house.RenderOnFloorer{}
			queue.Queue(func(render.RenderQueueState) {
				restestRoom.Render(allMats.Floor, allMats.Left, allMats.Right, camera.Zoom, opaquealpha, noDrawables, nilLos, noFloorDrawers)
			})
			queue.Purge()

			So(queue, rendertest.ShouldLookLikeFile, "restest", rendertest.Threshold(13))
		})

		Convey("drawing tutorial-entry", func() {
			tutRoom := loadRoom("tutorial-entry.room")
			if tutRoom.Wall.GetPath() == "" {
				panic("the 'tutorial-entry.room' file should have specified a texture for the walls")
			}
			allMats := housetest.MakeRoomMatsForCamera(tutRoom.Size, camera)

			queue.Queue(func(render.RenderQueueState) {
				tutRoom.SetupGlStuff(&house.RoomRealGl{})
				tutRoom.SetWallTransparency(false)
			})
			queue.Purge()

			tutRoom.LoadAndWaitForTexturesForTest()

			noDrawables := []house.Drawable{}
			var nilLos *house.LosTexture = nil
			noFloorDrawers := []house.RenderOnFloorer{}
			queue.Queue(func(render.RenderQueueState) {
				tutRoom.Render(allMats.Floor, allMats.Left, allMats.Right, camera.Zoom, opaquealpha, noDrawables, nilLos, noFloorDrawers)
			})
			queue.Purge()

			So(queue, rendertest.ShouldLookLikeFile, "tutorial-entry", rendertest.Threshold(13))
		})
	})
}
