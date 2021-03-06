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
lxc.tty.max = {{ .Source.args.LXC_TTY_MAX | pvr_ifNull "8" }}
lxc.pty.max = {{ .Source.args.LXC_PTY_MAX | pvr_ifNull "1024" }}
{{- if .Source.args.PV_DEBUG_MODE }}
lxc.log.file = /pv/logs/{{ .Source.name }}.log
{{- end }}
lxc.cgroup.devices.allow = {{ .Source.args.LXC_CGROUP_DEVICES_ALLOW | pvr_ifNull "a" }}
lxc.rootfs.path = overlayfs:/volumes/{{- .Source.name -}}/root.squashfs:/volumes/{{- .Source.name -}}/lxc-overlay/upper
lxc.init.cmd =
{{- if .Docker.Entrypoint }}
	{{- if pvr_isSlice .Docker.Entrypoint }}
		{{- if pvr_sliceIndex .Docker.Entrypoint 0 }}
			{{- "" }} {{ pvr_sliceIndex .Docker.Entrypoint 0}}{{ range pvr_sliceFrom .Docker.Entrypoint 1 }} "{{ . }}"{{ end }}
		{{- end }}
		{{- if and (.Docker.Cmd) (pvr_isSlice .Docker.Cmd) }}
			{{- "" }} {{ range pvr_sliceFrom .Docker.Cmd 0 }} "{{ . }}"{{ end }}
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
{{- if and (not .Docker.Cmd) (not .Docker.Entrypoint) }} /sbin/init{{- end }}
{{- if .Docker.WorkingDir }}
	{{- if ne .Docker.WorkingDir "" }}
lxc.init.cwd = {{ .Docker.WorkingDir }}
	{{- end }}
{{- end }}
{{- if .Docker.Env }}
	{{- range .Docker.Env }}
lxc.environment = {{ . }}
	{{- end }}
{{- end }}
lxc.namespace.keep = user
{{- if .Source.args.PV_LXC_NETWORK_TYPE -}}
{{- if eq .Source.args.PV_LXC_NETWORK_TYPE "host" -}}
{{ " " }} net
{{- end -}}
{{- else -}}
{{ " " }} net
{{- end -}}
{{ " " }} ipc
{{- if .Source.args.PV_LXC_DISABLE_CONSOLE }}
lxc.console.path = none
{{- end }}
lxc.mount.auto = {{ .Source.args.LXC_MOUNT_AUTO_PROC | pvr_ifNull "proc" }}
	{{- " " }} {{ .Source.args.LXC_MOUNT_AUTO_SYS | pvr_ifNull "sys:rw" }}
	{{- " " }} {{ .Source.args.LXC_MOUNT_AUTO_GROUP | pvr_ifNull "cgroup" }}
{{- if .Source.args.PV_DISABLE_AUTODEV }}
lxc.autodev = 0
{{- end }}
{{- if .Source.args.PV_SECURITY_FULLDEV }}
lxc.mount.entry = /dev/ dev none bind,rw,create=dir 0 0
{{- end }}
{{- if .Source.args.PV_SECURITY_WITH_HOST }}
lxc.mount.entry = / host none bind,rw,create=dir 0 0
{{- end }}
{{- if .Source.args.PV_SECURITY_WITH_HOSTPROC }}
lxc.mount.entry = /proc/ host/proc none bind,rw,create=dir 0 0
{{- end }}
{{- if .Source.args.PV_SECURITY_WITH_STORAGE }}
lxc.mount.entry = /storage storage none bind,rw,create=dir 0 0
{{- end }}
{{- if (not .Source.args.PV_RESOLV_CONF_DISABLE) }}
lxc.mount.entry = /etc/resolv.conf {{ .Source.args.PV_RESOLV_CONF_PATH | pvr_ifNull "etc/resolv.conf" }} none bind,rw,create=file 0 0
{{- end }}
{{- if (not .Source.args.PV_RUN_TMPFS_DISABLE) }}
lxc.mount.entry = tmpfs {{ .Source.args.PV_RUN_TMPFS_PATH | pvr_ifNull "run" }} tmpfs rw,nodev,relatime,create=dir,mode=755 0 0
{{- end }}
{{- $src := .Source -}}
{{- range $key, $value := pvr_mergePersistentMaps .Docker.Volumes $src.persistence -}}
{{- if ne $key "lxc-overlay" }}
lxc.mount.entry = /volumes/{{ $src.name }}/docker-{{ $key | trimSuffix "/" | replace "/" "-" }} {{ trimPrefix "/" $key }} none bind,rw,create=dir 0 0
{{- end -}}
{{- end }}
{{- if .Source.args.PV_LXC_NETWORK_TYPE -}}
{{- if eq .Source.args.PV_LXC_NETWORK_TYPE "veth" }}
lxc.net.0.type = veth
lxc.net.0.link = lxcbr0
lxc.net.0.flags = up
{{- if .Source.args.PV_LXC_NETWORK_IPV4_ADDRESS }}
lxc.net.0.ipv4.address = {{ .Source.args.PV_LXC_NETWORK_IPV4_ADDRESS }}
{{- if .Source.args.PV_LXC_NETWORK_IPV4_GATEWAY }}
lxc.net.0.ipv4.gateway = {{ .Source.args.PV_LXC_NETWORK_IPV4_GATEWAY }}
{{- else }}
lxc.net.0.ipv4.gateway = auto
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- if .Source.args.PV_LXC_EXTRA_CONF }}
{{ .Source.args.PV_LXC_EXTRA_CONF }}
{{- end }}
{{- if .Source.args.PV_FILEIMPORTS }}
	{{- range $k,$v := splitList "," .Source.args.PV_FILEIMPORTS }}
		{{- $sourcePath := splitList ":" $v | sprig_first }}
		{{- $targetPath := splitList ":" $v | sprig_last }}
lxc.mount.entry = /exports/{{- $sourcePath }} {{ $targetPath }} none bind,rw,create=file 0 0
	{{- end }}
{{- end }}
{{- if .Source.args.PV_VOLUME_MOUNTS }}
{{- range $k,$v := splitList "," .Source.args.PV_VOLUME_MOUNTS }}
{{- $volume := splitList ":" $v | sprig_first }}
{{- $mountTarget := splitList ":" $v | sprig_last }}
lxc.mount.entry = /volumes/{{- $src.name -}}/{{ $volume }} {{ $mountTarget }} none bind,rw,create=dir 0 0
{{- end }}
{{- end }}
`

	RUN_JSON = `{{ "" -}}
{
	"#spec": "service-manifest-run@1",
	"config": "lxc.container.conf",
	"name":"{{- .Source.name -}}",
	{{- if .Source.args.PV_RUNLEVEL }}
	"runlevel": "{{- .Source.args.PV_RUNLEVEL }}",
	{{- end }}
	"storage":{
		{{- range $key, $value := pvr_mergePersistentMaps .Docker.Volumes .Source.persistence -}}
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
	"exports": {{  .Source.exports | sprig_toPrettyJson | sprig_indent 8 }},
	"logs": {{  .Source.logs | sprig_toPrettyJson | sprig_indent 8 }},
	"type":"lxc",
	"root-volume": "root.squashfs",
	"volumes":[
		{{- $v := sprig_list }}
		{{- if .Source.args.PV_EXTRA_VOLUMES }}
			{{- $v = sprig_splitList "," .Source.args.PV_EXTRA_VOLUMES -}}
		{{- end }}
		{{- if .Source.args.PV_VOLUME_MOUNTS }}
			{{- $m := sprig_splitList "," .Source.args.PV_VOLUME_MOUNTS -}}
			{{- range $i, $j := $m }}
				{{- $key := sprig_splitList ":" $j | sprig_first }}
				{{- $v = sprig_append $v $key }}
			{{- end }}
		{{- end }}
		{{- $n := sprig_list }}
		{{- range $i, $j := $v }}
			{{- $q := quote $j }}
			{{- $n = sprig_append $n $q }}
		{{- end }}
		{{ join ",\\n" $n }}
	]
}`
)

func BuiltinLXCDockerHandler(values map[string]interface{}) (files map[string][]byte, err error) {
	files = make(map[string][]byte, 2)
	files["lxc.container.conf"] = compileTemplate(LXC_CONTAINER_CONF, values)
	files["run.json"] = compileTemplate(RUN_JSON, values)
	return
}
