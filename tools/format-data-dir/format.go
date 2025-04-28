package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type FmterType int

const (
	IgnoreFmter FmterType = iota
	GlSlFmter
	JsonFmter
	LuaFmter
	XmlFmter
)

var extensionToFmter = map[string]FmterType{
	"":        IgnoreFmter,
	".fev":    IgnoreFmter,
	".game":   IgnoreFmter,
	".gob":    IgnoreFmter,
	".jpg":    IgnoreFmter,
	".log":    IgnoreFmter,
	".player": IgnoreFmter,
	".png":    IgnoreFmter,
	".ttf":    IgnoreFmter,

	".fs":    GlSlFmter, // fragment shaders
	".house": JsonFmter,
	".json":  JsonFmter,
	".lua":   LuaFmter,
	".room":  JsonFmter,
	".vs":    GlSlFmter, // vertex shaders
	".xgml":  XmlFmter,
}

func haveFlag(flagname string) bool {
	for _, arg := range os.Args {
		if arg == flagname {
			return true
		}
	}

	return false
}

type Fmter func(string) (bool, error)

func nopfmter(string) (bool, error) {
	return false, nil
}

func notimplemented(tp string) Fmter {
	return func(s string) (bool, error) {
		fmt.Fprintf(os.Stderr, "warning: %q not implemented\n", tp)
		return false, nil
	}
}

func glslfmt(readOnly bool) Fmter {
	return notimplemented("glslfmt")
}

func jsonfmt(readOnly bool) Fmter {
	return notimplemented("jsonfmt")
}

func luafmt(readOnly bool) Fmter {
	return notimplemented("luafmt")
}

func xmlfmt(readOnly bool) Fmter {
	return notimplemented("xmlfmt")
}

func getfmter(tp FmterType, readOnly bool) Fmter {
	switch tp {
	case IgnoreFmter:
		return nopfmter
	case GlSlFmter:
		return glslfmt(readOnly)
	case JsonFmter:
		return jsonfmt(readOnly)
	case LuaFmter:
		return luafmt(readOnly)
	case XmlFmter:
		return xmlfmt(readOnly)
	default:
		panic(fmt.Errorf("unknown FmterType: %v", tp))
	}
}

// like 'go fmt' but for things under 'data/'
func main() {
	readOnly := true
	changeSet := []string{}

	if !haveFlag("--check") {
		readOnly = false
	}

	datadir := "data/"
	dirfile, err := os.Stat(datadir)
	if err != nil {
		panic(fmt.Errorf("os.Stat(%q) failed: %w", datadir, err))
	}
	if !dirfile.IsDir() {
		panic(fmt.Errorf("data directory (%q) isn't a directory", datadir))
	}

	// foreach file in data/
	fs.WalkDir(os.DirFS(datadir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If we couldn't read something under 'data/' fail hard
			return err
		}

		// Skip directories that we can't read
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Recurse into directories we can read
		if info.IsDir() {
			return nil
		}

		// We're dealing with a regular file. Grab its extension.
		ext := filepath.Ext(path)

		// if its extension is 'targeted'
		fmtertype, found := extensionToFmter[ext]
		if !found {
			// there's a file extension that we don't know about
			panic(fmt.Errorf("unknown extension %q for file %q", ext, path))
		}

		// run a formatter for that extension
		fmter := getfmter(fmtertype, readOnly)
		changeWanted, err := fmter(path)
		if err != nil {
			return err
		}
		if changeWanted {
			changeSet = append(changeSet, path)
		}

		return nil
	})

	for _, change := range changeSet {
		fmt.Printf("%s\n", change)
	}

	if readOnly && len(changeSet) > 0 {
		os.Exit(1)
	}
}
