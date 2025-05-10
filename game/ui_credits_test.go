package game_test

import (
	"fmt"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/runningwild/glop/gui"
	. "github.com/smartystreets/goconvey/convey"
)

func givenACreditsMenu() gametest.Drawer {
	parent := &gui.StandardParent{}
	err := game.InsertCreditsMenu(parent)
	if err != nil {
		panic(fmt.Errorf("couldn't insert credits menu: %w", err))
	}

	return parent.GetChildren()[0].(gametest.Drawer)
}

func TestUiCredits(t *testing.T) {
	Convey("UI for the Credits screen", t, func() {
		base.SetDatadir("../data")

		gametest.RunDrawingTest(givenACreditsMenu, "credits", func(gametest.DrawTestContext) {})
	})
}
