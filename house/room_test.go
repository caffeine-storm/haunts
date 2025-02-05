package house_test

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenARoomDef() *house.RoomDef {
	return &house.RoomDef{}
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
	room := GivenARoom("restest")
	logging.SetLoggingLevel(slog.LevelDebug)

	Convey("construction succeeds", func() {
		So(room, ShouldNotBeNil)
	})

	rendertest.WithGlForTest(200, 200, func(sys system.System, queue render.RenderQueueInterface) {
		registry.LoadAllRegistries()
		base.InitShaders(queue)
		texture.Init(queue)

		Convey("loading from registry", func() {
			restestRoom := loadRoom("restest.room")

			So(restestRoom, ShouldNotBeNil)
			So(restestRoom.Defname, ShouldEqual, "restest")
			So(restestRoom.Doors, ShouldHaveLength, 0)
		})

		SkipConvey("drawing restest", func() {
			restestRoom := loadRoom("restest.room")

			id := mathgl.Mat4{}
			id.Identity()

			nozoom := float32(1.0)
			opaquealpha := byte(255)
			noDrawables := []house.Drawable{}
			var nilLos *house.LosTexture = nil
			noFloorDrawers := []house.RenderOnFloorer{}
			queue.Queue(func(render.RenderQueueState) {
				restestRoom.Render(id, id, id, nozoom, opaquealpha, noDrawables, nilLos, noFloorDrawers)
			})
			queue.Purge()

			fmt.Printf("room: %+v\n", restestRoom)
			fmt.Printf("roomDef: %+v\n", restestRoom.RoomDef)

			So(queue, rendertest.ShouldLookLikeFile, "restest")
		})
	})
}
