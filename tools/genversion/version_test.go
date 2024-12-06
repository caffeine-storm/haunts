package genversion_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/MobRulesGames/haunts/tools/genversion"
)

func checkFmt(path string) error {
	cmd := exec.Command("gofmt", "-l", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	if len(output) != 0 {
		fmt.Printf("%s\n", string(output))
		return fmt.Errorf("target needs reformatting: %q", path)
	}
	return nil
}

func TestGeneratedFileIsFormatted(t *testing.T) {
	os.Chdir("../..")
	genversion.GenFile()

	// `gofmt -l ../../GEN_version.go` shouldn't complain
	err := checkFmt("GEN_version.go")
	if err != nil {
		t.Fatalf("checkFmt failed: %v", err)
	}
}
