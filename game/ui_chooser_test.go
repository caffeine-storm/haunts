package game_test

import (
	"fmt"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAUIChooser() *game.Chooser {
	someOptions := gametest.GivenSomeOptions()
	chooser, selectionChan, err := game.MakeChooser(someOptions)
	if err != nil {
		panic(fmt.Errorf("couldn't MakeChooser: %w", err))
	}
	go func() {
		<-selectionChan
	}()
	return chooser
}

func TestChooser(t *testing.T) {
	count := int64(5)
	Convey("Chooser UI", t, func() {
		base.SetDatadir("../data")
		chooser := givenAUIChooser()
		Convey("can draw chooser UI", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				chooser.Think(nil, count)
				count += 5
				return chooser
			}, "chooser")
		})
	})
}
