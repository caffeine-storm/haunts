package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

func getCheckFlag(flagname string) bool {
	for idx, arg := range os.Args[1:] {
		if arg == flagname {
			copy(os.Args[idx:], os.Args[idx+1:])
			newlen := len(os.Args) - 1
			os.Args = os.Args[:newlen]
			return true
		}
	}

	return false
}

// Returns an error if something went wrong. Returns true if the formatter
// suggests/has-applied changes.
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
	return func(path string) (bool, error) {
		f, err := os.Open(path)
		if err != nil {
			return false, fmt.Errorf("couldn't os.Open %q: %w", path, err)
		}
		defer f.Close()

		contents, err := io.ReadAll(f)
		if err != nil {
			return false, fmt.Errorf("couldn't io.ReadAll %q: %w", path, err)
		}

		var v any
		json.Unmarshal(contents, &v)
		formatted, err := json.MarshalIndent(v, "", "    ")
		if err != nil {
			return false, fmt.Errorf("couldn't json.Indent %q: %w", path, err)
		}
		if len(formatted) > 0 {
			// make sure there's a trailing newline
			if formatted[len(formatted)-1] != '\n' {
				formatted = append(formatted, '\n')
			}
		}

		diff := !bytes.Equal(contents, formatted)
		if readOnly || !diff {
			// If we shouldn't change anything or if there's nothing to change, we're
			// done.
			return diff, nil
		}

		// Otherwise, rewrite the input file with the indented version.
		replacement, err := os.Create(path)
		if err != nil {
			return diff, fmt.Errorf("couldn't os.Create %q: %w", path, err)
		}
		n, err := io.Copy(replacement, bytes.NewReader(formatted))
		if err != nil {
			return diff, fmt.Errorf("couldn't io.Copy to %q: %w", path, err)
		}
		if int(n) != len(formatted) {
			return diff, fmt.Errorf("incomplete write(%d): %q", n, path)
		}

		return diff, err
	}
}

func luafmt(readOnly bool) Fmter {
	return func(path string) (bool, error) {
		args := []string{}
		if readOnly {
			args = append(args, "--check")
		}
		args = append(args, path)

		cmd := exec.Command("stylua", args...)
		err := cmd.Run()

		// We passed '--check' and got a non-zero return so fmt it!
		if readOnly && err != nil {
			return true, nil
		}

		return false, err
	}
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
	readOnly := getCheckFlag("--check")

	ok := true
	for _, arg := range os.Args[1:] {
		ok = processFile(arg, readOnly) && ok
	}

	if !ok {
		os.Exit(1)
	}
}

func processFile(targetPath string, readOnly bool) bool {
	ext := filepath.Ext(targetPath)
	fmterType, found := extensionToFmter[ext]
	if !found {
		panic(fmt.Errorf("uknown extension(%s) for file %q", ext, targetPath))
	}

	if fmterType == IgnoreFmter {
		return true
	}

	// get and run a formatter for that extension
	fmter := getfmter(fmterType, readOnly)

	changeWanted, err := fmter(targetPath)
	if err != nil {
		panic(fmt.Errorf("formatting %q failed: %w", targetPath, err))
	}

	if changeWanted {
		fmt.Printf("%s\n", targetPath)

		if readOnly {
			return false
		}
	}

	return true
}
