package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <victim> <suite>\n", os.Args[0])
	os.Exit(1)
}

func readTestList(fpath string) []string {
	f, err := os.Open(fpath)
	if err != nil {
		panic(fmt.Errorf("couldn't os.Open %q: %w", fpath, err))
	}

	byteSlice, err := io.ReadAll(f)
	if err != nil {
		panic(fmt.Errorf("couldn't io.ReadAll %q: %w", fpath, err))
	}

	lines := bytes.Split(byteSlice, []byte{'\n'})
	ret := []string{}
	for _, line := range lines {
		if len(line) != 0 {
			ret = append(ret, string(line))
		}
	}

	slog.Info("readTestList", "fpath", fpath, "result", ret)
	return ret
}

func hasCrossTalk(victim, suite []string) bool {
	// First, randomize the order of the suite so that we can explore different
	// samplings than just contiguous slices of the original list.
	rand.Shuffle(len(suite), func(i, j int) {
		suite[i], suite[j] = suite[j], suite[i]
	})

	testpattern := strings.Join(append(victim, suite...), "\\|")
	testpattern = strings.ReplaceAll(testpattern, "'", "\\'")
	testrunargs := fmt.Sprintf("testrunargs=-run %s", testpattern)

	// TODO: if we care, we can make the "test-*" recipe a parameter.
	cmd := exec.Command("make", testrunargs, "test-nocache")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	slog.Info("going to run 'make' command", "cmd", cmd)
	err := cmd.Run()
	return err != nil
}

func showTest(parts []string) string {
	return strings.Join(parts, ", ")
}

func reportCrossTalk(victim, suite []string) {
	fmt.Printf("cross talk reduction: %q (%d) coupled with %q (%d) still fails\n", showTest(victim), len(victim), showTest(suite), len(suite))
}

func main() {
	if len(os.Args) != 3 {
		usage()
	}

	victim := os.Args[1]
	suite := os.Args[2]

	victimTests := readTestList(victim)
	suiteTests := readTestList(suite)
	fmt.Printf("attempting cross-talk reduction between %d victims and %d suite-mates\n", len(victimTests), len(suiteTests))

	// Run 'suiteTests' and 'victimTests'
	// if tests fail, assume it's because of cross talk
	//   try again with first half of suite
	//		 fail => first-half again
	//		 success => second-half

	if !hasCrossTalk(victimTests, suiteTests) {
		panic("couldn't repro with original specification")
	}

	for len(suiteTests) > 0 {
		if len(suiteTests) == 1 {
			reportCrossTalk(victimTests, suiteTests)
			return
		}
		slog.Info("looping", "suiteTests", suiteTests)
		suiteLen := len(suiteTests)

		firstHalf, secondHalf := suiteTests[0:suiteLen/2], suiteTests[suiteLen/2:]

		if hasCrossTalk(victimTests, firstHalf) {
			suiteTests = firstHalf
			continue
		}

		if hasCrossTalk(victimTests, secondHalf) {
			suiteTests = secondHalf
			continue
		}

		// We could try to sample differently but, for now, 'suiteTests' is a good
		// candidate.
		reportCrossTalk(victimTests, suiteTests)
		return
	}

	panic("ran out of input!?")
}
