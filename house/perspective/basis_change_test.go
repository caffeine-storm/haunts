package perspective_test

import (
	"math"
	"testing"

	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/mathgl"
)

const rightAngle = math.Pi / 2

func TestBoardToModelView(t *testing.T) {
	ident := &mathgl.Mat4{}
	ident.Identity()
	x, y, z := perspective.BoardToModelview(ident, 5, 7)

	if x != 5 || y != 7 || z != 0 {
		t.Fatalf("wait, wut?")
	}
}

// Copied from logs of the 'floor' matrix in a HouseViewer.
// Assumed to be a CC-rotation about Z by 45 degrees followed by a CC-rotation
// about X by 62 degrees. Note: in column-major order.
var rawfloor = mathgl.Mat4{
	7.078124, 3.3219588, 6.247571, 0,
	-7.078124, 3.3219588, 6.247571, 0,
	0, -8.844217, 4.6932755, 0,
	0, 0, 0, 1,
}

func TestModelviewToBoard(t *testing.T) {
	t.Run("identity transform", func(t *testing.T) {
		ident := &mathgl.Mat4{}
		ident.Identity()
		x, y, z := perspective.ModelviewToBoard(ident, 5, 7)

		if x != 5 || y != 7 || z != 0 {
			t.Fatalf("wait, wut?")
		}
	})

	t.Run("rotation and translation", func(t *testing.T) {
		xfrm := &mathgl.Mat4{}
		xfrm.Identity()

		tmp := &mathgl.Mat4{}
		tmp.Translation(4, 5, 6)
		xfrm.Multiply(tmp)

		tmp.RotationZ(rightAngle)
		xfrm.Multiply(tmp)

		xunit := mathgl.Vec3{X: 1}
		yunit := mathgl.Vec3{Y: 1}
		zunit := mathgl.Vec3{Z: 1}

		xunit.Transform(xfrm)
		yunit.Transform(xfrm)
		zunit.Transform(xfrm)

		if xunit != (mathgl.Vec3{X: 4, Y: 6, Z: 6}) {
			t.Fatalf("narp")
		}
		if yunit != (mathgl.Vec3{X: 3, Y: 5, Z: 6}) {
			t.Fatalf("negatory")
		}
		if zunit != (mathgl.Vec3{X: 4, Y: 5, Z: 7}) {
			t.Fatalf("gah!")
		}

		x, y, z := perspective.ModelviewToBoard(xfrm, 10, 20)

		if x != -16 || y != 15 || z != 6 {
			t.Fatalf("bad transform: result: (%v, %v, %v)", x, y, z)
		}
	})
}
