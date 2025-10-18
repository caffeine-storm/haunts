package gametest

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/mrgnet"
	"github.com/MobRulesGames/haunts/texture"
)

func GivenAScript() string {
	// TODO(tmckee): we should prefer an in-memory []byte instead of reading the
	// file in game.startGameScript
	return filepath.Join("versus", "basic.lua")
}

func GivenAHouseName() string {
	return "tutorial"
}

func GivenAScenario() game.Scenario {
	return game.Scenario{
		Script:    GivenAScript(),
		HouseName: GivenAHouseName(),
	}
}

func GivenAPlayer() *game.Player {
	return &game.Player{}
}

func GivenAGamePanelForScenario(scenario game.Scenario) *game.GamePanel {
	player := GivenAPlayer()
	noSpecialData := map[string]string{}
	noGameKey := mrgnet.GameKey("")
	ret := game.MakeGamePanel(scenario, player, noSpecialData, noGameKey)

	queue := texture.GetRenderQueue()
	queue.Purge()
	ret.SetLosModeAll()
	queue.Purge()

	return ret
}

func GivenAGamePanel() *game.GamePanel {
	scenario := GivenAScenario()
	return GivenAGamePanelForScenario(scenario)
}

func GivenAGame() *game.Game {
	panel := GivenAGamePanel()

	return panel.GetGame()
}

func GivenAnEntity() *game.Entity {
	g := GivenAGame()
	return game.MakeEntity("Bosch's Ghost", g)
}
