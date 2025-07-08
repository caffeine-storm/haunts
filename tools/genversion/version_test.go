package genversion_test

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"testing"

	"github.com/MobRulesGames/haunts/tools/genversion"
)

func checkFmt(fileContent io.Reader) error {
	cmd := exec.Command("gofmt", "-l")
	cmd.Stdin = fileContent

	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	if len(output) != 0 {
		fmt.Printf("%s\n", string(output))
		return fmt.Errorf("target needs reformatting")
	}

	return nil
}

func TestGeneratedFileIsFormatted(t *testing.T) {
	buf := &bytes.Buffer{}
	err := genversion.GenFile("c0ffeec0ffeec0ffeec0ffeec0ffeec0ffeec0ff", buf)
	if err != nil {
		panic(fmt.Errorf("couldn't GenFile on 0xc0ffee: %w", err))
	}

	// `gofmt -l <(buf)` shouldn't complain
	err = checkFmt(buf)
	if err != nil {
		t.Fatalf("checkFmt failed: %v", err)
	}
}
