package housetest_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/render"
)

func TestRoomMatricesHelpers(t *testing.T) {
	t.Run("pre-tilt", func(t *testing.T) {
		roomMats := housetest.PreTiltRoomMatrices()

		// This floor transform should rotate its input by 45 degrees about the
		// z-axis, then translate to adjust to the middle of the room.
		preTiltFloor := mathgl.Mat4{
			housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			-housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			0, 0, 1, 0,
			100, 100, 0, 1,
		}
		if !housetest.MatsAreEqual(roomMats[0], preTiltFloor) {
			t.Fatalf("expected matrix mismatch: expected %+v, got %+v", render.Showmat(preTiltFloor), render.Showmat(roomMats[0]))
		}
	})

	t.Run("non-zero-zero focus", func(t *testing.T) {
		roomMats := housetest.MakeRoomMatrices()

		// The floor transform should rotate its input by 45 degrees about
		// the z-axis, then translate to adjust by the focus.
		expectedFloor := mathgl.Mat4{
			housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			-housetest.JankyOneOverRoot2, housetest.JankyOneOverRoot2, 0, 0,
			0, 0, 1, 0,
			100, 100 - 10*housetest.JankyOneOverRoot2, 0, 1,
		}

		if !housetest.MatsAreEqual(roomMats[0], expectedFloor) {
			t.Fatalf("expected matrix mismatch:\nexpected:\n%v\ngot:\n%v", render.Showmat(expectedFloor), render.Showmat(roomMats[0]))
		}
	})
}
