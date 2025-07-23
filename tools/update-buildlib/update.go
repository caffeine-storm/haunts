package updatebuildlib

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

type Tree struct {
	data []byte
}

type TreeReader interface {
	Next() (*tar.Header, error)
	io.Reader
}

func MakeTreeFromTarball(tarball []byte) *Tree {
	return &Tree{
		data: tarball,
	}
}

func MakeTreeFromDirectory(dirpath string) *Tree {
	buf := &bytes.Buffer{}
	writer := tar.NewWriter(buf)

	err := writer.AddFS(os.DirFS(dirpath))
	if err != nil {
		panic(fmt.Errorf("couldn't writer.AddFS(%q): %w", dirpath, err))
	}

	err = writer.Close()
	if err != nil {
		panic(fmt.Errorf("couldn't writer.Close(): %w", err))
	}

	return &Tree{
		data: buf.Bytes(),
	}
}

func (t *Tree) Reader() TreeReader {
	return tar.NewReader(bytes.NewBuffer(t.data))
}

func readFileNames(rdr *tar.Reader) []string {
	ret := []string{}
	for {
		hdr, err := rdr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Errorf("couldn't scrape filenames: %w", err))
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		ret = append(ret, hdr.Name)
	}
	return ret
}

type fstat struct {
	name string
	data []byte
}

func readFiles(rdr *tar.Reader) []fstat {
	ret := []fstat{}

	for {
		hdr, err := rdr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Errorf("readFiles: rdr.Next failed: %w", err))
		}

		// We only care about 'regular' files
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		buf := &bytes.Buffer{}
		if _, err = io.Copy(buf, rdr); err != nil {
			panic(fmt.Errorf("readFiles: couldn't extract file contents: %w", err))
		}

		ret = append(ret, fstat{
			name: hdr.Name,
			data: buf.Bytes(),
		})
	}

	return ret
}

func showFileNames(contents []fstat) string {
	filenames := []string{}

	for _, entry := range contents {
		filenames = append(filenames, entry.name)
	}

	return strings.Join(filenames, ", ")
}

func diffFile(lhs, rhs []byte) string {
	ret := []string{}

	minLength := min(len(lhs), len(rhs))
	for i := 0; i < minLength; i++ {
		if lhs[i] != rhs[i] {
			ret = append(ret, fmt.Sprintf("[%d]: %d != %d", i, lhs[i], rhs[i]))
		}
	}

	if len(lhs) != len(rhs) {
		ret = append([]string{"size mismatch"}, ret...)
	}

	return strings.Join(ret, "\n")
}

func (lhs *Tree) Matches(rhs *Tree) bool {
	return lhs.Diff(rhs) == ""
}

func (lhs *Tree) Diff(rhs *Tree) string {
	lReader := tar.NewReader(bytes.NewReader(lhs.data))
	rReader := tar.NewReader(bytes.NewReader(rhs.data))

	lhsFiles := readFiles(lReader)
	rhsFiles := readFiles(rReader)

	lhsFileList := showFileNames(lhsFiles)
	rhsFileList := showFileNames(rhsFiles)

	if lhsFileList != rhsFileList {
		return fmt.Sprintf("structural difference: \n%s\n%s", lhsFileList, rhsFileList)
	}

	problems := []string{}
	for idx, lhsFile := range lhsFiles {
		rhsFile := rhsFiles[idx]

		if problem := diffFile(lhsFile.data, rhsFile.data); problem != "" {
			problems = append(problems, problem)
		}
	}

	return strings.Join(problems, "\n")
}
