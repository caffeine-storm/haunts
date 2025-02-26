package logging

import (
	"log/slog"

	"github.com/runningwild/glop/glog"
)

// Run 'fn' in a context where log messages at 'lvl' and above are propagated.
func Bracket(lvl slog.Level, fn func()) {
	fixup := SetLoggingLevel(lvl)
	defer fixup()
	fn()
}

func ErrorBracket(fn func()) {
	Bracket(slog.LevelError, fn)
}

func WarnBracket(fn func()) {
	Bracket(slog.LevelWarn, fn)
}

func InfoBracket(fn func()) {
	Bracket(slog.LevelInfo, fn)
}

func DebugBracket(fn func()) {
	Bracket(slog.LevelDebug, fn)
}

func TraceBracket(fn func()) {
	Bracket(glog.LevelTrace, fn)
}
