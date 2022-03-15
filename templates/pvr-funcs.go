//
// Copyright 2019  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//
package templates

import (
	"reflect"
	"strings"
)

var pvrFuncMap = map[string]interface{}{
	"mergePersistentMaps": func(m1 interface{}, m2 interface{}) interface{} {
		mapR := map[string]interface{}{}
		map1 := map[string]interface{}{}
		if m1 != nil {
			map1 = m1.(map[string]interface{})
			for k := range map1 {
				k1 := strings.TrimSuffix(k, "/")
				mapR[k1] = "permanent"
			}
		}
		map2 := map[string]interface{}{}
		if m2 != nil {
			map2 = m2.(map[string]interface{})
			for k, v := range map2 {
				k1 := strings.TrimSuffix(k, "/")
				mapR[k1] = v
			}
		}

		return mapR
	},
	"sliceIndex": func(content []interface{}, i int) interface{} {
		if len(content) > i {
			return content[i]
		}
		return nil
	},
	"sliceFrom": func(content []interface{}, from int) interface{} {
		if len(content) > from {
			return content[from:]
		}
		return nil
	},
	"sliceTo": func(content []interface{}, to int) interface{} {
		if len(content) >= to {
			return content[:to]
		}
		return nil
	},
	"isSlice": func(content interface{}) bool {
		// nil is not a slice for us
		if content == nil {
			return false
		}
		val := reflect.ValueOf(content)
		if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
			return true
		}
		return false
	},
	"ifNull": func(arg interface{}, value interface{}) interface{} {

		if value != nil {
			return value
		}

		return arg
	},
	"jsonIndent": func(first int, rest int, char string, value string) interface{} {
		builder := strings.Builder{}
		lines := strings.Split(value, "\n")
		var isRest bool
		for _, v := range lines {
			var count = rest
			if !isRest {
				count = first
				isRest = true
			} else {
				builder.WriteString("\n")
			}
			for i := 0; i < count; i++ {
				_, _ = builder.WriteString(char)
			}
			builder.WriteString(v)
		}
		return builder.String()
	},
}

func PvrFuncMap() map[string]interface{} {
	sfm := make(map[string]interface{}, len(pvrFuncMap))
	for k, v := range pvrFuncMap {
		sfm[k] = v
	}
	return sfm
}
