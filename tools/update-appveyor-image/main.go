package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func GetLatestImageHash() (string, error) {
	// - Look at github.com/caffeine-storm/haunts-custom-build-image for the latest
	// commit.
	resp, err := http.Get("https://api.github.com/repos/caffeine-storm/haunts-custom-build-image/branches/main")
	if err != nil {
		return "", fmt.Errorf("couldn't GET branch object: %w", err)
	}
	defer resp.Body.Close()

	var mapping map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&mapping)
	if err != nil {
		return "", fmt.Errorf("couldn't deocde payload: %w", err)
	}

	commitVal, ok := mapping["commit"]
	if !ok {
		return "", fmt.Errorf("no 'commit' field: %v", mapping)
	}

	commit, ok := commitVal.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("bad type in payload: expected object, got %T", commitVal)
	}

	shaValue, ok := commit["sha"]
	if !ok {
		return "", fmt.Errorf("no 'sha' field: %v", commit)
	}

	shaString, ok := shaValue.(string)
	if !ok {
		return "", fmt.Errorf("expecting string for sha, got %T", shaValue)
	}

	return shaString, nil
}

func UpdateHash(oldLine, newHash string) string {
	prefix, _, ok := strings.Cut(oldLine, "=")
	if !ok {
		panic(fmt.Errorf("bad line %q", oldLine))
	}
	return prefix + "=" + newHash
}

// patch $haunts/appveyor.yml to reference that hash in its 'install' script.
func Patch(configFile, newHash string) (string, error) {
	// Don't bother parsing the .yml; just find/replace the line
	in, err := os.Open(configFile)
	if err != nil {
		return "", fmt.Errorf("couldn't os.Open %q: %w", configFile, err)
	}
	defer in.Close()

	newFile := configFile + ".new"
	out, err := os.Create(newFile)
	if err != nil {
		return "", fmt.Errorf("couldn't os.Create %q: %w", newFile, err)
	}
	defer out.Close()

	scanner := bufio.NewScanner(in)
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "- export HASH=") {
			line = UpdateHash(line, newHash)
		}
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return "", fmt.Errorf("WriteString failed: %w", err)
		}
	}

	return newFile, nil
}

func main() {
	configPath := os.Args[1]

	commitHash, err := GetLatestImageHash()
	if err != nil {
		panic(err)
	}

	newFile, err := Patch(configPath, commitHash)
	if err != nil {
		panic(err)
	}

	err = os.Rename(newFile, configPath)
	if err != nil {
		panic(err)
	}
}
