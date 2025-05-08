package house_test

import (
	"image"
	"math"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestRoomViewer(t *testing.T) {
	Convey("house.roomViewer", t, RoomViewerSpecs)
}

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

var jankyOneOverRoot2 = housetest.JankyOneOverRoot2

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
		camera := housetest.Camera().ForSize(400, 400).AtAngle(0).At(200/housetest.JankyOneOverRoot2, 0)
		floorMatrix := housetest.MakeRoomMatsForCamera(house.BlankRoomSize(), camera).Floor

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

// Test cross-talk was causing strange render issues; this exists to be a smoke
// test for rendering/GL state.
func TestTexturedQuadRegr(t *testing.T) {
	Convey("drawing textured quads", t, func() {
		screen := image.Rect(0, 0, 50, 50)
		rendertest.DeprecatedWithGlForTest(screen.Dx(), screen.Dy(), func(sys system.System, queue render.RenderQueueInterface) {
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

func RoomViewerSpecs() {
	base.SetDatadir("../data")

	screenRegion := gui.Region{
		Point: gui.Point{
			X: 0, Y: 0,
		},
		Dims: gui.Dims{
			Dx: 256, Dy: 256,
		},
	}
	rendertest.DeprecatedWithGlForTest(screenRegion.Dx, screenRegion.Dy, func(sys system.System, queue render.RenderQueueInterface) {
		registry.LoadAllRegistries()
		base.InitShaders(queue)
		texture.Init(queue)
		room := loadRoom("restest.room")

		Convey("can be made", func() {
			rv := house.MakeRoomViewer(room, 0)
			So(rv, ShouldNotBeNil)

			// TODO(#10): don't skip, fix!
			SkipConvey("can be drawn", func() {
				g := guitest.MakeStubbedGui(screenRegion.Dims)
				g.AddChild(rv)
				queue.Queue(func(render.RenderQueueState) {
					logging.TraceBracket(func() {
						g.Draw()
					})
				})
				queue.Purge()

				So(queue, rendertest.ShouldLookLikeFile, "room-viewer", rendertest.Threshold(0))
			})
		})
	})
}
