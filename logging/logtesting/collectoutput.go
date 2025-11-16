package logtesting

import (
	"os"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/caffeine-storm/glop/gloptest"
)

// Like gloptest.CollectOuput but knows about the haunts.logging package and
// what's needed to capture that output.
func CollectOutput(fn func()) []string {
	return gloptest.CollectOutput(func() {
		// We can redirect to the current os.Stderr because that's what
		// gloptest.CollectOutput will be collecting.
		reset := logging.Redirect(os.Stderr)
		defer reset()

		fn()
	})
}
