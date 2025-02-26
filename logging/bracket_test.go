package logging_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gloptest"
	"github.com/stretchr/testify/assert"
)

const canary = "lololololololol"

func TestLoggingBracket(t *testing.T) {
	t.Run("at error should block trace", func(t *testing.T) {
		loglines := gloptest.CollectOutput(func() {
			logging.Bracket(slog.LevelError, func() {
				logging.Trace(canary)
			})
		})

		assert.NotContains(t, strings.Join(loglines, "\n"), canary)
	})
}
