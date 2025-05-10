package game_test

import (
	"fmt"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"

	. "github.com/smartystreets/goconvey/convey"
)

func givenAnOnlineMenu() *game.OnlineMenu {
	parent := &gui.StandardParent{}
	err := game.InsertOnlineMenu(parent)
	if err != nil {
		panic(fmt.Errorf("couldn't insert start menu: %w", err))
	}

	return parent.GetChildren()[0].(*game.OnlineMenu)
}

func TestUiOnline(t *testing.T) {
	Convey("UI for starting an online game", t, func() {
		base.SetDatadir("../data")

		uiMaker := func(render.RenderQueueInterface) gametest.Drawer {
			return givenAnOnlineMenu()
		}
		gametest.RunDrawingTest(uiMaker, "online", func(gametest.DrawTestContext) {})
	})
}
