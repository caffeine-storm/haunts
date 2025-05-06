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

type RoomMats struct {
	Floor, IFloor, Left, ILeft, Right, IRight mathgl.Mat4
}

func MakeRoomMatsForCamera(room *house.Room, cam CameraConfig) RoomMats {
	floor, ifloor, left, ileft, right, iright := house.MakeRoomMatsForTest(room, cam.Region, cam.FocusX, cam.FocusY, cam.Angle, cam.Zoom)

	return RoomMats{
		Floor:  floor,
		IFloor: ifloor,
		Left:   left,
		ILeft:  ileft,
		Right:  right,
		IRight: iright,
	}
}

type CameraConfig struct {
	FocusX, FocusY float32
	Angle          float32
	Zoom           float32
	Region         gui.Region
}

func Camera() CameraConfig {
	return CameraConfig{
		FocusX: float32(0),
		FocusY: float32(0),
		Angle:  float32(62),
		Zoom:   float32(1.0),
		Region: gui.Region{
			Point: gui.Point{X: 0, Y: 0},
			Dims:  gui.Dims{Dx: 64, Dy: 64},
		},
	}
}

func (c CameraConfig) At(x, y float32) CameraConfig {
	c.FocusX = x
	c.FocusY = y
	return c
}

func (c CameraConfig) ForSize(dims gui.Dims) CameraConfig {
	c.Region.Dims = dims
	return c
}
