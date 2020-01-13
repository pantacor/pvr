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
	"strings"
	"testing"
)

func TestCompileTemplate(t *testing.T) {
	t.Run("empty LXC_CONTAINER_CONF", testLXCContainerConf)
}

func testLXCContainerConf(t *testing.T) {
	result := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"Docker": map[string]interface{}{
			"WorkingDir": "/foo",
		},
	})
	if len(result) == 0 {
		t.Error("error when compiling template with empty data")
	}

	if !strings.Contains(string(result), "lxc.init.cwd = /foo") {
		t.Error("docker working dir was not found in generated string")
	}
}
