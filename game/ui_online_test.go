package game_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"

	. "github.com/smartystreets/goconvey/convey"
)

func givenAnOnlineMenuLayout() *game.OnlineLayout {
	ret := game.OnlineLayout{}

	err := base.LoadAndProcessObject(filepath.Join(base.GetDataDir(), "ui", "start", "online", "layout.json"), "json", &ret)
	if err != nil {
		panic(fmt.Errorf("couldn't load layout: %w", err))
	}

	return &ret
}

func givenAnOnlineMenu() *game.OnlineMenu {
	parent := &gui.StandardParent{}
	// layout := givenAnOnlineMenuLayout()
	err := game.InsertOnlineMenu(parent)
	if err != nil {
		panic(fmt.Errorf("couldn't insert start menu: %w", err))
	}

	return parent.GetChildren()[0].(*game.OnlineMenu)
}

func TestUiOnline(t *testing.T) {
	Convey("UI for starting an online game", t, func() {
		base.SetDatadir("../data")
		onlineScreen := givenAnOnlineMenu()

		logging.DebugBracket(func() {
			gametest.RunDrawingTest(onlineScreen, "online", func(gametest.DrawTestContext) {})
		})
	})
}
