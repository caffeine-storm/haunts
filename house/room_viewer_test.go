package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/house"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRoomViewer(t *testing.T) {
	Convey("house.roomViewer", t, RoomViewerSpecs)
}

func RoomViewerSpecs() {
	room := GivenARoom()

	Convey("can be made", func() {
		rv := house.MakeRoomViewer(room, 0)
		So(rv, ShouldNotBeNil)
	})
}
