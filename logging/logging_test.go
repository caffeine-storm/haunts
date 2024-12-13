package logging_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/logging/logtesting"
	"github.com/runningwild/glop/glog"
	. "github.com/smartystreets/goconvey/convey"
)

func parseSourceAttr(line string) (string, bool) {
	idx := strings.LastIndex(line, "source=")
	if idx == -1 {
		return "", false
	}

	sourcePlus := line[idx+(len("source=")):]
	parts := strings.SplitN(sourcePlus, ":", 2)
	if len(parts) != 2 {
		return "", false
	}

	return parts[0], true
}

func ShouldContainSourceRef(outputStream io.Reader, target string) string {
	outputBytes, err := io.ReadAll(outputStream)
	if err != nil {
		panic(fmt.Errorf("couldn't io.ReadAll: %w", err))
	}

	outputLines := bytes.Split(outputBytes, []byte("\n"))
	for _, line := range outputLines {
		sourceAttr, found := parseSourceAttr(string(line))
		if !found {
			continue
		}

		if strings.Contains(sourceAttr, target) {
			// Found it!
			return ""
		}
	}

	return fmt.Sprintf("did not find %q amongst output %q", target, bytes.Join(outputLines, []byte{'\n'}))
}

func ShouldReference(actual interface{}, expected ...interface{}) string {
	lineReader, ok := actual.(io.Reader)
	if !ok {
		panic(fmt.Errorf("'actual' had wrong type: want io.Reader, got %T", actual))
	}

	srcRef, ok := expected[0].(string)
	if !ok {
		panic(fmt.Errorf("'expected[0]' had wrong type: want string, got %T", expected[0]))
	}

	return ShouldContainSourceRef(lineReader, srcRef)
}

func LoggingSpec() {
	Convey("using logging through the base package", func() {
		Convey("the source attribute in a log message", func() {
			logOutput := logging.RedirectOutput(io.Discard)
			base.DeprecatedLog().Info("a test message")

			Convey("should reference the client code", func() {
				So(logOutput, ShouldReference, "logging/logging_test.go")
			})
		})
		Convey("should print when running tests", func() {
			lines := logtesting.CollectOutput(func() {
				base.DeprecatedLog().Error("collected message")
			})
			So(strings.Join(lines, "\n"), ShouldContainSubstring, "collected message")
		})
	})

	Convey("using logging directly", func() {
		Convey("the source attribute in a log message", func() {
			logOutput := logging.RedirectOutput(io.Discard)
			logging.Info("a test message")

			Convey("should reference the client code", func() {
				So(logOutput, ShouldReference, "logging/logging_test.go")
			})
		})

		Convey("should print when running tests", func() {
			lines := logtesting.CollectOutput(func() {
				logging.Error("collected message")
			})
			So(strings.Join(lines, "\n"), ShouldContainSubstring, "collected message")
		})

		Convey("tracing should be supported during tests", func() {
			logging.SetLogLevel(glog.LevelTrace)
			lines := logtesting.CollectOutput(func() {
				logging.Trace("a trace message")
			})
			So(strings.Join(lines, "\n"), ShouldContainSubstring, "a trace message")
			Convey("traced messages should be properly attributed", func() {
				So(strings.Join(lines, "\n"), ShouldContainSubstring, "logging/logging_test.go")
			})
		})
	})

	Convey("redirection should be resettable", func() {
		buf1 := &bytes.Buffer{}

		oldErrorLogger := logging.ErrorLogger()
		resetRedirect := logging.Redirect(buf1)

		oldErrorLogger.Error("oldErrorLogger message 1")
		logging.Error("logging.Error() message 1")

		resetRedirect()

		oldErrorLogger.Error("oldErrorLogger message 2")
		logging.Error("logging.Error() message 2")

		// Only 'logging.Error() message 1' should have been sent to buf1
		bufferedOut := buf1.String()
		So(bufferedOut, ShouldContainSubstring, "logging.Error() message 1")
		So(bufferedOut, ShouldNotContainSubstring, "message 2")
		So(bufferedOut, ShouldNotContainSubstring, "oldErrorLogger")
	})
}

func TestLogging(t *testing.T) {
	Convey("base.{Info,Warn,Error} specification", t, LoggingSpec)
}
