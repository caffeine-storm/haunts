package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MobRulesGames/haunts/tools/genversion"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s path/to/.git/HEAD path/to/gen/version.go", os.Args[0])
		os.Exit(1)
	}

	inpath := os.Args[1]
	headBytes, err := os.ReadFile(inpath)
	if err != nil {
		panic(fmt.Errorf("couldn't os.ReadFile(%q): %w", inpath, err))
	}

	// the contents of .git/HEAD might be a raw hash like
	//   c0ffeec0ffec0ffec0ffec0ffec0ffec0ffeec0f
	// or a line like
	//   ref: refs/heads/main'
	var commitHash []byte
	if bytes.HasPrefix(headBytes, []byte("ref: ")) {
		ref := strings.TrimSpace(strings.SplitAfterN(string(headBytes), " ", 2)[1])
		commitHashPath := filepath.Join(filepath.Dir(inpath), string(ref))
		commitHash, err = os.ReadFile(commitHashPath)
		if err != nil {
			panic(fmt.Errorf("couldn't os.ReadFile(%q): %w", commitHashPath, err))
		}
	} else {
		commitHash = headBytes
	}
	commitHash = bytes.TrimSpace(commitHash)

	outpath := os.Args[2]
	targetdir := filepath.Dir(outpath)
	err = os.MkdirAll(targetdir, 0755)
	if err != nil {
		panic(fmt.Errorf("couldn't os.MkdirAll(%q): %w", targetdir, err))
	}

	outFile, err := os.Create(outpath)
	if err != nil {
		panic(fmt.Errorf("couldn't os.Create(%q): %w", outpath, err))
	}

	err = genversion.GenFile(string(commitHash), outFile)
	if err != nil {
		panic(fmt.Errorf("GenFile failed: %w", err))
	}
}
