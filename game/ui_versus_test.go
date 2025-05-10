package game_test

import (
	"fmt"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"
	. "github.com/smartystreets/goconvey/convey"
)

func panicInsteadOfReplace(x gui.WidgetParent) error {

	panic(fmt.Errorf("panicInsteadOfReplace got called with %v", x))
}

func givenAVersusMenu() gametest.Drawer {
	parent := &gui.StandardParent{}
	err := game.InsertVersusMenu(parent, panicInsteadOfReplace)
	if err != nil {
		panic(fmt.Errorf("couldn't insert versus menu: %w", err))
	}

	return parent.GetChildren()[0].(gametest.Drawer)
}

func TestUiVersus(t *testing.T) {
	Convey("UI for starting an online game", t, func() {
		base.SetDatadir("../data")

		logging.DebugBracket(func() {
			gametest.RunDrawingTest(givenAVersusMenu, "versus", func(gametest.DrawTestContext) {})
		})
	})
}
