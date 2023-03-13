// Copyright 2022  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.
package libpvr

import (
	"strings"

	jsonpatch "github.com/asac/json-patch"
	cjson "github.com/gibson042/canonicaljson-go"
)

func copyState(state map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range state {
		result[k] = v
	}
	return result
}

func FilterByFrags(state map[string]interface{}, frags string) (map[string]interface{}, error) {
	pJSONMap := copyState(state)

	partPrefixes := []string{}
	unpartPrefixes := []string{}
	if frags != "" {
		parsePrefixes := strings.Split(frags, ",")
		for _, v := range parsePrefixes {
			if !strings.HasPrefix(v, "-") {
				partPrefixes = append(partPrefixes, v)
			} else {
				unpartPrefixes = append(unpartPrefixes, v[1:])
			}
		}
	}

	for k := range pJSONMap {
		found := true
		for _, v := range partPrefixes {
			if k == v {
				found = true
				break
			}
			if !strings.HasSuffix(v, "/") {
				v += "/"
			}
			if strings.HasPrefix(k, v) {
				found = true
				break
			} else {
				found = false
			}
		}
		if !found {
			delete(pJSONMap, k)
		}
	}

	for _, partPrefix := range partPrefixes {
		// remove all files for name "app/"
		for k := range pJSONMap {
			if strings.HasPrefix(k, partPrefix) {
				delete(pJSONMap, k)
			}
		}

		// add back all from new map
		for k, v := range state {
			if strings.HasPrefix(k, partPrefix) {
				pJSONMap[k] = v
			}
		}
	}

	for _, unpartPrefix := range unpartPrefixes {
		// remove all files for name "app/"
		for k := range pJSONMap {
			if strings.HasPrefix(k, unpartPrefix) {
				delete(pJSONMap, k)
			}
		}
	}

	return pJSONMap, nil
}

func OverwriteState(state PvrMap, newState PvrMap) {
	for key := range state {
		delete(state, key)
	}
	for key, value := range newState {
		state[key] = value
	}
}

// PatchState update a json with a patch
func PatchState(srcBuff, patchBuff []byte, srcFrags, patchFrag string, merge bool, state *PvrMap) ([]byte, map[string]interface{}, error) {
	var srcState PvrMap
	var patchState PvrMap

	err := cjson.Unmarshal(srcBuff, &srcState)
	if err != nil {
		return nil, nil, err
	}

	err = cjson.Unmarshal(patchBuff, &patchState)
	if err != nil {
		return nil, nil, err
	}

	if len(patchState) == 0 && len(strings.Split(patchFrag, ",")) == len(strings.Split(srcFrags, ",")) {
		patchState, err = renameKeys(srcState, srcFrags, patchFrag)
		if err != nil {
			return nil, nil, err
		}
	}

	patchState, err = FilterByFrags(patchState, patchFrag)
	if err != nil {
		return nil, nil, err
	}

	var jsonMerged []byte
	if merge {
		pJSONMap, err := FilterByFrags(srcState, srcFrags)
		if err != nil {
			return nil, nil, err
		}

		srcJsonMap, err := cjson.Marshal(pJSONMap)
		if err != nil {
			return nil, nil, err
		}

		jsonDataSelect, err := cjson.Marshal(patchState)
		if err != nil {
			return nil, nil, err
		}

		jsonMerged, err = jsonpatch.MergePatch(srcJsonMap, jsonDataSelect)
		if err != nil {
			return nil, nil, err
		}
	} else {
		jsonMerged, err = cjson.Marshal(patchState)
		if err != nil {
			return nil, nil, err
		}
	}

	err = cjson.Unmarshal(jsonMerged, &patchState)
	if state != nil {
		OverwriteState(*state, patchState)
	}

	return jsonMerged, patchState, err
}

func renameKeys(state map[string]interface{}, srcFrags, patchFrag string) (map[string]interface{}, error) {
	pJSONMap := map[string]interface{}{}
	srcPrefixes := strings.Split(srcFrags, ",")
	pathPrefixes := strings.Split(patchFrag, ",")

	for index, key := range srcPrefixes {
		newKey := pathPrefixes[index]
		pJSONMap[newKey] = state[key]
	}

	return pJSONMap, nil
}
