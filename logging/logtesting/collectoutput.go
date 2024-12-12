package logtesting

import (
	"os"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gloptest"
)

// Like gloptest.CollectOuput but knows about the haunts.logging package and
// what's needed to capture that output.
func CollectOutput(fn func()) []string {
	return gloptest.CollectOutput(func() {
		reset := logging.Redirect(os.Stderr)
		defer reset()

		fn()
	})
}
