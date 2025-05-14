package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRoomViewer(t *testing.T) {
	base.SetDatadir("../data")

	Convey("house.roomViewer", t, func() {
		gametest.RunDrawingTest(func() gametest.Drawer {
			room := loadRoom("restest.room")
			return house.MakeRoomViewer(room, 0)
		}, "room-viewer")
	})
}
