package game_test

import (
	"fmt"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/caffeine-storm/glop/gui"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAMainBar() gametest.Drawer {
	aGame := gametest.GivenAGame()
	mainBar, err := game.MakeMainBar(aGame)
	if err != nil {
		panic(fmt.Errorf("couldn't MakeMainBar: %w", err))
	}

	wrapper := &gui.AnchorBox{}
	wrapper.AddChild(mainBar, gui.Anchor{})

	return wrapper
}

func TestMainBar(t *testing.T) {
	Convey("UI for the main bar", t, func(c C) {
		base.SetDatadir("../data")
		gametest.RunDrawingTest(c, givenAMainBar, "mainbar")
	})
}
