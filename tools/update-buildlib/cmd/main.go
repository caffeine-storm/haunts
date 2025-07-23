package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	updatebuildlib "github.com/MobRulesGames/haunts/tools/update-buildlib"
)

func makeTargetPath(basedir, tarname string) string {
	// Because github
	prefix := "buildlib-main"
	withoutPrefix, ok := strings.CutPrefix(tarname, prefix)
	if !ok {
		panic(fmt.Errorf("couldn't cut-prefix of %q from %q", prefix, tarname))
	}
	return filepath.Join(basedir, withoutPrefix)
}

func main() {
	upstreamUrl := "https://github.com/caffeine-storm/buildlib/archive/refs/heads/main.tar.gz"
	localBuildlibDir := "build"

	c := http.Client{}
	resp, err := c.Get(upstreamUrl)
	if err != nil {
		panic(fmt.Errorf("couldn't GET %q: %w", upstreamUrl, err))
	}
	defer resp.Body.Close()

	inflated, err := gzip.NewReader(resp.Body)
	if err != nil {
		panic(fmt.Errorf("couldn't gunzip: %q: %w", upstreamUrl, err))
	}

	data, err := io.ReadAll(inflated)
	if err != nil {
		panic(fmt.Errorf("couldn't io.ReadAll: %q: %w", upstreamUrl, err))
	}
	upstreamTree := updatebuildlib.MakeTreeFromTarball(data)
	localVersion := updatebuildlib.MakeTreeFromDirectory(localBuildlibDir)

	if localVersion.Matches(upstreamTree) {
		return
	}

	// Otherwise, extract the upstream tarball over the local directory.
	tarReader := upstreamTree.Reader()
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}

		if err != nil {
			log.Fatalf("tarReader.Next() failed: %v", err)
		}

		if hdr.Typeflag != tar.TypeReg {
			log.Printf("debug: skipping entry of type# %v", hdr.Typeflag)
			continue
		}

		targetPath := makeTargetPath(localBuildlibDir, hdr.Name)

		log.Printf("extracting %q", targetPath)
		output, err := os.Create(targetPath)
		if err != nil {
			log.Printf("error: couldn't os.Create(%q): %v", targetPath, err)
			continue
		}
		if _, err := io.Copy(output, tarReader); err != nil {
			log.Printf("error: io.Copy(%q) failed: %v", targetPath, err)
		}
	}
}
