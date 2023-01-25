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

package pvjson

import (
	"bytes"
	"encoding/json"

	cjson "github.com/gibson042/canonicaljson-go"
)

type MarshalOptions struct {
	Canonical bool
}

func Marshal(v interface{}, opts ...MarshalOptions) ([]byte, error) {
	var options MarshalOptions
	if len(opts) == 0 {
		options = MarshalOptions{Canonical: false}
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.Canonical {
		return cjson.Marshal(v)
	}

	return json.MarshalIndent(v, "", "    ")
}

func Unmarshal(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	return nil
}
