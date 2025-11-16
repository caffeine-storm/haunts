package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	_ "github.com/MobRulesGames/haunts/game/actions"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/caffeine-storm/glop/gui"
	"github.com/caffeine-storm/mathgl"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEntity(t *testing.T) {
	Convey("EntitySpecs", t, func() {
		base.SetDatadir("../data")
		registry.LoadAllRegistries()
		game.LoadAllEntities()

		Convey("can draw an entity", func(c C) {
			gametest.RunDrawingTest(c, func() gametest.Drawer {
				ent := gametest.GivenAnEntity()
				return gametest.DrawerAdapter(func(region gui.Region, _ gui.DrawingContext) {
					// Render takes 'pre-image' co-ordinates so, for this test, we'll
					// compute a position and width based on the screen size given in
					// 'region'.
					leftx := float32(region.Dx) / 2
					bottomy := float32(region.Dy) / 2
					ent.Render(mathgl.Vec2{X: leftx, Y: bottomy}, float32(region.Dx)/5)
				})()
			}, "bosch-ghost")
		})
	})
}
