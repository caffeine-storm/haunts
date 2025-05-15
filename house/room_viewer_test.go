package house_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"
	. "github.com/smartystreets/goconvey/convey"
)

type drawWithLoggingTrace struct {
	gametest.Drawer
}

func (it *drawWithLoggingTrace) Draw(reg gui.Region, ctx gui.DrawingContext) {
	logging.TraceBracket(func() {
		it.Drawer.Draw(reg, ctx)
	})
}

func TestRoomViewer(t *testing.T) {
	base.SetDatadir("../data")

	Convey("house.roomViewer", t, func() {
		gametest.RunDrawingTest(func() gametest.Drawer {
			room := loadRoom("restest.room")
			return &drawWithLoggingTrace{
				house.MakeRoomViewer(room, 62),
			}
		}, "room-viewer")
	})
}
