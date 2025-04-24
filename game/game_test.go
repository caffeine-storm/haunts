package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/mrgnet"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAScript() string {
	return ""
}

func givenAPlayer() *game.Player {
	return &game.Player{}
}

func givenAGamePanel() *game.GamePanel {
	scriptString := givenAScript()
	player := givenAPlayer()
	noSpecialData := map[string]string{}
	noGameKey := mrgnet.GameKey("")
	return game.MakeGamePanel(scriptString, player, noSpecialData, noGameKey)
}

func TestMakeGamePanel(t *testing.T) {
	Convey("GamePanelSpecs", t, func() {
		base.SetDatadir("../data")
		Convey("can draw", func() {
			thePanel := givenAGamePanel()
			gametest.RunDrawingTest(thePanel, "game-panel", func(gametest.DrawTestContext) {})
		})
	})
}
