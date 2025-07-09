package game_test

import (
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	. "github.com/smartystreets/goconvey/convey"
)

type stubTimer struct {
	val time.Time
}

func (t *stubTimer) Now() time.Time {
	return t.val
}

func givenAnOverlay(gm *game.Game) gametest.Drawer {
	// Use a stubTimer so that output is consistent from run-to-run.
	return game.MakeOverlayWithTimer(gm, &stubTimer{
		val: time.UnixMilli(42),
	})
}

func givenAnOverlayWithWaypoints() gametest.Drawer {
	agame := givenAGame()
	agame.Waypoints = []game.Waypoint{
		{
			Name:   "wp1",
			Side:   game.SideExplorers,
			X:      80,
			Y:      15,
			Radius: 40.0,
			Active: true,
		},
	}

	return givenAnOverlay(agame)
}

func TestOverlay(t *testing.T) {
	base.SetDatadir("../data")
	Convey("can be drawn", t, func(c C) {
		gametest.RunDrawingTest(c, givenAnOverlayWithWaypoints, "overlay")
	})
}
