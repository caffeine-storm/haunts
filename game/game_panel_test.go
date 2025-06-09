package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/mrgnet"
	. "github.com/smartystreets/goconvey/convey"
)

var _ gametest.Drawer = (*game.GamePanel)(nil)

func givenAGamePanel() gametest.Drawer {
	scenario := givenAScenario()
	player := givenAPlayer()
	noSpecialData := map[string]string{}
	noGameKey := mrgnet.GameKey("")
	return game.MakeGamePanel(scenario, player, noSpecialData, noGameKey)
}

func TestGamePanel(t *testing.T) {
	// TODO(#10): once the Overlay and House Viewer are working, we can come back
	// to getting the GamePanel working.
	SkipConvey("GamePanelSpecs", t, func() {
		base.SetDatadir("../data")
		Convey("can draw game panel", func() {
			gametest.RunDrawingTest(givenAGamePanel, "game-panel")
		})
	})
}
