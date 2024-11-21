package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/house"
	. "github.com/smartystreets/goconvey/convey"
)

func GivenARoom() *house.Room {
	return &house.Room{}
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
