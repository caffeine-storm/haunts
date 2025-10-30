package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"runtime"
	"time"

	"github.com/runningwild/glop/glog"
)

type stdLogInterceptor interface {
	Printf(format string, v ...interface{})
}

type Logger interface {
	glog.Logger
	stdLogInterceptor
}

type hauntsLogger struct {
	glog.Logger
}

func (log *hauntsLogger) Printf(msg string, args ...interface{}) {
	doLog(log, slog.LevelInfo, fmt.Sprintf(msg, args...))
}

var _ Logger = (*hauntsLogger)(nil)

var (
	traceLogger *hauntsLogger
	debugLogger *hauntsLogger
	infoLogger  *hauntsLogger
	warnLogger  *hauntsLogger
	errorLogger *hauntsLogger
)

func init() {
	traceLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelInfo,
		}),
	}
	debugLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelInfo,
		}),
	}
	infoLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelInfo,
		}),
	}
	warnLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelWarn,
		}),
	}
	errorLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelError,
		}),
	}
}

func DefaultLogger() Logger {
	return InfoLogger()
}

func TraceLogger() Logger {
	return traceLogger
}

func DebugLogger() Logger {
	return debugLogger
}

func InfoLogger() Logger {
	return infoLogger
}

func WarnLogger() Logger {
	return warnLogger
}

func ErrorLogger() Logger {
	return errorLogger
}

func Log(msg string, args ...interface{}) {
	doLog(infoLogger, slog.LevelInfo, msg, args...)
}

func Trace(msg string, args ...interface{}) {
	doLog(traceLogger, glog.LevelTrace, msg, args...)
}

func Debug(msg string, args ...interface{}) {
	doLog(debugLogger, slog.LevelDebug, msg, args...)
}

func Info(msg string, args ...interface{}) {
	doLog(infoLogger, slog.LevelInfo, msg, args...)
}

func Warn(msg string, args ...interface{}) {
	doLog(warnLogger, slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...interface{}) {
	doLog(errorLogger, slog.LevelError, msg, args...)
}

// Like 'Redirect' but also return a duped reader for watching log output.
func RedirectAndSpy(out io.Writer) (func(), *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	cleanup := Redirect(io.MultiWriter(out, buffer))

	return cleanup, buffer
}

// Call this to redirect all logging output to the given io.Writer. A cleanup
// function that undoes the redirect is returned.
func Redirect(newOut io.Writer) func() {
	oldTraceLogger := traceLogger
	traceLogger = &hauntsLogger{
		Logger: glog.WithRedirect(oldTraceLogger, newOut),
	}

	oldDebugLogger := debugLogger
	debugLogger = &hauntsLogger{
		Logger: glog.WithRedirect(oldDebugLogger, newOut),
	}

	oldInfoLogger := infoLogger
	infoLogger = &hauntsLogger{
		Logger: glog.WithRedirect(oldInfoLogger, newOut),
	}

	oldWarnLogger := warnLogger
	warnLogger = &hauntsLogger{
		Logger: glog.WithRedirect(oldWarnLogger, newOut),
	}

	oldErrorLogger := errorLogger
	errorLogger = &hauntsLogger{
		Logger: glog.WithRedirect(oldErrorLogger, newOut),
	}
	return func() {
		traceLogger = oldTraceLogger
		debugLogger = oldDebugLogger
		infoLogger = oldInfoLogger
		warnLogger = oldWarnLogger
		errorLogger = oldErrorLogger
	}
}

// Until we migrate lots of old log.Logger calls, we'll keep a log.Logger
// around.
// TODO(tmckee): delegate logging from 'logger' to 'slogger' so that all logs
// are structured/leveled conveniently.

type (
	gloggy     = glog.Logger
	baseLogger struct {
		*log.Logger
		gloggy
	}
)

func doLog(logger *hauntsLogger, lvl slog.Level, msg string, args ...interface{}) {
	if !logger.Enabled(context.Background(), lvl) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, doLog, <helper>]
	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	r.Add(args...)
	logger.Handler().Handle(context.Background(), r)
}

// Equivalent to glog.InfoLogger().Error
func (*baseLogger) Error(msg string, args ...interface{}) {
	doLog(infoLogger, slog.LevelError, msg, args...)
}

// Equivalent to glog.InfoLogger().Warn
func (*baseLogger) Warn(msg string, args ...interface{}) {
	doLog(infoLogger, slog.LevelWarn, msg, args...)
}

// Equivalent to glog.InfoLogger().Info
func (*baseLogger) Info(msg string, args ...interface{}) {
	doLog(infoLogger, slog.LevelInfo, msg, args...)
}

// Equivalent to glog.InfoLogger().Debug
func (*baseLogger) Debug(msg string, args ...interface{}) {
	doLog(infoLogger, slog.LevelDebug, msg, args...)
}

// Equivalent to glog.InfoLogger().Trace
func (*baseLogger) Trace(msg string, args ...interface{}) {
	doLog(infoLogger, glog.LevelTrace, msg, args...)
}

// Tells the 'Default Logger' to changes its verbosity.
func SetDefaultLoggerLevel(lvl slog.Level) {
	infoLogger.Logger = glog.Relevel(infoLogger.Logger, lvl)
}

// Tells loggers to include messages at the given level. Returns an 'undo'
// closure.
func SetLoggingLevel(lvl slog.Level) func() {
	oldTraceLogger := traceLogger.Logger
	oldDebugLogger := debugLogger.Logger
	oldInfoLogger := infoLogger.Logger
	oldWarnLogger := warnLogger.Logger
	oldErrorLogger := errorLogger.Logger

	traceLogger.Logger = glog.Relevel(traceLogger, lvl)
	debugLogger.Logger = glog.Relevel(debugLogger, lvl)
	infoLogger.Logger = glog.Relevel(infoLogger, lvl)
	warnLogger.Logger = glog.Relevel(warnLogger, lvl)
	errorLogger.Logger = glog.Relevel(errorLogger, lvl)

	return func() {
		traceLogger.Logger = oldTraceLogger
		debugLogger.Logger = oldDebugLogger
		infoLogger.Logger = oldInfoLogger
		warnLogger.Logger = oldWarnLogger
		errorLogger.Logger = oldErrorLogger
	}
}
