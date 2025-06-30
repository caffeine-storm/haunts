package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
)

func TestButton(t *testing.T) {
	base.SetDatadir("../data")

	t.Run("hover effects respect mouse position", func(t *testing.T) {
		screenDims := gui.Dims{
			Dx: 256, Dy: 256,
		}
		testbuilder.WithSize(screenDims.Dx, screenDims.Dy, func(queue render.RenderQueueInterface) {
			queue.Queue(func(st render.RenderQueueState) {
				globals.SetRenderQueueState(st)
			})
			queue.Purge()

			ctx := gametest.GivenADrawingContext(screenDims)
			base.InitDictionaries(ctx)
			texture.Init(queue)

			btn := game.Button{
				X: 3,
				Y: 5,
			}
			btn.Text.String = "some button text"
			btn.Text.Justification = "center"
			btn.Text.Size = 18

			queue.Queue(func(render.RenderQueueState) {
				btn.RenderAt(0, 0)
			})
			queue.Purge()

			if !btn.Over(7, 10) {
				t.Fatalf("button hit test should have hit but missed")
			}
		})
	})
}
