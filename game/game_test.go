package game_test

import (
	"path/filepath"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/mrgnet"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAScript() string {
	return filepath.Join("versus", "basic.lua")
}

func givenAPlayer() *game.Player {
	return &game.Player{}
}

func givenAHouseName() string {
	return "tutorial"
}

func givenAScenario() game.Scenario {
	return game.Scenario{
		Script:    givenAScript(),
		HouseName: givenAHouseName(),
	}
}

var _ gametest.Drawer = (*game.GamePanel)(nil)

func givenAGamePanel() gametest.Drawer {
	scenario := givenAScenario()
	player := givenAPlayer()
	noSpecialData := map[string]string{}
	noGameKey := mrgnet.GameKey("")
	return game.MakeGamePanel(scenario, player, noSpecialData, noGameKey)
}

func TestGamePanel(t *testing.T) {
	// TODO(#10): re-enable this after pre-factors
	SkipConvey("GamePanelSpecs", t, func() {
		base.SetDatadir("../data")
		Convey("can draw game panel", func() {
			logging.TraceBracket(func() {
				gametest.RunDrawingTest(givenAGamePanel, "game-panel", func(gametest.DrawTestContext) {})
			})
		})
	})
}
