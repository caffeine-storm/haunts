package game_test

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/game"
)

func givenAScript() string {
	// TODO(tmckee): we should prefer an in-memory []byte instead of reading the
	// file in game.startGameScript
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
