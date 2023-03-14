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
	"reflect"
	"testing"

	cjson "github.com/gibson042/canonicaljson-go"
	"github.com/stretchr/testify/assert"
)

type args struct {
	srcBuff   []byte
	patchBuff []byte
	srcFrags  string
	destFrag  string
	merge     bool
	state     *PvrMap
}

type Test struct {
	name    string
	args    args
	want    map[string]interface{}
	wantErr bool
}

func TestPatchState(t *testing.T) {
	tt := Test{
		name: "Update state without fragments",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key1\":\"value2\"}"),
			srcFrags:  "",
			destFrag:  "",
			merge:     true,
			state:     nil,
		},
		want: map[string]interface{}{
			"key1": "value2",
		},
	}

	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Update with patch fragment",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key1\":\"value2\", \"key2\":\"value2\"}"),
			srcFrags:  "",
			destFrag:  "key2",
			merge:     false,
			state:     nil,
		},
		want: map[string]interface{}{
			"key2": "value2",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Merge with patch fragment",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key1\":\"value2\", \"key2\":\"value2\"}"),
			srcFrags:  "",
			destFrag:  "key2",
			merge:     true,
			state:     nil,
		},
		want: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Update with source fragment negative",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key2\":\"value2\"}"),
			srcFrags:  "-key1",
			destFrag:  "",
			merge:     false,
			state:     nil,
		},
		want: map[string]interface{}{
			"key2": "value2",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Update with source fragment positive",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key1\":\"value2\",\"key2\":\"value2\"}"),
			srcFrags:  "key1",
			destFrag:  "",
			merge:     false,
			state:     nil,
		},
		want: map[string]interface{}{
			"key1": "value2",
			"key2": "value2",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Update with dest fragment negative",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key1\":\"value2\",\"key2\":\"value2\"}"),
			srcFrags:  "key1",
			destFrag:  "-key2",
			merge:     false,
			state:     nil,
		},
		want: map[string]interface{}{
			"key1": "value2",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Copy with new name src frag",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{}"),
			srcFrags:  "key1",
			destFrag:  "key2",
			merge:     false,
			state:     nil,
		},
		want: map[string]interface{}{
			"key2": "value1",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Copy multi with new name src frag",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\",\"key2\":\"value2\"}"),
			patchBuff: []byte("{}"),
			srcFrags:  "key1,key2",
			destFrag:  "key3,key1",
			merge:     false,
			state:     nil,
		},
		want: map[string]interface{}{
			"key1": "value2",
			"key3": "value1",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	tt = Test{
		name: "Merge states",
		args: args{
			srcBuff:   []byte("{\"key1\":\"value1\"}"),
			patchBuff: []byte("{\"key1\":\"value2\",\"key2\":\"value2\"}"),
			srcFrags:  "",
			destFrag:  "",
			merge:     true,
			state:     nil,
		},
		want: map[string]interface{}{
			"key1": "value2",
			"key2": "value2",
		},
	}
	t.Run(tt.name, func(t *testing.T) {
		_, got, err := PatchState(tt.args.srcBuff, tt.args.patchBuff, tt.args.srcFrags, tt.args.destFrag, tt.args.merge, tt.args.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PatchState() = %v, want %v", got, tt.want)
		}
	})

	t.Run("Merge new bsp to device", func(t *testing.T) {
		srcBuff := []byte("{\"#spec\":\"pantavisor-service-system@1\",\"_hostconfig/pvr/docker.json\":{\"platforms\":[\"linux/arm64\",\"linux/arm\"]},\"_sigs/bsp.json\":{\"#spec\":\"pvs@2\",\"protected\":\"eyJhbGciOiJSUzI1NiIsImp3ayI6eyJrdHkiOiJSU0EiLCJuIjoidjRJa0w1MzBYRlkyT1V4aG9vOWVRaWZ1TmxRRk9mOGxHblowUlVGSm9aTkRRNWlHRWlxeXhNMEtpOVh5UUpVV1g2RHpXY3lSVVZPUklJS29hRS1yZlB3YUh1QmdOVDRXY2RzRjN6NTRsWGQ5TUNsdWdfejl5bEgtVFNLUGFKam9ORlAyTVdDX0p5dXUxbG5ucW5iNUdzUlQ1WFh1RlA4MTNwVDlnX25fRmFWbDNuendMUnQ5ZmhHOGs3Y1M5MlJyR2Y0R1pqQnYzUEtORnJWOHBvTmZ1aDI5WHJtcTBWc3ktVGpPNUphT2Q2bzVFcXhGZXBKTnMtS21mMUdMZjlfYS1rN0I3YUQyQWdKYy1hSk9GZ2VEVjNtUFBESXFfbmtPeG9tYXA0a3ZlR0RjbkxMcDg2T1VkUEJaLWVjZHl4ZmhRT3plcm44c2o4ODQ1bmhmWlpPRkN3IiwiZSI6IkFRQUIifSwicHZzIjp7ImluY2x1ZGUiOlsiYnNwLyoqIiwiX2NvbmZpZy9ic3AvKioiXSwiZXhjbHVkZSI6WyJic3AvYnVpbGQuanNvbiIsImJzcC9zcmMuanNvbiJdfSwidHlwIjoiUFZTIiwieDVjIjpbIk1JSURhakNDQWxLZ0F3SUJBZ0lSQUtCc2hlL0F3UDlkYjZ1T1NJTFN1OEl3RFFZSktvWklodmNOQVFFTEJRQXdIREVhTUJnR0ExVUVBd3dSVUdGdWRHRjJhWE52Y2lCRVpYWWdRMEV3SGhjTk1qSXdOakV5TVRBd01ERTRXaGNOTWpRd09URTBNVEF3TURFNFdqQWFNUmd3RmdZRFZRUUREQTl3ZGkxa1pYWmxiRzl3WlhJdE1ERXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDL2dpUXZuZlJjVmpZNVRHR2lqMTVDSis0MlZBVTUveVVhZG5SRlFVbWhrME5EbUlZU0tyTEV6UXFMMWZKQWxSWmZvUE5aekpGUlU1RWdncWhvVDZ0OC9Cb2U0R0ExUGhaeDJ3WGZQbmlWZDMwd0tXNkQvUDNLVWY1TklvOW9tT2cwVS9ZeFlMOG5LNjdXV2VlcWR2a2F4RlBsZGU0VS96WGVsUDJEK2Y4VnBXWGVmUEF0RzMxK0VieVR0eEwzWkdzWi9nWm1NRy9jOG8wV3RYeW1nMSs2SGIxZXVhclJXekw1T003a2xvNTNxamtTckVWNmtrMno0cVovVVl0LzM5cjZUc0h0b1BZQ0FsejVvazRXQjROWGVZODhNaXIrZVE3R2lacW5pUzk0WU55Y3N1bnpvNVIwOEZuNTV4M0xGK0ZBN042dWZ5eVB6emptZUY5bGs0VUxBZ01CQUFHamdhZ3dnYVV3Q1FZRFZSMFRCQUl3QURBZEJnTlZIUTRFRmdRVTFlM3haaEhSQkp6MkpKSnlSMTZmMkxHMjJHTXdWd1lEVlIwakJGQXdUb0FVV1l1UStwS0M0SWJwbUNtdkUvajRJZXFSZ1QyaElLUWVNQnd4R2pBWUJnTlZCQU1NRVZCaGJuUmhkbWx6YjNJZ1JHVjJJRU5CZ2hSamNRWjBlaXFxdEJrODBlQmpJQVNUU2NXSnlqQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBekFMQmdOVkhROEVCQU1DQjRBd0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFCVzBVQ3pINW03a3E0UTRiL2p6UUszTlB5OVBGOHA5bGg4S3M1ZHZISUFRYkxtTUt5RmJYUWlyblhxZWVlSFg5K01URTZMN21KVVo2RVpvYmUxamlKNlRHM3BycTgzV3JRSEVoQlBXWTk1YWxkUEFuWEJ4SE9LejlvL3FhMytlVVlYdWhNWitQUGJZZGNtWm9lODJBNzdKVnNHMUpGY2VpRGxYVnpiOC8ranVtZ3gwVTNzWFc3YUFyMVpXYU00VmpaTHRsZndCT0ovUUNNOGx0UmFSMk96N21UamZVNTNaWDNROHNWQVRHZVVFaEdsU3pISFRCNC9vTWdGdndhaE9pZlN5Mm5STUUvQXkvQ2pMMy9IMjFFMEdEcC9pQm9BZ1BvNy9BaE1kRTlpRjJXZ3RjQXdFVUVwdmp2V0k0WWs3WWJtMzNQSXRUaGw3Q0NOY280dU5POW89Il19\",\"signature\":\"qEaqpCZajbxqpCnSlUVjPwNT7i8es1vRBFJ2XsMYQZrDrB-tsLBKOcIUz8llQ7PqQadPI9-t75s7Cq8kYfvddskOAwGwupInuBE55W6_4IdIJILKwEpbLzsVlrzjbzzIONEnxtWSWHUi53SCT9ow9eSaumnVxSBB-vvGIf9_GJY03Vcp0gPf7lp6wDJEdlnZKTU2z0aYCNd4c3qOJuDuqC30p3c1v71Yr335O4ErpoNzq7Ras0fWlQB4mM2iMClWXueR87xNmXJSsuxE7bfMsAR8TFwFBOx8UQmyzW4tX6r0MoVR2wXUIyTRHmNMTFr_91pOEGWBNd0IFCA4eDn2dg\"},\"awconnect/lxc.container.conf\":\"5a123b380dc18df181187feeacdbb7c545985ae2b3bfe17ccf01074315f3f6fa\",\"awconnect/root.squashfs\":\"089044ec5e5622e429cddfcdee262a7c4b912cd78dbc5f1e071459e3a1b9e8b9\",\"awconnect/root.squashfs.docker-digest\":\"119dc3861b6aa75a7c9a31a5d0b6b7f4ba132a945a2919d868f7989804bc789c\",\"awconnect/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"awconnect\",\"root-volume\":\"root.squashfs\",\"storage\":{\"docker--etc-NetworkManager-system-connections\":{\"persistence\":\"permanent\"},\"docker--var-crypt\":{\"disk\":\"dm-versatile\",\"persistence\":\"permanent\"},\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"awconnect/src.json\":{\"#spec\":\"service-manifest-src@1\",\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/lib/systemd/systemd\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Hostname\":\"\",\"Image\":\"sha256:cbb59b5b25379488c01d9fb924a0488db43a03df8112556f807ad0c71c78255a\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"Volumes\":{\"/etc/NetworkManager/system-connections/\":{}},\"WorkingDir\":\"/opt/wifi-connect/\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/wifi-connect@sha256:81809a6b8f99fc47c5485250c18bfe81fa91fa7339c8b3bb1a27cf5bf53e84e7\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/wifi-connect\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v5\",\"persistence\":{\"/var/crypt\":\"permanent@dm-versatile\"},\"template\":\"builtin-lxc-docker\"},\"bsp/addon-plymouth.cpio.xz4\":\"beae6a7bb235916cac52bcfece64c30615cded8c4c640e6941e7ecabe53b4920\",\"bsp/build.json\":{\"altrepogroups\":\"\",\"branch\":\"master\",\"commit\":\"bf97c28a09c0bee32c44f3c2f2acc74c22beb0a2\",\"gitdescribe\":\"015-rc5-107-gbf97c28\",\"pipeline\":\"691217908\",\"platform\":\"rpi64-5.10.y\",\"project\":\"pantacor/pv-manifest\",\"pvrversion\":\"pvr version 034-37-g84c3032c\",\"target\":\"rpi64-5.10.y\",\"time\":\"2022-11-10 14:44:34 +0000\"},\"bsp/drivers.json\":{\"#spec\":\"driver-aliases@1\",\"all\":{\"bluetooth\":[\"btbcm\",\"hci_uart\"],\"wifi\":[\"brcmutil\",\"brcmfmac\"],\"wireguard\":[\"wireguard\"]},\"dtb:all\":{}},\"bsp/firmware.squashfs\":\"056c913e7bd3471998bfa561ab3dccba160a8cdbcc2c6691f10ea78ec0694003\",\"bsp/kernel.img\":\"7a76b2ab1af99c98b8e94b7d68e07f1b32f03c9adbebd875a736cce8a46cf3e2\",\"bsp/modules.squashfs\":\"70fb342c7b07adc4e8f713fb84a13ebd0a0e77d877323b593fb886542e39f20c\",\"bsp/pantavisor\":\"6e3f635ec69a59695273373f69fb3f1812f5d15cc942048dd121e448bd1e730f\",\"bsp/run.json\":{\"addons\":[\"addon-plymouth.cpio.xz4\"],\"firmware\":\"firmware.squashfs\",\"initrd\":\"pantavisor\",\"initrd_config\":\"trail.config\",\"linux\":\"kernel.img\",\"modules\":\"modules.squashfs\"},\"bsp/src.json\":{\"#spec\":\"bsp-manifest-src@1\",\"pvr\":\"https://pvr.pantahub.com/pantahub-ci/rpi64_5_10_y_bsp_latest#bsp,_sigs,groups.json\"},\"bsp/trail.config\":\"b236948bd3caaf31c00af82fbeeb0816962e39a96ad6c5cf9bfc1a547cdbe005\",\"checkpoint.json\":{\"major\":\"2022-11-10T17:26:05+01:00\"},\"disks.json\":[{\"name\":\"dm-versatile\",\"path\":\"/storage/dm-crypt-file/disk1/versatile.img,8,versatile_key\",\"type\":\"dm-crypt-versatile\"}],\"groups.json\":[{\"description\":\"Containers which volumes we want to mount but not to be started\",\"name\":\"data\",\"restart_policy\":\"system\",\"status_goal\":\"MOUNTED\"},{\"description\":\"Container or containers that are in charge of setting network connectivity up for the board\",\"name\":\"root\",\"restart_policy\":\"system\",\"status_goal\":\"STARTED\"},{\"description\":\"Middleware and utility containers\",\"name\":\"platform\",\"restart_policy\":\"system\",\"status_goal\":\"STARTED\"},{\"description\":\"Application level containers\",\"name\":\"app\",\"restart_policy\":\"container\",\"status_goal\":\"STARTED\"}],\"lab/lxc.container.conf\":\"fc68a412438d6a6a03b7a780a6bd973d8ca0842b8eae642ae89a55190d4648f5\",\"lab/mdev.json\":{\"rules\":[\"ttyUSB[0-9]* 0:0 666\",\"rfkill 0:0 666\",\"bus/usb/.* 0:0 666\"]},\"lab/root.squashfs\":\"7f935df0e17b05968f8decf54d1eb970c553e12d3eb2bb007957bff1679af6b3\",\"lab/root.squashfs.docker-digest\":\"4bc30dab60d143ba615438f5553c4298302a4758cd53bc65f3e395b630d19f8c\",\"lab/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"lab\",\"root-volume\":\"root.squashfs\",\"runlevel\":\"app\",\"storage\":{\"docker--work\":{\"persistence\":\"permanent\"},\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"lab/src.json\":{\"#spec\":\"service-manifest-src@1\",\"args\":{\"PV_RUNLEVEL\":\"app\"},\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/bin/init\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Hostname\":\"\",\"Image\":\"sha256:f9a26bc1669815c04bb882185dbc6432ec943c60d921646eb8c6b105df07011e\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"WorkingDir\":\"\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/labutils@sha256:fc8f5c98828b885033c2767351b7de123ff94445f73166c0011285aef8e100f2\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/labutils\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v6-main\",\"persistence\":{\"/work\":\"permanent\"},\"template\":\"builtin-lxc-docker\"},\"network-mapping.json\":{},\"pv-avahi/lxc.container.conf\":\"f4662361f72fd9f2d7fdc3dcc51e66c59c2f8b16d13e14bd1f331a2889b869bb\",\"pv-avahi/root.squashfs\":\"7f64ab9af93a3c408ba2d52bd1ea35fb8df2cc0f6a35405ba5b37236ffe5bf24\",\"pv-avahi/root.squashfs.docker-digest\":\"5798c689e445f7d5132af907bdba082f0211c82be5dcceb4dfed3c5db31f2b64\",\"pv-avahi/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"pv-avahi\",\"root-volume\":\"root.squashfs\",\"storage\":{\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"pv-avahi/src.json\":{\"#spec\":\"service-manifest-src@1\",\"args\":{\"PV_VOLUME_IMPORTS\":[\"awconnect/root.squashfs:/var/lib/:/var/lib-awconnect\"]},\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/sbin/init\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Hostname\":\"\",\"Image\":\"sha256:701f4bbc1d5b707fc7daabf637cb2fa12532d05ceea6be24f8bb15a55805843b\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"WorkingDir\":\"\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/pv-avahi@sha256:895b2af2b5d407235f2b8c7c568532108a44946898694282340ef0315e2afb28\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/pv-avahi\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v6\",\"persistence\":{},\"template\":\"builtin-lxc-docker\"},\"pvr-sdk/_dm/root.squashfs.json\":{\"data_device\":\"root.squashfs\",\"hash_device\":\"root.squashfs.hash\",\"root_hash\":\"15e5be26821275e8123b57d5695754901d8c9126c32992299f0e8a72057d99bf\",\"type\":\"dm-verity\"},\"pvr-sdk/lxc.container.conf\":\"a69205914de2e8b95270f94591d5c015796590da5273db35ef6b2ed40631fcca\",\"pvr-sdk/root.squashfs\":\"00350a17f7e5432a1db53a6b90341b5da135197f7cdd605d46c84cb37f171bf8\",\"pvr-sdk/root.squashfs.docker-digest\":\"09378a77251ebf4a8f34069ad1eaa84e3be8cebe8fc19393130c2a3c51874997\",\"pvr-sdk/root.squashfs.hash\":\"280dd8b0946d281cec167bd059adc8c920e1872f4cc2113cc8b137266f0a372a\",\"pvr-sdk/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"pvr-sdk\",\"root-volume\":\"dm:root.squashfs\",\"storage\":{\"docker--etc-dropbear\":{\"persistence\":\"permanent\"},\"docker--etc-volume\":{\"persistence\":\"permanent\"},\"docker--home-pantavisor-.ssh\":{\"persistence\":\"permanent\"},\"docker--var-pvr-sdk\":{\"persistence\":\"permanent\"},\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"pvr-sdk/src.json\":{\"#spec\":\"service-manifest-src@1\",\"args\":{\"PV_LXC_EXTRA_CONF\":\"lxc.mount.entry = /volumes/_pv/addons/plymouth/text-io var/run/plymouth-io-sockets none bind,rw,optional,create=dir 0 0\",\"PV_SECURITY_WITH_STORAGE\":\"yes\"},\"dm_enabled\":{\"root.squashfs\":true},\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/sbin/init\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\",\"PVR_DISABLE_SELF_UPGRADE=true\",\"PVR_CONFIG_DIR=/var/pvr-sdk/.pvr\"],\"Hostname\":\"\",\"Image\":\"sha256:9436692279f2f91803912e058833771136eed558745c5cba63ec5222b74fc779\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"Volumes\":{\"/etc-volume\":{},\"/etc/dropbear\":{},\"/home/pantavisor/.ssh\":{},\"/var/pvr-sdk\":{}},\"WorkingDir\":\"/workspace\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/pvr-sdk@sha256:e3c1e3a5a2b2fce555429e0f5d3d193984192ef69d231919e26ce54814ea45c3\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/pvr-sdk\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v6\",\"persistence\":{\"/var/dmcrypt/volume\":\"permanent@dm-versatile\"},\"template\":\"builtin-lxc-docker\"},\"storage-mapping.json\":{}}")
		patchBuff := []byte("{\"#spec\":\"pantavisor-service-system@1\",\"_sigs/bsp.json\":{\"#spec\":\"pvs@2\",\"protected\":\"eyJhbGciOiJSUzI1NiIsImp3ayI6eyJrdHkiOiJSU0EiLCJuIjoidjRJa0w1MzBYRlkyT1V4aG9vOWVRaWZ1TmxRRk9mOGxHblowUlVGSm9aTkRRNWlHRWlxeXhNMEtpOVh5UUpVV1g2RHpXY3lSVVZPUklJS29hRS1yZlB3YUh1QmdOVDRXY2RzRjN6NTRsWGQ5TUNsdWdfejl5bEgtVFNLUGFKam9ORlAyTVdDX0p5dXUxbG5ucW5iNUdzUlQ1WFh1RlA4MTNwVDlnX25fRmFWbDNuendMUnQ5ZmhHOGs3Y1M5MlJyR2Y0R1pqQnYzUEtORnJWOHBvTmZ1aDI5WHJtcTBWc3ktVGpPNUphT2Q2bzVFcXhGZXBKTnMtS21mMUdMZjlfYS1rN0I3YUQyQWdKYy1hSk9GZ2VEVjNtUFBESXFfbmtPeG9tYXA0a3ZlR0RjbkxMcDg2T1VkUEJaLWVjZHl4ZmhRT3plcm44c2o4ODQ1bmhmWlpPRkN3IiwiZSI6IkFRQUIifSwicHZzIjp7ImluY2x1ZGUiOlsiYnNwLyoqIiwiX2NvbmZpZy9ic3AvKioiXSwiZXhjbHVkZSI6WyJic3AvYnVpbGQuanNvbiIsImJzcC9zcmMuanNvbiJdfSwidHlwIjoiUFZTIiwieDVjIjpbIk1JSURhakNDQWxLZ0F3SUJBZ0lSQUtCc2hlL0F3UDlkYjZ1T1NJTFN1OEl3RFFZSktvWklodmNOQVFFTEJRQXdIREVhTUJnR0ExVUVBd3dSVUdGdWRHRjJhWE52Y2lCRVpYWWdRMEV3SGhjTk1qSXdOakV5TVRBd01ERTRXaGNOTWpRd09URTBNVEF3TURFNFdqQWFNUmd3RmdZRFZRUUREQTl3ZGkxa1pYWmxiRzl3WlhJdE1ERXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDL2dpUXZuZlJjVmpZNVRHR2lqMTVDSis0MlZBVTUveVVhZG5SRlFVbWhrME5EbUlZU0tyTEV6UXFMMWZKQWxSWmZvUE5aekpGUlU1RWdncWhvVDZ0OC9Cb2U0R0ExUGhaeDJ3WGZQbmlWZDMwd0tXNkQvUDNLVWY1TklvOW9tT2cwVS9ZeFlMOG5LNjdXV2VlcWR2a2F4RlBsZGU0VS96WGVsUDJEK2Y4VnBXWGVmUEF0RzMxK0VieVR0eEwzWkdzWi9nWm1NRy9jOG8wV3RYeW1nMSs2SGIxZXVhclJXekw1T003a2xvNTNxamtTckVWNmtrMno0cVovVVl0LzM5cjZUc0h0b1BZQ0FsejVvazRXQjROWGVZODhNaXIrZVE3R2lacW5pUzk0WU55Y3N1bnpvNVIwOEZuNTV4M0xGK0ZBN042dWZ5eVB6emptZUY5bGs0VUxBZ01CQUFHamdhZ3dnYVV3Q1FZRFZSMFRCQUl3QURBZEJnTlZIUTRFRmdRVTFlM3haaEhSQkp6MkpKSnlSMTZmMkxHMjJHTXdWd1lEVlIwakJGQXdUb0FVV1l1UStwS0M0SWJwbUNtdkUvajRJZXFSZ1QyaElLUWVNQnd4R2pBWUJnTlZCQU1NRVZCaGJuUmhkbWx6YjNJZ1JHVjJJRU5CZ2hSamNRWjBlaXFxdEJrODBlQmpJQVNUU2NXSnlqQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBekFMQmdOVkhROEVCQU1DQjRBd0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFCVzBVQ3pINW03a3E0UTRiL2p6UUszTlB5OVBGOHA5bGg4S3M1ZHZISUFRYkxtTUt5RmJYUWlyblhxZWVlSFg5K01URTZMN21KVVo2RVpvYmUxamlKNlRHM3BycTgzV3JRSEVoQlBXWTk1YWxkUEFuWEJ4SE9LejlvL3FhMytlVVlYdWhNWitQUGJZZGNtWm9lODJBNzdKVnNHMUpGY2VpRGxYVnpiOC8ranVtZ3gwVTNzWFc3YUFyMVpXYU00VmpaTHRsZndCT0ovUUNNOGx0UmFSMk96N21UamZVNTNaWDNROHNWQVRHZVVFaEdsU3pISFRCNC9vTWdGdndhaE9pZlN5Mm5STUUvQXkvQ2pMMy9IMjFFMEdEcC9pQm9BZ1BvNy9BaE1kRTlpRjJXZ3RjQXdFVUVwdmp2V0k0WWs3WWJtMzNQSXRUaGw3Q0NOY280dU5POW89Il19\",\"signature\":\"USKn4KGF40edouHs8U2fLG5I10GD_3UkTx654A2aP7LiZoX5N2Bv-EJQyvJcOqevZ-e9cRpN1JiJkps8s2oWCK0wyh9jLe6fXP3pmnf3yCYoIi6CRisbKf3xaFpjnyCS7cT5EHM9U4PRf_cC2Vfx2XKsro40G7ZP1dalXSPz5YsdFlYBFmFIov6zNtkQ7CnjtAYPUQnA8wgWRneobe8AF-NTH2NPkbHvLbh7MyrkLKBYqpQ6UjCyECuV0TCZ88zf7Bbp37YWiEygi58ueQEzKJD4XQ020SwNxOe65CGC-vMSmmOhL-QN8-2nI2Xm9EUzmqjE1-6_9-2hfuCg5wkwJQ\"},\"bsp/addon-plymouth.cpio.xz4\":\"beae6a7bb235916cac52bcfece64c30615cded8c4c640e6941e7ecabe53b4920\",\"bsp/build.json\":{\"altrepogroups\":\"\",\"branch\":\"master\",\"commit\":\"db8e48ddfac75511c6c368c7678f1b1b2268666c\",\"gitdescribe\":\"015-rc5-105-gdb8e48d\",\"pipeline\":\"689898438\",\"platform\":\"rpi64-5.10.y\",\"project\":\"pantacor/pv-manifest\",\"pvrversion\":\"pvr version 034-37-g84c3032c\",\"target\":\"rpi64-5.10.y\",\"time\":\"2022-11-09 13:39:16 +0000\"},\"bsp/drivers.json\":{\"#spec\":\"driver-aliases@1\",\"all\":{\"bluetooth\":[\"btbcm\",\"hci_uart\"],\"wifi\":[\"brcmutil\",\"brcmfmac\"],\"wireguard\":[\"wireguard\"]},\"dtb:all\":{}},\"bsp/firmware.squashfs\":\"8de41e48b9074b740f3eb871ed2eb50ec8facfcb2ea3da40491bcf1697a797ce\",\"bsp/kernel.img\":\"9f6ca0146a4672322f18a5ab8822606c737867cd2ccc5a71a9c3e687d1c61176\",\"bsp/modules.squashfs\":\"370719b05ebc375ca6c296e02f5a279098323dcdc19df9d51ad203ff40541152\",\"bsp/pantavisor\":\"9b6d07981a3cb7adb0eedeae0be1a938859043d50ddfebeeccb6fd7bf225a719\",\"bsp/run.json\":{\"addons\":[\"addon-plymouth.cpio.xz4\"],\"firmware\":\"firmware.squashfs\",\"initrd\":\"pantavisor\",\"initrd_config\":\"trail.config\",\"linux\":\"kernel.img\",\"modules\":\"modules.squashfs\"},\"bsp/src.json\":{\"#spec\":\"bsp-manifest-src@1\",\"pvr\":\"https://pvr.pantahub.com/pantahub-ci/rpi64_5_10_y_bsp_latest#bsp,_sigs,groups.json\"},\"bsp/trail.config\":\"b236948bd3caaf31c00af82fbeeb0816962e39a96ad6c5cf9bfc1a547cdbe005\"}")
		srcFrags := ""
		destFrag := ""
		merge := true
		wantErr := false
		want := []byte("{\"#spec\":\"pantavisor-service-system@1\",\"_hostconfig/pvr/docker.json\":{\"platforms\":[\"linux/arm64\",\"linux/arm\"]},\"_sigs/bsp.json\":{\"#spec\":\"pvs@2\",\"protected\":\"eyJhbGciOiJSUzI1NiIsImp3ayI6eyJrdHkiOiJSU0EiLCJuIjoidjRJa0w1MzBYRlkyT1V4aG9vOWVRaWZ1TmxRRk9mOGxHblowUlVGSm9aTkRRNWlHRWlxeXhNMEtpOVh5UUpVV1g2RHpXY3lSVVZPUklJS29hRS1yZlB3YUh1QmdOVDRXY2RzRjN6NTRsWGQ5TUNsdWdfejl5bEgtVFNLUGFKam9ORlAyTVdDX0p5dXUxbG5ucW5iNUdzUlQ1WFh1RlA4MTNwVDlnX25fRmFWbDNuendMUnQ5ZmhHOGs3Y1M5MlJyR2Y0R1pqQnYzUEtORnJWOHBvTmZ1aDI5WHJtcTBWc3ktVGpPNUphT2Q2bzVFcXhGZXBKTnMtS21mMUdMZjlfYS1rN0I3YUQyQWdKYy1hSk9GZ2VEVjNtUFBESXFfbmtPeG9tYXA0a3ZlR0RjbkxMcDg2T1VkUEJaLWVjZHl4ZmhRT3plcm44c2o4ODQ1bmhmWlpPRkN3IiwiZSI6IkFRQUIifSwicHZzIjp7ImluY2x1ZGUiOlsiYnNwLyoqIiwiX2NvbmZpZy9ic3AvKioiXSwiZXhjbHVkZSI6WyJic3AvYnVpbGQuanNvbiIsImJzcC9zcmMuanNvbiJdfSwidHlwIjoiUFZTIiwieDVjIjpbIk1JSURhakNDQWxLZ0F3SUJBZ0lSQUtCc2hlL0F3UDlkYjZ1T1NJTFN1OEl3RFFZSktvWklodmNOQVFFTEJRQXdIREVhTUJnR0ExVUVBd3dSVUdGdWRHRjJhWE52Y2lCRVpYWWdRMEV3SGhjTk1qSXdOakV5TVRBd01ERTRXaGNOTWpRd09URTBNVEF3TURFNFdqQWFNUmd3RmdZRFZRUUREQTl3ZGkxa1pYWmxiRzl3WlhJdE1ERXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDL2dpUXZuZlJjVmpZNVRHR2lqMTVDSis0MlZBVTUveVVhZG5SRlFVbWhrME5EbUlZU0tyTEV6UXFMMWZKQWxSWmZvUE5aekpGUlU1RWdncWhvVDZ0OC9Cb2U0R0ExUGhaeDJ3WGZQbmlWZDMwd0tXNkQvUDNLVWY1TklvOW9tT2cwVS9ZeFlMOG5LNjdXV2VlcWR2a2F4RlBsZGU0VS96WGVsUDJEK2Y4VnBXWGVmUEF0RzMxK0VieVR0eEwzWkdzWi9nWm1NRy9jOG8wV3RYeW1nMSs2SGIxZXVhclJXekw1T003a2xvNTNxamtTckVWNmtrMno0cVovVVl0LzM5cjZUc0h0b1BZQ0FsejVvazRXQjROWGVZODhNaXIrZVE3R2lacW5pUzk0WU55Y3N1bnpvNVIwOEZuNTV4M0xGK0ZBN042dWZ5eVB6emptZUY5bGs0VUxBZ01CQUFHamdhZ3dnYVV3Q1FZRFZSMFRCQUl3QURBZEJnTlZIUTRFRmdRVTFlM3haaEhSQkp6MkpKSnlSMTZmMkxHMjJHTXdWd1lEVlIwakJGQXdUb0FVV1l1UStwS0M0SWJwbUNtdkUvajRJZXFSZ1QyaElLUWVNQnd4R2pBWUJnTlZCQU1NRVZCaGJuUmhkbWx6YjNJZ1JHVjJJRU5CZ2hSamNRWjBlaXFxdEJrODBlQmpJQVNUU2NXSnlqQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBekFMQmdOVkhROEVCQU1DQjRBd0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFCVzBVQ3pINW03a3E0UTRiL2p6UUszTlB5OVBGOHA5bGg4S3M1ZHZISUFRYkxtTUt5RmJYUWlyblhxZWVlSFg5K01URTZMN21KVVo2RVpvYmUxamlKNlRHM3BycTgzV3JRSEVoQlBXWTk1YWxkUEFuWEJ4SE9LejlvL3FhMytlVVlYdWhNWitQUGJZZGNtWm9lODJBNzdKVnNHMUpGY2VpRGxYVnpiOC8ranVtZ3gwVTNzWFc3YUFyMVpXYU00VmpaTHRsZndCT0ovUUNNOGx0UmFSMk96N21UamZVNTNaWDNROHNWQVRHZVVFaEdsU3pISFRCNC9vTWdGdndhaE9pZlN5Mm5STUUvQXkvQ2pMMy9IMjFFMEdEcC9pQm9BZ1BvNy9BaE1kRTlpRjJXZ3RjQXdFVUVwdmp2V0k0WWs3WWJtMzNQSXRUaGw3Q0NOY280dU5POW89Il19\",\"signature\":\"USKn4KGF40edouHs8U2fLG5I10GD_3UkTx654A2aP7LiZoX5N2Bv-EJQyvJcOqevZ-e9cRpN1JiJkps8s2oWCK0wyh9jLe6fXP3pmnf3yCYoIi6CRisbKf3xaFpjnyCS7cT5EHM9U4PRf_cC2Vfx2XKsro40G7ZP1dalXSPz5YsdFlYBFmFIov6zNtkQ7CnjtAYPUQnA8wgWRneobe8AF-NTH2NPkbHvLbh7MyrkLKBYqpQ6UjCyECuV0TCZ88zf7Bbp37YWiEygi58ueQEzKJD4XQ020SwNxOe65CGC-vMSmmOhL-QN8-2nI2Xm9EUzmqjE1-6_9-2hfuCg5wkwJQ\"},\"awconnect/lxc.container.conf\":\"5a123b380dc18df181187feeacdbb7c545985ae2b3bfe17ccf01074315f3f6fa\",\"awconnect/root.squashfs\":\"089044ec5e5622e429cddfcdee262a7c4b912cd78dbc5f1e071459e3a1b9e8b9\",\"awconnect/root.squashfs.docker-digest\":\"119dc3861b6aa75a7c9a31a5d0b6b7f4ba132a945a2919d868f7989804bc789c\",\"awconnect/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"awconnect\",\"root-volume\":\"root.squashfs\",\"storage\":{\"docker--etc-NetworkManager-system-connections\":{\"persistence\":\"permanent\"},\"docker--var-crypt\":{\"disk\":\"dm-versatile\",\"persistence\":\"permanent\"},\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"awconnect/src.json\":{\"#spec\":\"service-manifest-src@1\",\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/lib/systemd/systemd\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Hostname\":\"\",\"Image\":\"sha256:cbb59b5b25379488c01d9fb924a0488db43a03df8112556f807ad0c71c78255a\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"Volumes\":{\"/etc/NetworkManager/system-connections/\":{}},\"WorkingDir\":\"/opt/wifi-connect/\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/wifi-connect@sha256:81809a6b8f99fc47c5485250c18bfe81fa91fa7339c8b3bb1a27cf5bf53e84e7\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/wifi-connect\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v5\",\"persistence\":{\"/var/crypt\":\"permanent@dm-versatile\"},\"template\":\"builtin-lxc-docker\"},\"bsp/addon-plymouth.cpio.xz4\":\"beae6a7bb235916cac52bcfece64c30615cded8c4c640e6941e7ecabe53b4920\",\"bsp/build.json\":{\"altrepogroups\":\"\",\"branch\":\"master\",\"commit\":\"db8e48ddfac75511c6c368c7678f1b1b2268666c\",\"gitdescribe\":\"015-rc5-105-gdb8e48d\",\"pipeline\":\"689898438\",\"platform\":\"rpi64-5.10.y\",\"project\":\"pantacor/pv-manifest\",\"pvrversion\":\"pvr version 034-37-g84c3032c\",\"target\":\"rpi64-5.10.y\",\"time\":\"2022-11-09 13:39:16 +0000\"},\"bsp/drivers.json\":{\"#spec\":\"driver-aliases@1\",\"all\":{\"bluetooth\":[\"btbcm\",\"hci_uart\"],\"wifi\":[\"brcmutil\",\"brcmfmac\"],\"wireguard\":[\"wireguard\"]},\"dtb:all\":{}},\"bsp/firmware.squashfs\":\"8de41e48b9074b740f3eb871ed2eb50ec8facfcb2ea3da40491bcf1697a797ce\",\"bsp/kernel.img\":\"9f6ca0146a4672322f18a5ab8822606c737867cd2ccc5a71a9c3e687d1c61176\",\"bsp/modules.squashfs\":\"370719b05ebc375ca6c296e02f5a279098323dcdc19df9d51ad203ff40541152\",\"bsp/pantavisor\":\"9b6d07981a3cb7adb0eedeae0be1a938859043d50ddfebeeccb6fd7bf225a719\",\"bsp/run.json\":{\"addons\":[\"addon-plymouth.cpio.xz4\"],\"firmware\":\"firmware.squashfs\",\"initrd\":\"pantavisor\",\"initrd_config\":\"trail.config\",\"linux\":\"kernel.img\",\"modules\":\"modules.squashfs\"},\"bsp/src.json\":{\"#spec\":\"bsp-manifest-src@1\",\"pvr\":\"https://pvr.pantahub.com/pantahub-ci/rpi64_5_10_y_bsp_latest#bsp,_sigs,groups.json\"},\"bsp/trail.config\":\"b236948bd3caaf31c00af82fbeeb0816962e39a96ad6c5cf9bfc1a547cdbe005\",\"checkpoint.json\":{\"major\":\"2022-11-10T17:26:05+01:00\"},\"disks.json\":[{\"name\":\"dm-versatile\",\"path\":\"/storage/dm-crypt-file/disk1/versatile.img,8,versatile_key\",\"type\":\"dm-crypt-versatile\"}],\"groups.json\":[{\"description\":\"Containers which volumes we want to mount but not to be started\",\"name\":\"data\",\"restart_policy\":\"system\",\"status_goal\":\"MOUNTED\"},{\"description\":\"Container or containers that are in charge of setting network connectivity up for the board\",\"name\":\"root\",\"restart_policy\":\"system\",\"status_goal\":\"STARTED\"},{\"description\":\"Middleware and utility containers\",\"name\":\"platform\",\"restart_policy\":\"system\",\"status_goal\":\"STARTED\"},{\"description\":\"Application level containers\",\"name\":\"app\",\"restart_policy\":\"container\",\"status_goal\":\"STARTED\"}],\"lab/lxc.container.conf\":\"fc68a412438d6a6a03b7a780a6bd973d8ca0842b8eae642ae89a55190d4648f5\",\"lab/mdev.json\":{\"rules\":[\"ttyUSB[0-9]* 0:0 666\",\"rfkill 0:0 666\",\"bus/usb/.* 0:0 666\"]},\"lab/root.squashfs\":\"7f935df0e17b05968f8decf54d1eb970c553e12d3eb2bb007957bff1679af6b3\",\"lab/root.squashfs.docker-digest\":\"4bc30dab60d143ba615438f5553c4298302a4758cd53bc65f3e395b630d19f8c\",\"lab/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"lab\",\"root-volume\":\"root.squashfs\",\"runlevel\":\"app\",\"storage\":{\"docker--work\":{\"persistence\":\"permanent\"},\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"lab/src.json\":{\"#spec\":\"service-manifest-src@1\",\"args\":{\"PV_RUNLEVEL\":\"app\"},\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/bin/init\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Hostname\":\"\",\"Image\":\"sha256:f9a26bc1669815c04bb882185dbc6432ec943c60d921646eb8c6b105df07011e\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"WorkingDir\":\"\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/labutils@sha256:fc8f5c98828b885033c2767351b7de123ff94445f73166c0011285aef8e100f2\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/labutils\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v6-main\",\"persistence\":{\"/work\":\"permanent\"},\"template\":\"builtin-lxc-docker\"},\"network-mapping.json\":{},\"pv-avahi/lxc.container.conf\":\"f4662361f72fd9f2d7fdc3dcc51e66c59c2f8b16d13e14bd1f331a2889b869bb\",\"pv-avahi/root.squashfs\":\"7f64ab9af93a3c408ba2d52bd1ea35fb8df2cc0f6a35405ba5b37236ffe5bf24\",\"pv-avahi/root.squashfs.docker-digest\":\"5798c689e445f7d5132af907bdba082f0211c82be5dcceb4dfed3c5db31f2b64\",\"pv-avahi/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"pv-avahi\",\"root-volume\":\"root.squashfs\",\"storage\":{\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"pv-avahi/src.json\":{\"#spec\":\"service-manifest-src@1\",\"args\":{\"PV_VOLUME_IMPORTS\":[\"awconnect/root.squashfs:/var/lib/:/var/lib-awconnect\"]},\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/sbin/init\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Hostname\":\"\",\"Image\":\"sha256:701f4bbc1d5b707fc7daabf637cb2fa12532d05ceea6be24f8bb15a55805843b\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"WorkingDir\":\"\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/pv-avahi@sha256:895b2af2b5d407235f2b8c7c568532108a44946898694282340ef0315e2afb28\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/pv-avahi\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v6\",\"persistence\":{},\"template\":\"builtin-lxc-docker\"},\"pvr-sdk/_dm/root.squashfs.json\":{\"data_device\":\"root.squashfs\",\"hash_device\":\"root.squashfs.hash\",\"root_hash\":\"15e5be26821275e8123b57d5695754901d8c9126c32992299f0e8a72057d99bf\",\"type\":\"dm-verity\"},\"pvr-sdk/lxc.container.conf\":\"a69205914de2e8b95270f94591d5c015796590da5273db35ef6b2ed40631fcca\",\"pvr-sdk/root.squashfs\":\"00350a17f7e5432a1db53a6b90341b5da135197f7cdd605d46c84cb37f171bf8\",\"pvr-sdk/root.squashfs.docker-digest\":\"09378a77251ebf4a8f34069ad1eaa84e3be8cebe8fc19393130c2a3c51874997\",\"pvr-sdk/root.squashfs.hash\":\"280dd8b0946d281cec167bd059adc8c920e1872f4cc2113cc8b137266f0a372a\",\"pvr-sdk/run.json\":{\"#spec\":\"service-manifest-run@1\",\"config\":\"lxc.container.conf\",\"drivers\":{\"manual\":[],\"optional\":[],\"required\":[]},\"name\":\"pvr-sdk\",\"root-volume\":\"dm:root.squashfs\",\"storage\":{\"docker--etc-dropbear\":{\"persistence\":\"permanent\"},\"docker--etc-volume\":{\"persistence\":\"permanent\"},\"docker--home-pantavisor-.ssh\":{\"persistence\":\"permanent\"},\"docker--var-pvr-sdk\":{\"persistence\":\"permanent\"},\"lxc-overlay\":{\"persistence\":\"boot\"}},\"type\":\"lxc\",\"volumes\":[]},\"pvr-sdk/src.json\":{\"#spec\":\"service-manifest-src@1\",\"args\":{\"PV_LXC_EXTRA_CONF\":\"lxc.mount.entry = /volumes/_pv/addons/plymouth/text-io var/run/plymouth-io-sockets none bind,rw,optional,create=dir 0 0\",\"PV_SECURITY_WITH_STORAGE\":\"yes\"},\"dm_enabled\":{\"root.squashfs\":true},\"docker_config\":{\"AttachStderr\":false,\"AttachStdin\":false,\"AttachStdout\":false,\"Cmd\":[\"/sbin/init\"],\"Domainname\":\"\",\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\",\"PVR_DISABLE_SELF_UPGRADE=true\",\"PVR_CONFIG_DIR=/var/pvr-sdk/.pvr\"],\"Hostname\":\"\",\"Image\":\"sha256:9436692279f2f91803912e058833771136eed558745c5cba63ec5222b74fc779\",\"OpenStdin\":false,\"StdinOnce\":false,\"Tty\":false,\"User\":\"\",\"Volumes\":{\"/etc-volume\":{},\"/etc/dropbear\":{},\"/home/pantavisor/.ssh\":{},\"/var/pvr-sdk\":{}},\"WorkingDir\":\"/workspace\"},\"docker_digest\":\"registry.gitlab.com/pantacor/pv-platforms/pvr-sdk@sha256:e3c1e3a5a2b2fce555429e0f5d3d193984192ef69d231919e26ce54814ea45c3\",\"docker_name\":\"registry.gitlab.com/pantacor/pv-platforms/pvr-sdk\",\"docker_source\":\"remote,local\",\"docker_tag\":\"arm32v6\",\"persistence\":{\"/var/dmcrypt/volume\":\"permanent@dm-versatile\"},\"template\":\"builtin-lxc-docker\"},\"storage-mapping.json\":{}}")

		_, got, err := PatchState(srcBuff, patchBuff, srcFrags, destFrag, merge, nil)
		if (err != nil) != wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, wantErr)
			return
		}
		result, err := cjson.Marshal(got)
		if (err != nil) != wantErr {
			t.Errorf("PatchState() error = %v, wantErr %v", err, wantErr)
			return
		}

		assert.Equal(t, string(result), string(want))
	})
}
