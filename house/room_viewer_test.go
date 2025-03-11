package house_test

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
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

func matsAreEqual(lhs, rhs mathgl.Mat4) bool {
	for i := range lhs {
		if lhs[i] != rhs[i] {
			return false
		}
	}
	return true
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

func TestMath(t *testing.T) {
	assert := assert.New(t)
	s, c := sincos(0)
	assert.Equal(pair(0, 1), pair(s, c), "0")

	s, c = sincos(math.Pi)
	assert.Equal(pair(0, -1), pair(s, c), "math.Pi")

	s, c = sincos(math.Pi / 2)
	assert.Equal(pair(1, 0), pair(s, c), "math.Pi/2")

	s, c = sincos(math.Pi / 4)
	// mathgl is trading accuracy for speed but should at least be internally
	// consistent.
	jankyOneOverRoot2 := mathgl.Fsin32(math.Pi / 4)
	assert.Equal(pair(jankyOneOverRoot2, jankyOneOverRoot2), pair(s, c), "math.Pi/2")
}

func TestMakeRoomMats(t *testing.T) {
	t.Run("blank room", func(t *testing.T) {
		DefaultRoomMatrices := func() []mathgl.Mat4 {
			defaultRoom := house.BlankRoom()
			defaultRegion := gui.Region{
				Point: gui.Point{X: 0, Y: 0},
				Dims:  gui.Dims{Dx: 200, Dy: 200},
			}
			defaultFocus := struct {
				X, Y float32
			}{
				X: 0,
				Y: 0,
			}
			defaultAngle := float32(0)
			defaultZoom := float32(1)
			a, b, c, d, e, f := house.MakeRoomMatsForTest(defaultRoom, defaultRegion, defaultFocus.X, defaultFocus.Y, defaultAngle, defaultZoom)

			return []mathgl.Mat4{a, b, c, d, e, f}
		}

		roomMats := DefaultRoomMatrices()

		jankyOneOverRoot2 := mathgl.Fsin32(math.Pi / 4)

		// The floor transform should rotate its input by 45 degrees about the
		// z-axis, then translate to adjust to the middle of the room then by the
		// focus.
		defaultFloor := mathgl.Mat4{
			jankyOneOverRoot2, jankyOneOverRoot2, 0, 0,
			-jankyOneOverRoot2, jankyOneOverRoot2, 0, 0,
			0, 0, 1, 0,
			100, 100, 0, 1,
		}
		if !matsAreEqual(roomMats[0], defaultFloor) {
			t.Fatalf("expected matrix mismatch: expected %+v, got %+v", render.Showmat(defaultFloor), render.Showmat(roomMats[0]))
		}
	})
	t.Run("non-zero-zero focus", func(t *testing.T) {
		MakeRoomMatrices := func() []mathgl.Mat4 {
			defaultRoom := house.BlankRoom()
			defaultRegion := gui.Region{
				Point: gui.Point{X: 0, Y: 0},
				Dims:  gui.Dims{Dx: 200, Dy: 200},
			}
			nonZeroFocus := struct {
				X, Y float32
			}{
				X: 5,
				Y: 5,
			}
			defaultAngle := float32(0)
			defaultZoom := float32(1)
			a, b, c, d, e, f := house.MakeRoomMatsForTest(defaultRoom, defaultRegion, nonZeroFocus.X, nonZeroFocus.Y, defaultAngle, defaultZoom)

			return []mathgl.Mat4{a, b, c, d, e, f}
		}

		roomMats := MakeRoomMatrices()

		jankyOneOverRoot2 := mathgl.Fsin32(math.Pi / 4)

		// The floor transform should rotate its input by 45 degrees about
		// the z-axis, then translate to adjust by the focus.
		expectedFloor := mathgl.Mat4{
			jankyOneOverRoot2, jankyOneOverRoot2, 0, 0,
			-jankyOneOverRoot2, jankyOneOverRoot2, 0, 0,
			0, 0, 1, 0,
			100, 100 - 10*jankyOneOverRoot2, 0, 1,
		}

		if !matsAreEqual(roomMats[0], expectedFloor) {
			t.Fatalf("expected matrix mismatch:\nexpected:\n%v\ngot:\n%v", render.Showmat(expectedFloor), render.Showmat(roomMats[0]))
		}
	})

	t.Run("floor matrix properly smushes a floor image", func(t *testing.T) {
		// TODO(#10): don't skip, fix!
		t.Skip("#10: skipping so we can focus elsewhere for now")

		floorMatrix := &mathgl.Mat4{
			8.622922, 8.622922, 0, 0,
			-8.622922, 8.622922, 0, 0,
			0, 0, 1, 0,
			128, 41.770782, 0, 1,
		}

		imageName := "floor_21_mahogany_tiny"
		screenRegion := image.Rect(0, 0, 400, 400)
		expected := rendertest.MustLoadImage(fmt.Sprintf("testdata/images/%s_expected.png", imageName))

		rendertest.WithGlForTest(screenRegion.Dx(), screenRegion.Dy(), func(sys system.System, queue render.RenderQueueInterface) {

			var result image.Image
			queue.Queue(func(st render.RenderQueueState) {
				tex := rendertest.GivenATexture(fmt.Sprintf("images/%s.png", imageName))
				render.WithMatrixInMode(floorMatrix, render.MatrixModeModelView, func() {
					rendertest.DrawTexturedQuad(screenRegion, tex, st.Shaders())
				})

				debug.LogAndClearGlErrors(logging.ErrorLogger())

				result = debug.ScreenShotRgba(screenRegion.Dx(), screenRegion.Dy())
			})
			queue.Purge()

			ok := rendertest.ImagesAreWithinThreshold(result, expected, rendertest.Threshold(3), color.RGBA{A: 255})

			if !ok {
				outfileName := fmt.Sprintf("testdata/images/%s.rej.png", imageName)
				outfile, err := os.Create(outfileName)
				if err != nil {
					panic(fmt.Errorf("couldn't os.Create %q: %w", outfileName, err))
				}
				defer outfile.Close()

				err = png.Encode(outfile, result)
				if err != nil {
					panic(fmt.Errorf("couldn't png.Encode %q: %w", outfileName, err))
				}

				t.Fatalf("image mismatch")
			}
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
	rendertest.WithGlForTest(screenRegion.Dx, screenRegion.Dy, func(sys system.System, queue render.RenderQueueInterface) {
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
