package base_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/MobRulesGames/haunts/base"
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
	return fmt.Sprintf("did not find %q amongst output", target)
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
	Convey("the source attribute in a log message", func() {
		logOutput := base.SetupLogger("../testdata")
		base.Log().Info("a test message")

		Convey("should reference the client code", func() {
			So(logOutput, ShouldReference, "base/utils_test.go")
		})
	})
}

func TestLogging(t *testing.T) {
	Convey("base.{Info,Warn,Error} specification", t, LoggingSpec)
}
