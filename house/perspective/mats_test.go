package perspective_test

import (
	"image"
	"math"
	"testing"

	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
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

func TestMakeRoomMats(t *testing.T) {
	Convey("floor matrix properly smushes a floor image", t, func() {
		cam := housetest.Camera().ForSize(400, 400).AtAngle(0).AtFocus(200/housetest.JankyOneOverRoot2, 0)
		floorMatrix := perspective.MakeRoomMats(house.BlankRoomSize(), cam.Region, cam.FocusX, cam.FocusY, cam.Angle, cam.Zoom).Floor

		screen := image.Rect(0, 0, 400, 400)
		rendertest.DeprecatedWithGlForTest(screen.Dx(), screen.Dy(), func(sys system.System, queue render.RenderQueueInterface) {
			queue.Queue(func(st render.RenderQueueState) {
				debug.LogAndClearGlErrors(logging.ErrorLogger())
				tex := rendertest.GivenATexture("mahogany/input.png")
				render.WithMultMatrixInMode(&floorMatrix, render.MatrixModeModelView, func() {
					rendertest.DrawTexturedQuad(screen, tex, st.Shaders())
				})
			})
			queue.Purge()

			So(queue, rendertest.ShouldLookLikeFile, "mahogany")
		})
	})
}
