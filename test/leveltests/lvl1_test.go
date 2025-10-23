package leveltests_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/test/leveltests"
)

func TestLevel1(t *testing.T) {
	leveltests.EndToEndTest(t, "level-01", func(tst leveltests.Planner) {
		tst.PlanAndRun(
			tst.StartApplication(),
			tst.ChooseVersusMode(),
			tst.ChooseLevel(leveltests.Level1),
			tst.ChooseDenizens(),
			tst.PlaceRoster(),
		)
	})
}
