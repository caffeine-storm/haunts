package genversion

import (
	"io"
	"text/template"
)

var outputTemplate = template.Must(template.New("output").Parse(outputTemplateStr))

const outputTemplateStr = `package gen

func Version() string {
	return "{{.}}"
}
`

func GenFile(commitHash string, outFile io.Writer) error {
	return outputTemplate.Execute(outFile, commitHash)
}
