//
// Copyright 2017-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package templates

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/leekchan/gtf"
)

func prefixFuncMap(funcMap map[string]interface{}, prefix string, keep bool) map[string]interface{} {
	newMap := map[string]interface{}{}
	for k, v := range funcMap {
		if keep {
			newMap[k] = v
		}
		newMap[fmt.Sprintf("%s_%s", prefix, k)] = v
	}
	return newMap
}

func compileTemplate(content string, values map[string]interface{}) (result []byte, err error) {
	buffer := bytes.NewBuffer(result)
	templ := template.Must(template.New("compiled-template").
		Funcs(prefixFuncMap(sprig.TxtFuncMap(), "sprig", true)).
		Funcs(prefixFuncMap(gtf.GtfTextFuncMap, "gtf", false)).
		Funcs(prefixFuncMap(PvrFuncMap(), "pvr", false)).
		Parse(content))
	err = templ.Execute(buffer, values)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
