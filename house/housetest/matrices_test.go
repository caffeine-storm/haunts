package housetest_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/caffeine-storm/glop/render"
	"github.com/caffeine-storm/mathgl"
)

func TestRoomMatricesHelpers(t *testing.T) {
	t.Run("pre-tilt", func(t *testing.T) {
		camera := housetest.PreTiltCamera().ForSize(200, 200)
		floorMat := housetest.MakeRoomMatsForCamera(*house.BlankRoomSize(), camera).Floor

		// This floor transform should rotate its input by 45 degrees about the
		// z-axis, then translate to adjust to the middle of the room.
		preTiltFloor := mathgl.Mat4{
			housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			-housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			0, 0, 1, 0,
			100, 100, 0, 1,
		}
		if !housetest.MatsAreEqual(floorMat, preTiltFloor) {
			t.Fatalf("matrix mismatch: expected %+v, got %+v", render.Showmat(preTiltFloor), render.Showmat(floorMat))
		}
	})

	t.Run("non-zero-zero focus", func(t *testing.T) {
		camera := housetest.Camera().
			AtFocus(5, 5).
			ForSize(200, 200).
			AtAngle(0)
		floorMatrix := housetest.MakeRoomMatsForCamera(*house.BlankRoomSize(), camera).Floor

		// The floor transform should rotate its input by 45 degrees about
		// the z-axis, then translate to adjust by the focus.
		expectedFloor := mathgl.Mat4{
			housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			-housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			0, 0, 1, 0,
			100, 100 - 10*housetest.JankyOneOverRoot2, 0, 1,
		}

		if !housetest.MatsAreEqual(floorMatrix, expectedFloor) {
			t.Fatalf("matrix mismatch:\nexpected:\n%v\ngot:\n%v", render.Showmat(expectedFloor), render.Showmat(floorMatrix))
		}
	})
}
