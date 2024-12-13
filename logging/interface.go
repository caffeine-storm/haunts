package logging

import (
	"bytes"
	"context"
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
	log.Logger.Log(context.Background(), slog.LevelInfo, msg, args...)
}

var _ Logger = (*hauntsLogger)(nil)

var debugLogger *hauntsLogger
var infoLogger *hauntsLogger
var warnLogger *hauntsLogger
var errorLogger *hauntsLogger

func init() {
	debugLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelDebug,
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
	DefaultLogger().Info(msg, args...)
}

func Debug(msg string, args ...interface{}) {
	DebugLogger().Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	InfoLogger().Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	WarnLogger().Info(msg, args...)
}

func Error(msg string, args ...interface{}) {
	ErrorLogger().Error(msg, args...)
}

// Call this to redirect all logging output to the given io.Writer. A cleanup
// function that undoes the redirect is returned.
func Redirect(newOut io.Writer) func() {
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

type gloggy = glog.Logger
type baseLogger struct {
	*log.Logger
	gloggy
}

func doLog(lvl slog.Level, msg string, args ...interface{}) {
	if !infoLogger.Enabled(context.Background(), lvl) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, <helper>, doLog]
	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	r.Add(args...)
	infoLogger.Handler().Handle(context.Background(), r)
}

// Equivalent to glog.InfoLogger().Error
func (*baseLogger) Error(msg string, args ...interface{}) {
	doLog(slog.LevelError, msg, args...)
}

// Equivalent to glog.InfoLogger().Warn
func (*baseLogger) Warn(msg string, args ...interface{}) {
	doLog(slog.LevelWarn, msg, args...)
}

// Equivalent to glog.InfoLogger().Info
func (*baseLogger) Info(msg string, args ...interface{}) {
	doLog(slog.LevelInfo, msg, args...)
}

// Equivalent to glog.InfoLogger().Debug
func (*baseLogger) Debug(msg string, args ...interface{}) {
	doLog(slog.LevelDebug, msg, args...)
}

// Equivalent to glog.InfoLogger().Trace
func (*baseLogger) Trace(msg string, args ...interface{}) {
	doLog(glog.LevelTrace, msg, args...)
}

func SetupLogger(logSink io.Writer) *bytes.Buffer {
	logConsole := &bytes.Buffer{}
	logWriter := io.MultiWriter(logConsole, logSink)

	debugLogger.Logger = glog.WithRedirect(debugLogger.Logger, logWriter)
	infoLogger.Logger = glog.WithRedirect(infoLogger.Logger, logWriter)
	warnLogger.Logger = glog.WithRedirect(warnLogger.Logger, logWriter)
	errorLogger.Logger = glog.WithRedirect(errorLogger.Logger, logWriter)

	return logConsole
}

// Tells the 'Default Logger' to changes its verbosity.
func SetLogLevel(lvl slog.Level) {
	infoLogger.Logger = glog.Relevel(infoLogger.Logger, lvl)
}
