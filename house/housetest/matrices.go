package housetest

import (
	"math"

	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/perspective"
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

func MakeRoomMatsForCamera(roomSize house.RoomSize, cam CameraConfig) perspective.RoomMats {
	floor, ifloor, left, ileft, right, iright := perspective.MakeRoomMats(&roomSize, cam.Region, cam.FocusX, cam.FocusY, cam.Angle, cam.Zoom)

	return perspective.RoomMats{
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

func PreTiltCamera() CameraConfig {
	return Camera().AtAngle(0)
}

// TODO(tmckee:clean): rename AtFocus
func (c CameraConfig) At(x, y float32) CameraConfig {
	c.FocusX = x
	c.FocusY = y
	return c
}

func (c CameraConfig) ForSize(dx, dy int) CameraConfig {
	c.Region.Dims = gui.Dims{Dx: dx, Dy: dy}
	return c
}

func (c CameraConfig) AtAngle(theta float32) CameraConfig {
	c.Angle = theta
	return c
}

func (c CameraConfig) AtZoom(zoom float32) CameraConfig {
	c.Zoom = zoom
	return c
}
