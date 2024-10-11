package updateglop

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseRev(goGetCommandData string) (string, error) {
	line, _, found := strings.Cut(goGetCommandData, ": parsing go.mod")
	if !found {
		return "", fmt.Errorf("couldn't strings.Cut input")
	}

	target := "glop@"
	rev := line[strings.LastIndex(line, target)+len(target):]

	// rev should look like 'v0.0.0-20241010144107-015f009dded2'
	matched, err := regexp.Match(`v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+-[[:digit:]]{14}-[[:xdigit:]]{12}`, []byte(rev))
	if err != nil {
		panic(fmt.Errorf("regex error: %v", err))
	}

	if !matched {
		panic(fmt.Errorf("input parsing failed to match %q against a version", rev))
	}

	return rev, nil
}
