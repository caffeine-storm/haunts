package perspective

import (
	"math"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
)

type RoomSizey interface {
	GetDx() int
	GetDy() int
}

type RoomMats struct {
	Floor, IFloor, Left, ILeft, Right, IRight mathgl.Mat4
}

// TODO(tmckee:clean): taking a size by interface is leading to a bunch of
// deref/address-of noise; pick something that works and use it everywhere
func MakeRoomMats(roomSize RoomSizey, region gui.Region, focusx, focusy, angle, zoom float32) (ret RoomMats) {
	// Note: repeated matrix multiplication is equivalent to composing
	// application of a series of transforms in reverse. So, we build up a
	// transform by multiplying logical pieces but its easiest to see the overall
	// transform by reading in the opposite order. Hence, we start at 'Step 5'.
	var m mathgl.Mat4

	// Step 5: translate the resulting (possibly-squished) diamond to the centre
	// of a target region.
	ret.Floor.Translation(float32(region.X+region.Dx/2), float32(region.Y+region.Dy/2), 0)

	// Step 4: rotate about the z axis to put the bottom-left (and, from step3,
	// most +'ve in z point) at the bottom, and the top-right at the top.
	// NOTE: If we want to change 45 to *anything* else then we need to do the
	// appropriate math for rendering quads for furniture
	m.RotationZ(45 * math.Pi / 180)
	ret.Floor.Multiply(&m)

	// Step 3: rotate about an axis so as to "push back" the top-right and "pull
	// forward" the bottom-left by a given angle.
	m.RotationAxisAngle(mathgl.Vec3{X: -1, Y: 1}, -float32(angle)*math.Pi/180)
	ret.Floor.Multiply(&m)

	// Step 2: zoom in or out on the floor.
	s := float32(zoom)
	m.Scaling(s, s, s)
	ret.Floor.Multiply(&m)

	// Step 1: Move the viewer so that the focus is at the origin, and hence
	// becomes centered in the window.
	m.Translation(-focusx, -focusy, 0)
	ret.Floor.Multiply(&m)

	// Step 0: Assume an input floor from (x,y) to (x+dx, x+dy), rotated to match
	// our natural co-ordinates.

	ret.IFloor.Assign(&ret.Floor)
	ret.IFloor.Inverse()

	// Also make the mats for the left and right walls based on this mat
	ret.Left.Assign(&ret.Floor)
	m.RotationX(-math.Pi / 2)
	ret.Left.Multiply(&m)
	m.Translation(0, 0, float32(roomSize.GetDy()))
	ret.Left.Multiply(&m)
	ret.ILeft.Assign(&ret.Left)
	ret.ILeft.Inverse()

	ret.Right.Assign(&ret.Floor)
	m.RotationX(-math.Pi / 2)
	ret.Right.Multiply(&m)
	m.RotationY(-math.Pi / 2)
	ret.Right.Multiply(&m)
	m.Scaling(1, 1, 1)
	ret.Right.Multiply(&m)
	m.Translation(0, 0, -float32(roomSize.GetDx()))
	ret.Right.Multiply(&m)
	swap_x_y := mathgl.Mat4{
		0, 1, 0, 0,
		1, 0, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	ret.Right.Multiply(&swap_x_y)
	ret.IRight.Assign(&ret.Right)
	ret.IRight.Inverse()

	logging.Trace("makeRoomMats returning",
		"roomsize", roomSize,
		"region", region,
		"focusx", focusx,
		"focusy", focusy,
		"angle", angle,
		"zoom", zoom,
		"floor", render.Showmat(ret.Floor),
		"left", render.Showmat(ret.Left),
		"right", render.Showmat(ret.Right),
	)
	return
}
