package perspective_test

import (
	"image"
	"math"
	"testing"

	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

var jankyOneOverRoot2 = housetest.JankyOneOverRoot2

func sincos(f float32) (float32, float32) {
	return mathgl.Fsin32(f), mathgl.Fcos32(f)
}

type floatPair struct {
	a, b float32
}

func pair(a, b float32) floatPair {
	return floatPair{
		a: a,
		b: b,
	}
}

func TestMath(t *testing.T) {
	assert := assert.New(t)
	s, c := sincos(0)
	assert.Equal(pair(0, 1), pair(s, c), "0")

	s, c = sincos(math.Pi)
	assert.Equal(pair(0, -1), pair(s, c), "math.Pi")

	s, c = sincos(math.Pi / 2)
	assert.Equal(pair(1, 0), pair(s, c), "math.Pi/2")

	s, c = sincos(math.Pi / 4)
	assert.Equal(pair(jankyOneOverRoot2, jankyOneOverRoot2), pair(s, c), "math.Pi/4")
}

func TestMakeFloorTransforms(t *testing.T) {
	Convey("floor matrix properly smushes a floor image", t, func(c C) {
		screen := image.Rect(0, 0, 400, 400)
		cam := housetest.Camera().
			ForSize(screen.Dx(), screen.Dy()).
			AtAngle(0).
			AtFocus(float32(screen.Dx())*housetest.JankyOneOverRoot2, 0)
		floorMatrix, _ := perspective.MakeFloorTransforms(cam.Region, cam.FocusX, cam.FocusY, cam.Angle, cam.Zoom)

		testbuilder.New().WithSize(screen.Dx(), screen.Dy()).WithQueue().Run(func(queue render.RenderQueueInterface) {
			queue.Queue(func(st render.RenderQueueState) {
				render.LogAndClearGlErrors(logging.ErrorLogger())
				tex, cleanup := rendertest.GivenATexture("mahogany/input.png")
				defer cleanup()
				render.WithMultMatrixInMode(&floorMatrix, render.MatrixModeModelView, func() {
					rendertest.DrawTexturedQuad(screen, tex, st.Shaders())
				})
			})
			queue.Purge()

			c.So(queue, rendertest.ShouldLookLikeFile, "mahogany")
		})
	})
}
