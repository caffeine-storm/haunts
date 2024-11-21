package updateglop_test

import (
	"os"
	"testing"

	updateglop "github.com/MobRulesGames/haunts/tools/update-glop"
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
	t.Run("CanParseNewDownlad", func(t *testing.T) {
		testdata, err := os.ReadFile("testdata/get-output-2.txt")
		if err != nil {
			panic(err)
		}

		rev, err := updateglop.ParseRev(string(testdata))
		if err != nil {
			t.Fatalf("parseRev failed: %v", err)
		}

		expectedRev := "v0.0.0-20241101164606-3ab444d2b0a1"
		if rev != expectedRev {
			t.Fatalf("expected %q but got %q", expectedRev, rev)
		}
	})
}
