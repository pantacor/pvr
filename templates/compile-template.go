package templates

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
)

func compileTemplate(content string, values map[string]interface{}) (result []byte) {
	buffer := bytes.NewBuffer(result)
	templ := template.Must(template.New("compiled-template").Funcs(sprig.TxtFuncMap()).Parse(content))
	templ.Execute(buffer, values)
	return buffer.Bytes()
}
