package perspective

import (
	"math"

	"github.com/MobRulesGames/mathgl"
)

// Returns a matrix to transform entities from "floor space" to one that is
// coplanar with the screen. In effect, to "stand up" entities that are "lying
// down".
//
// TODO(tmckee#47): the x/y parameters should be of type house.BoardSpaceUnit
// (but that would cause a circular import r.n.)
func MakeStandupTransform(roomX, roomY int) *mathgl.Mat4 {
	near_x, near_y := float32(roomX), float32(roomY)
	step := &mathgl.Mat4{}
	standup := &mathgl.Mat4{}
	standup.Identity()

	// Step 4, undo the initial translation
	step.Translation(near_x, near_y, 0)
	standup.Multiply(step)

	// Step 3, rotate about (-1, 1, 0) to undo the floor's "tilt" rotation.
	axis := mathgl.Vec3{X: -1, Y: 1, Z: 0}
	// TODO(tmckee:clean): don't hardcode '62'; read it from a RoomMats field.
	step.RotationAxisAngle(axis, 62*math.Pi/180)
	standup.Multiply(step)

	// Step 2, rotate about Z to undo the floor's Z rotation.
	step.RotationZ(-math.Pi / 4.0)
	standup.Multiply(step)

	// Step 1, translate the viewer to move the target object to (0, 0).
	step.Translation(-near_x, -near_y, 0)
	standup.Multiply(step)
	return standup
}
