package perspective

import (
	"math"

	"github.com/MobRulesGames/mathgl"
)

// Returns a matrix to transform entities from "floor space" to one that is
// coplanar with the screen. In effect, to "stand up" entities that are "lying
// down".
func MakeStandupTransform(mats *RoomMats, roomX, roomY BoardSpaceUnit) *mathgl.Mat4 {
	near_x, near_y := float32(roomX), float32(roomY)
	step := &mathgl.Mat4{}
	standup := &mathgl.Mat4{}
	standup.Identity()

	// Step 4, undo the initial translation
	step.Translation(near_x, near_y, 0)
	standup.Multiply(step)

	// Step 3, rotate about Z to undo the floor's Z rotation.
	step.RotationZ(-math.Pi / 4.0)
	standup.Multiply(step)

	// Step 2, rotate about (1, 0, 0) to undo the floor's "tilt" rotation.
	step.RotationX(-mats.angle * math.Pi / 180)
	standup.Multiply(step)

	// Step 1, translate the viewer to move the target object to (0, 0).
	step.Translation(-near_x, -near_y, 0)
	standup.Multiply(step)
	return standup
}
