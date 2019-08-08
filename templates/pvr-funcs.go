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

import "reflect"

var pvrFuncMap = map[string]interface{}{
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
}

func PvrFuncMap() map[string]interface{} {
	sfm := make(map[string]interface{}, len(pvrFuncMap))
	for k, v := range pvrFuncMap {
		sfm[k] = v
	}
	return sfm
}
