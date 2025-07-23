package updatebuildlib_test

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	updatebuildlib "github.com/MobRulesGames/haunts/tools/update-buildlib"
)

func testdataFile(key string) string {
	return filepath.Join("testdata", key)
}

func readTreeFromTestdata(key string) *updatebuildlib.Tree {
	path := testdataFile(key)

	var f io.ReadCloser
	f, err := os.Open(path)
	if err != nil {
		panic(fmt.Errorf("couldn't os.Open(%q): %w", path, err))
	}
	defer f.Close()

	if strings.HasSuffix(path, ".gz") {
		f, err = gzip.NewReader(f)
		if err != nil {
			panic(fmt.Errorf("couldn't gunzip(%q): %w", path, err))
		}
	}

	data, err := io.ReadAll(f)
	if err != nil {
		panic(fmt.Errorf("couldn't io.ReadAll(%q): %w", path, err))
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

	t.Run("'upstream' does not match 'local'", func(t *testing.T) {
		tlocal := readTreeFromTestdata("rev1.tar.gz")
		tremote := readTreeFromTestdata("rev2.tar.gz")

		if tlocal.Matches(tremote) {
			t.Fatalf("different trees should look different")
		}
	})
}

func TestMakeFromDirectory(t *testing.T) {
	dirpath := testdataFile("extracted")
	tFromDir := updatebuildlib.MakeTreeFromDirectory(dirpath)
	tFromFile := readTreeFromTestdata("rev1.tar.gz")

	delta := tFromDir.Diff(tFromFile)
	if delta != "" {
		t.Fatalf("the 'extracted' directory should match the tarball it came from but got a diff: %q", delta)
	}
}
