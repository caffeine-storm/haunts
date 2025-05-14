package regressions_test

import (
	"image"
	"testing"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
	. "github.com/smartystreets/goconvey/convey"
)

// Test cross-talk was causing strange render issues; this exists to be a smoke
// test for rendering/GL state.
func TestTexturedQuadRegr(t *testing.T) {
	Convey("drawing textured quads", t, func() {
		screen := image.Rect(0, 0, 50, 50)
		testbuilder.New().WithSize(screen.Dx(), screen.Dy()).WithQueue().Run(func(queue render.RenderQueueInterface) {
			queue.Queue(func(st render.RenderQueueState) {
				debug.LogAndClearGlErrors(logging.ErrorLogger())
				tex := rendertest.GivenATexture("images/red.png")

				rendertest.DrawTexturedQuad(screen, tex, st.Shaders())
			})
			queue.Purge()

			So(queue, rendertest.ShouldLookLikeFile, "heckinwhat")
		})
	})
}
