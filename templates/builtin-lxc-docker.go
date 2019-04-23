package templates

const (
	LXC_CONTAINER_CONF = `
lxc.tty = 4
lxc.pts = 1024
lxc.cgroup.devices.allow = a
lxc.utsname = major
lxc.rootfs = overlayfs:/volumes/limescan-mini.squashfs:/storage/limescan-gini.disk.11/upper
lxc.init_cmd = /bin/systemd
lxc.mount.auto = proc sys:rw cgroup-full
lxc.mount.entry = /dev dev none bind,rw,create=dir 0 0
lxc.mount.entry = /storage pvstorage none bind,rw,create=dir 0 0
lxc.mount.entry = /etc/resolv.conf etc/resolv.conf none bind,rw,create=file 0 0
lxc.mount.entry=tmpfs run tmpfs rw,nodev,relatime,mode=755 0 0
lxc.environment = force_noncontainer=1`

	RUN_JSON = `
{
	"#spec":"service-manifest-run@1",
	"config":"lxc.container.conf",
	"name":"limescan-device",
	"root-volume":"root.squashfs",
	"storage":{
		"lxc-overlay":{
			"persistence":"revision"
		}
	},
	"type":"lxc",
	"volumes":[
		"something-else.squashfs"
	]
}`
)

func BuiltinLXCDockerHandler(values map[string]interface{}) (files map[string][]byte, err error) {
	files = make(map[string][]byte, 2)
	files["lxc.container.conf"] = compileTemplate(LXC_CONTAINER_CONF, values)
	files["run.json"] = compileTemplate(RUN_JSON, values)
	return
}
