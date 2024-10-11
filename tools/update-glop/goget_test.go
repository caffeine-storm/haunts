package updateglop_test

import (
	"os"
	"testing"

	updateglop "github.com/caffeine-storm/update-glop"
)

func TestGoGetCommand(t *testing.T) {
	t.Run("CanParseRev", func(t *testing.T) {
		testdata, err := os.ReadFile("testdata/get-output.txt")
		if err != nil {
			panic(err)
		}

		rev, err := updateglop.ParseRev(string(testdata))
		if err != nil {
			t.Fatalf("parseRev failed: %v", err)
		}

		expectedRev := "v0.0.0-20241010144107-015f009dded2"
		if rev != expectedRev {
			t.Fatalf("expected %q but got %q", expectedRev, rev)
		}
	})
}
