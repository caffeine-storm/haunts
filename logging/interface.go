package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"

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

var errorLogger Logger

func init() {
	errorLogger = &hauntsLogger{
		Logger: glog.New(&glog.Opts{
			Level: slog.LevelError,
		}),
	}
	fmt.Printf("errorLogger: %v\n", errorLogger)
}

func ErrorLogger() Logger {
	fmt.Printf("getting error logger!!!\n")
	return errorLogger
}

func Error(msg string, args ...interface{}) {
	ErrorLogger().Error(msg, args...)
}

// Call this to redirect all logging output to the given io.Writer. A cleanup
// function that undoes the redirect is returned.
func Redirect(newOut io.Writer) func() {
	oldLogger := ErrorLogger()
	errorLogger = &hauntsLogger{
		Logger: glog.WithRedirect(oldLogger, newOut),
	}
	return func() {
		errorLogger = oldLogger
	}
}
