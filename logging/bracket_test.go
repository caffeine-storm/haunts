package logging_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/glog"
	"github.com/stretchr/testify/assert"
)

const canary = "lololololololol"

func TestLoggingBracket(t *testing.T) {
	t.Run("at error blocks trace", func(t *testing.T) {
		buf := &bytes.Buffer{}
		fixup := logging.Redirect(buf)

		logging.Bracket(slog.LevelError, func() {
			logging.Trace(canary)
		})

		fixup()

		assert.NotContains(t, buf.String(), canary)
	})

	t.Run("at trace emits info", func(t *testing.T) {
		buf := &bytes.Buffer{}
		fixup := logging.Redirect(buf)

		logging.Bracket(glog.LevelTrace, func() {
			logging.Info(canary)
		})

		fixup()

		assert.Contains(t, buf.String(), canary)
	})
}

func TestTraceBracket(t *testing.T) {
	buf := &bytes.Buffer{}
	fixup := logging.Redirect(buf)

	logging.TraceBracket(func() {
		logging.Trace(canary)
	})

	fixup()

	assert.Contains(t, buf.String(), canary)
}
