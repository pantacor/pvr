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
	"strings"
	"testing"
)

func TestCompileTemplate(t *testing.T) {
	t.Run("docker cmd", testLXCContainerConf__Docker_Cmd)
	t.Run("docker cmd2", testLXCContainerConf__Docker_Cmd2)
	t.Run("docker entrypoint", testLXCContainerConf__Docker_Entrypoint)
	t.Run("docker entrypoint2", testLXCContainerConf__Docker_Entrypoint2)
	t.Run("docker entrypoint3", testLXCContainerConf__Docker_Entrypoint3)
	t.Run("docker entrypointcmd", testLXCContainerConf__Docker_EntrypointCmd)
	t.Run("docker entrypointcmd", testLXCContainerConf__Docker_EntrypointCmd2)
	t.Run("docker workingdir", testLXCContainerConf__Docker_WorkingDir)
	t.Run("PV_IMPORT_VOLUMES LXC_CONTAINER_CONF", testLXCContainerConf_PV_VOLUME_IMPORTS)
}

func ErrorIf(t *testing.T, test bool, args ...interface{}) {
	t.Helper()
	if test {
		t.Error(args...)
	}
}
func ErrorIfNot(t *testing.T, test bool, args ...interface{}) {
	t.Helper()
	if !test {
		t.Error(args...)
	}
}

func testLXCContainerConf__Docker_Cmd(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Cmd": "run.sh",
		},
	})

	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh"), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_Cmd2(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Cmd": []interface{}{"run.sh", "something"},
		},
	})
	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh \"something\""), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_Entrypoint(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Entrypoint": "run.sh",
		},
	})

	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh"), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_Entrypoint2(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Entrypoint": []interface{}{"run.sh"},
		},
	})

	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh"), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_Entrypoint3(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Entrypoint": []interface{}{"run.sh", "something"},
		},
	})

	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh \"something\""), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_EntrypointCmd(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Entrypoint": []interface{}{"run.sh", "something"},
			"Cmd":        []interface{}{"runit"},
		},
	})

	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh \"something\" \"runit\""), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_EntrypointCmd2(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"Entrypoint": []interface{}{"run.sh", "something"},
			"Cmd":        "runit",
		},
	})

	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cmd = run.sh \"something\" \"runit\""), "pattern not found in generated string "+string(result))
}

func testLXCContainerConf__Docker_WorkingDir(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Docker": map[string]interface{}{
			"WorkingDir": "/foo",
		},
	})
	ErrorIf(t, len(result) == 0, "error when compiling template")
	ErrorIfNot(t, strings.Contains(string(result), "lxc.init.cwd = /foo"), "pattern not found in generated string "+string(result))
}

// Source.args.PV_VOLUME_IMPORTS: [ <import1>, <import2>, ... ]
//
//	import: <container>:[<subdir>@]<originvolume>:<destdir>[:<mountflags>]
func testLXCContainerConf_PV_VOLUME_IMPORTS(t *testing.T) {
	result, _ := compileTemplate(LXC_CONTAINER_CONF, map[string]interface{}{
		"name": "container1",
		"Source": map[string]interface{}{
			"args": map[string]interface{}{
				"PV_VOLUME_IMPORTS": []string{
					"cont:/foo/fol:/faldara",
					"cont:subdir@/foo/fol:/faldara-sub",
					"cont:subdir@/foo/fol:/faldara-sub:ro",
				},
			},
		},
	})
	if len(result) == 0 {
		t.Error("error when compiling template with empty data")
	}

	if !strings.Contains(string(result), "lxc.mount.entry = /volumes/cont/docker--foo-fol faldara none bind,rw,create=dir 0 0") {
		t.Error("mount entry not set properly, but: " + string(result))
	}

	if !strings.Contains(string(result), "lxc.mount.entry = /volumes/cont/docker--foo-fol/subdir faldara-sub none bind,rw,create=dir 0 0") {
		t.Error("mount entry not set properly, but: " + string(result))
	}

	if !strings.Contains(string(result), "lxc.mount.entry = /volumes/cont/docker--foo-fol/subdir faldara-sub none bind,ro,create=dir 0 0") {
		t.Error("mount entry not set properly, but: " + string(result))
	}
}
