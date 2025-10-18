package leveltests_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/test/leveltests"
)

func TestLevel1(t *testing.T) {
	leveltests.EndToEndTest(t, leveltests.Level1, leveltests.ModePassNPlay, func(tst leveltests.Tester) {
		tst.ValidateExpectations("initial")
	})
}
