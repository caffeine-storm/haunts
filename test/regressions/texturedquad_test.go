package regressions_test

import (
	"fmt"
	"image"
	"testing"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

// Test cross-talk was causing strange render issues; this exists to be a smoke
// test for rendering/GL state.
func TestTexturedQuadRegr(t *testing.T) {
	Convey("drawing textured quads", t, func() {
		screen := image.Rect(0, 0, 50, 50)
		rendertest.DeprecatedWithGlForTest(screen.Dx(), screen.Dy(), func(sys system.System, queue render.RenderQueueInterface) {
			queue.Queue(func(st render.RenderQueueState) {
				debug.LogAndClearGlErrors(logging.ErrorLogger())
				fmt.Printf("before\n\n\n\n")
				tex := rendertest.GivenATexture("images/red.png")
				fmt.Printf("after\n\n\n\n")

				rendertest.DrawTexturedQuad(screen, tex, st.Shaders())
			})
			queue.Purge()

			So(queue, rendertest.ShouldLookLikeFile, "heckinwhat")
		})
	})
}
