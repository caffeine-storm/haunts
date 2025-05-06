package housetest

import (
	"math"

	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/gui"
)

// mathgl is trading accuracy for speed but should at least be internally
// consistent.
var JankyOneOverRoot2 = mathgl.Fsin32(math.Pi / 4)

func MatsAreEqual(lhs, rhs mathgl.Mat4) bool {
	for i := range lhs {
		if lhs[i] != rhs[i] {
			return false
		}
	}
	return true
}

func PreTiltRoomMatrices() []mathgl.Mat4 {
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

func MakeRoomMatrices() []mathgl.Mat4 {
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
