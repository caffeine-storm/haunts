package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	_ "github.com/MobRulesGames/haunts/game/actions"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/gui"
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
					x := float32(region.Dx) / 2
					y := float32(region.Dy) / 2
					ent.Render(mathgl.Vec2{X: x, Y: y}, float32(region.Dx)/5)
				})()
			}, "bosch-ghost")
		})
	})
}
