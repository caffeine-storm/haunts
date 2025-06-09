package game_test

import "github.com/MobRulesGames/haunts/game"

func givenAScenario() game.Scenario {
	return game.Scenario{
		Script:    givenAScript(),
		HouseName: givenAHouseName(),
	}
}
