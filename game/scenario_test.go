package game_test

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/game"
)

func givenAScript() string {
	return filepath.Join("versus", "basic.lua")
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
