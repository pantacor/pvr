package templates

const (
	LXC_CONTAINER_CONF = `{{ "" -}}
lxc.tty.max = 4
lxc.pty.max = 1024
lxc.cgroup.devices.allow = a
lxc.rootfs.path = overlayfs:/volumes/{{- .Source.name -}}/root.squashfs:/volumes/{{- .Source.name -}}/lxc-overlay/upper
lxc.init.cmd =
{{- if .Docker.Entrypoint }}
	{{- if pvrIsSlice .Docker.Entrypoint }}
		{{- if pvrSliceIndex .Docker.Entrypoint 0 }}
			{{- "" }} {{ pvrSliceIndex .Docker.Entrypoint 0}}{{ range pvrSliceFrom .Docker.Entrypoint 1 }} "{{ . }}"{{ end }}
		{{- end }}
	{{- else }}
		{{- "" }} {{ .Docker.Entrypoint }}
	{{- end }}
{{- else }}
	{{- if .Docker.Cmd }}
		{{- if pvrIsSlice .Docker.Cmd }}
			{{- if pvrSliceIndex .Docker.Cmd 0 }}
				{{- "" }} {{ pvrSliceIndex .Docker.Cmd 0}}{{ range pvrSliceFrom .Docker.Cmd 1 }} "{{ . }}"{{ end }}
			{{- end }}
		{{- else }}
			{{- "" }} {{ .Docker.Cmd }}
		{{- end }}
	{{- end }}
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
lxc.mount.entry = /dev/ dev none bind,rw,create=dir 0 0
lxc.mount.entry = /storage pvstorage none bind,rw,create=dir 0 0
lxc.mount.entry = /etc/resolv.conf etc/resolv.conf none bind,rw,create=file 0 0
lxc.mount.entry = tmpfs run tmpfs rw,nodev,relatime,mode=755 0 0
{{- with $src := .Source -}}
{{- range $key, $value := $src.persistence -}}
{{- if ne $key "lxc-overlay" }}
lxc.mount.entry = /volumes/{{ $src.name }}/docker-{{ $key | replace "/" "-" }} {{ trimPrefix "/" $key }} none bind,rw,create=dir 0 0
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
		"docker-{{ $key | replace "/" "-" -}}": {
			"persistence": "{{ $value }}"
		},
		{{- end -}}
		{{- end }}
		"lxc-overlay" : {
			"persistence": "{{ index .Source.persistence "lxc-overlay" }}"
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
