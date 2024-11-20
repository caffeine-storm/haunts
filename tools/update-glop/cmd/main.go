package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	updateglop "github.com/caffeine-storm/update-glop"
)

func main() {
	// - Run 'go get -u github.com/caffeine-storm/glop'
	// - It fails, but prints a revision we want.
	// - Use 'go mod edit -replace=$glop-repo=$glop-repo@new-ver' to edit Haunts'
	// go.mod file

	oldRepo := "github.com/runningwild/glop"
	cstormRepo := "github.com/caffeine-storm/glop"
	cmd := exec.Command("go", "get", "-u", cstormRepo+"@latest")

	// Defeat the default cache at proxy.golang.org or else we can get stale
	// results.
	cmd.Env = append(os.Environ(), "GONOPROXY="+cstormRepo)

	data, err := cmd.CombinedOutput()
	// We _EXPECT_ the go get to fail
	if err == nil {
		panic(fmt.Errorf("'go get ...' didn't fail?"))
	}

	rev, err := updateglop.ParseRev(string(data))
	if err != nil {
		panic(fmt.Errorf("couldn't ParseRev: %v", err))
	}

	replaceStr := fmt.Sprintf("-replace=%s=%s@%s", oldRepo, cstormRepo, rev)
	target, err := filepath.Abs("../../go.mod")
	if err != nil {
		panic(fmt.Errorf("couldn't abspath: %w", err))
	}
	fmt.Printf("targeting go.mod file: %q\n", path.Clean(target))
	cmd = exec.Command("go", "mod", "edit", replaceStr, "../../go.mod")
	fmt.Printf("running command: %v\n", cmd)
	data, err = cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("couldn't run 'go mod edit...': %v\n%s", err, data))
	}

	fmt.Printf("don't forget to run 'go mod tidy'!\n")
}
