package updatebuildlib_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	updatebuildlib "github.com/MobRulesGames/haunts/tools/update-buildlib"
)

func testdataFile(key string) string {
	return filepath.Join("testdata", key)
}

func readTreeFromTestdata(key string) *updatebuildlib.Tree {
	path := testdataFile(key)
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("couldn't os.ReadFile(%q): %w", path, err))
	}

	return updatebuildlib.MakeTreeFromTarball(data)
}

func TestCheckForUpstreamChanges(t *testing.T) {
	t.Run("'upstream' matches 'local'", func(t *testing.T) {
		tlocal := readTreeFromTestdata("rev1.tar.gz")
		tremote := readTreeFromTestdata("rev1.tar.gz")

		if !tlocal.Matches(tremote) {
			t.Fatalf("the same tree should compare equal")
		}
	})
}
