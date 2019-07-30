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

const (
	LXC_CONTAINER_CONF = `{{ "" -}}
lxc.tty.max = {{ .Source.vars.LXC_TTY_MAX | pvr_ifNull "8" }}
lxc.pty.max = {{ .Source.vars.LXC_PTY_MAX | pvr_ifNull "1024" }}
{{- if .Source.vars.PV_DEBUG_MODE }}
lxc.log.file = {{ .Source.name }}.log
{{- end }}
lxc.cgroup.devices.allow = {{ .Source.vars.LXC_CGROUP_DEVICES_ALLOW | pvr_ifNull "a" }}
lxc.rootfs.path = overlayfs:/volumes/{{- .Source.name -}}/root.squashfs:/volumes/{{- .Source.name -}}/lxc-overlay/upper
lxc.init.cmd =
{{- if .Docker.Entrypoint }}
	{{- if pvr_isSlice .Docker.Entrypoint }}
		{{- if pvr_sliceIndex .Docker.Entrypoint 0 }}
			{{- "" }} {{ pvr_sliceIndex .Docker.Entrypoint 0}}{{ range pvr_sliceFrom .Docker.Entrypoint 1 }} "{{ . }}"{{ end }}
		{{- end }}
	{{- else }}
		{{- "" }} {{ .Docker.Entrypoint }}
	{{- end }}
{{- else }}
	{{- if .Docker.Cmd }}
		{{- if pvr_isSlice .Docker.Cmd }}
			{{- if pvr_sliceIndex .Docker.Cmd 0 }}
				{{- "" }} {{ pvr_sliceIndex .Docker.Cmd 0}}{{ range pvr_sliceFrom .Docker.Cmd 1 }} "{{ . }}"{{ end }}
			{{- end }}
		{{- else }}
			{{- "" }} {{ .Docker.Cmd }}
		{{- end }}
	{{- end }}
{{- end }}
{{- if and (.Docker.WorkingDir) (ne .Docker.WorkingDir "") }}
lxc.init.cwd = {{ .Docker.WorkingDir }}
{{- end }}
{{- if and (not .Docker.Cmd) (not .Docker.Entrypoint) }} /sbin/init{{- end }}
{{- if .Docker.Env }}
	{{- range .Docker.Env }}
lxc.environment = {{ . }}
	{{- end }}
{{- end }}
lxc.namespace.keep = user net ipc
lxc.console.path = none
lxc.mount.auto = proc sys:rw cgroup-full
{{- if .Source.vars.PV_SECURITY_FULLDEV }}
lxc.mount.entry = /dev/ dev none bind,rw,create=dir 0 0
{{- end }}
{{- if .Source.vars.PV_SECURITY_WITH_STORAGE }}
lxc.mount.entry = /storage storage none bind,rw,create=dir 0 0
{{- end }}
lxc.mount.entry = /etc/resolv.conf etc/resolv.conf none bind,rw,create=file 0 0
lxc.mount.entry = tmpfs run tmpfs rw,nodev,relatime,mode=755 0 0
{{- with $src := .Source -}}
{{- range $key, $value := $src.persistence -}}
{{- if ne $key "lxc-overlay" }}
lxc.mount.entry = /volumes/{{ $src.name }}/docker-{{ $key | trimSuffix "/" | replace "/" "-" }} {{ trimPrefix "/" $key }} none bind,rw,create=dir 0 0
{{- end -}}
{{- end -}}
{{- end }}
`

	RUN_JSON = `{{ "" -}}
{
	"#spec":"service-manifest-run@1",
	"config":"lxc.container.conf",
	"name":"{{- .Source.name -}}",
	"storage":{
		{{- range $key, $value := .Source.persistence -}}
		{{- if ne $key "lxc-overlay" }}
		"docker-{{ $key | trimSuffix "/" | replace "/" "-" -}}": {
			"persistence": "{{ $value }}"
		},
		{{- end -}}
		{{- end }}
		"lxc-overlay" : {
			"persistence": "{{ if index .Source.persistence "lxc-overlay" }}{{ index .Source.persistence "lxc-overlay" }}{{ else }}boot{{ end }}"
		}
	},
	"type":"lxc",
	"root-volume": "root.squashfs",
	"volumes":[]
}`
)

func BuiltinLXCDockerHandler(values map[string]interface{}) (files map[string][]byte, err error) {
	files = make(map[string][]byte, 2)
	files["lxc.container.conf"] = compileTemplate(LXC_CONTAINER_CONF, values)
	files["run.json"] = compileTemplate(RUN_JSON, values)
	return
}
