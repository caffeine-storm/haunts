package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/house"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenARoomDef() *house.RoomDef {
	return &house.RoomDef{}
}

func GivenARoom() *house.Room {
	roomDef := GivenARoomDef()
	return &house.Room{
		RoomDef: roomDef,
	}
}

func TestRoom(t *testing.T) {
	Convey("house.Room", t, RoomSpecs)
}

func RoomSpecs() {
	Convey("can be made", func() {
		room := GivenARoom()
		So(room, ShouldNotBeNil)
	})
}
