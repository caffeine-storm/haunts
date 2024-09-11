//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var outputTemplate = template.Must(template.New("output").Parse(outputTemplateStr))

const outputTemplateStr = `
package main
func Version() string {
  return "{{.}}"
}
`

func main() {
	headBytes, err := os.ReadFile(filepath.Join("..", ".git", "HEAD"))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	head := strings.TrimSpace(string(headBytes))

	target := filepath.Join("..", "GEN_version.go")
	os.Remove(target) // Don't care about errors on this one
	f, err := os.Create(target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	outputTemplate.Execute(f, head)
	f.Close()
}
