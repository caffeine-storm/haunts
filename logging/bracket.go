package logging

import "log/slog"

// Run 'fn' in a context where log messages at 'lvl' and above are propagated.
func Bracket(lvl slog.Level, fn func()) {
	fixup := SetLoggingLevel(lvl)
	defer fixup()
	fn()
}
