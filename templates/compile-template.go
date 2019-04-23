package templates

import (
	"bytes"
	"text/template"
)

func compileTemplate(content string, values map[string]interface{}) (result []byte) {
	buffer := bytes.NewBuffer(result)
	templ := template.Must(template.New("compiled-template").Parse(content))
	templ.Execute(buffer, values)
	return buffer.Bytes()
}
