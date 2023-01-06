//
// Copyright 2020-2023  Pantacor Ltd.
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
package libpvr

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.com/pantacor/pantahub-base/logs"
)

type LogFormatter interface {
	Init(template string) error
	DoLog(m *logs.Entry) error
}

type LogFormatterJson struct{}

func (s *LogFormatterJson) Init(format string) error {
	return nil
}

func (s *LogFormatterJson) DoLog(m *logs.Entry) error {
	buf, err := json.Marshal(m)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, string(buf))
	return nil
}

type LogFormatterTemplate struct {
	template string
}

func (s *LogFormatterTemplate) Init(template string) error {
	s.template = template
	return nil
}

func (s *LogFormatterTemplate) DoLog(v *logs.Entry) error {
	r, err := SprintTmpl(s.template, v)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, r)
	return nil
}
