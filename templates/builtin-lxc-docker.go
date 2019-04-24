package templates

const (
	LXC_CONTAINER_CONF = `
lxc.tty = 4
lxc.pts = 1024
lxc.cgroup.devices.allow = a
lxc.utsname = major
lxc.rootfs = overlayfs:/volumes/{{- .Source.name -}}/root.squashfs:/volumes/{{- .Source.name -}}/lxc-overlay/upper
lxc.init.cmd = {{ .Docker.Entrypoint }}
lxc.mount.auto = proc sys:rw cgroup-full
lxc.mount.entry = /dev dev none bind,rw,create=dir 0 0
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
lxc.environment = force_noncontainer=1
`

	RUN_JSON = `
{
	"#spec":"service-manifest-run@1",
	"config":"lxc.container.conf",
	"name":"{{- .Source.name -}}",
	"root-volume":"root.squashfs",
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
	"root-volume: "root.squashfs",
	"volumes":[]
}`
)

func BuiltinLXCDockerHandler(values map[string]interface{}) (files map[string][]byte, err error) {
	files = make(map[string][]byte, 2)
	files["lxc.container.conf"] = compileTemplate(LXC_CONTAINER_CONF, values)
	files["run.json"] = compileTemplate(RUN_JSON, values)
	return
}
