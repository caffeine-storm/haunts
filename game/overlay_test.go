package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAnOverlay() gametest.Drawer {
	return game.MakeOverlay(givenAGame())
}

func TestOverlay(t *testing.T) {
	Convey("can be drawn", t, func() {
		gametest.RunTracedDrawingTest(givenAnOverlay, "overlay")
	})
}
