PVR operates on two types of entities:

Sumodules:

- PVR Repository
- PVR Pantahub
- PVR Devel

# PVR Repository

## Features

- simple repository in json format with object store
- checkout and commit any working directory to repository
- look at working tree diffs with `status` and `diff` commands
- get, push and apply patches as incremental updates
- all repo operations are atomic and can be recovered
- object store can be local or in cloud/CDN

## Requirement

- a server backend implementing state CRUD primitives like offered by pantahub "trails"

## Install

1. From gitlab:

```
$ go get gitlab.com/pantacor/pvr
$ go build -o ~/bin/pvr gitlab.com/pantacor/pvr
```

## Get Started

pvr is about transforming a directory structure that contains files as well as json files into
a single managable, diffable and mergeable json file that unambiguously defines directory state.

To leverage the features of json diff/merge etc. all json files found in a directory tree will
get inlined while all other objects will get a sha reference entry with the files themselves
getting stored in a flat objects directory.

## Basics

To start a pvr from scratch you use the pvr init command which sets you up.

```
$ pvr init
```

However, more likely is that you want to consume an existing pvr made by you
or someone else and maybe change things against it:

```
$ pvr clone /path/to/pvr/repo example2
 -> mkdir example2
 -> pvr init
 -> pvr get /path/to/pvr/repo
 -> pvr checkout
```

While working on changes to your local checkout, you can use `status` and `diff`
to observe your current changes:

```
$ pvr status
A newfile.txt
D deleted.txt
C some.json
C working.txt
```

This means that `newfile.txt` is new, working.txt and some.json changed and
`deleted.txt` got removed from your working directory.

You can introspect the changes through the `diff` command:

```
$ pvr diff
{
	"deleted.txt": null,
	"newfile.txt": "dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5",
	"some.json": {
		"values": "2"
	},
	"working.txt": "9c7ab50fa91f3d78744043af5f88dce6bacd336f47733ff6a38090da3ff1a5de"
}
```

Being happy with what you see, you can checkpoint your working state using the
`commit` command:

```
$ pvr commit
Committing some.json
Committing working.txt
Adding newfile.txt
Removing deleted.txt
```

This will atomically update the json file in your repo after ensuring all the
right objects have been synched into the objects store of that pvr repo.

After committing your changes you might want to store your current repository
state for reuse or archiving purpose. You can do so using the `push` command:

```
$ pvr push /tmp/myrepo
```

You can always get a birds view on things in your repo by dumping the complete
current json:

```
$ pvr json
{...}
```

You can also push your repository to a pvr compliant REST backend. In this
case to a device trails (replace device id with your device)

```
$ pvr post https://api.pantahub.com/trails/<DEVICEID>
```

You can later clone that very repo to use it as a starting point or get
its content to update another repo.

## Internals

The pvr repository has the following structure in v1:

### The state json

```
$ cat json
{
  "spec": "pantavisor-multi-platform@1",
  "brcm.tar.gz": "8862f6feea4f6d01e28adc674285640874da19d7594dd80ed42ff7fb4dc0eea3",
  "pp/test.txt": "ad6da30bb62fae51c770574a5ca33c5e8e4bbc67fd6c5e7c094c34ad52a28e4d",
  "pp/test1.txt": "ad6da30bb62fae51c770574a5ca33c5e8e4bbc67fd6c5e7c094c34ad52a28e4d",
  "test.json": {
    "I": [
      "thank",
      "you"
    ],
    "My": "Mother",
    "more": "than"
  }
}
```

### The Objects Repository

Every PVR Repo is backed by an object repository which has for each
file a hashed file in it. These will be used on device as hardlink or on a checkout as a source to copy the files referenced from the state json file above.

By default the objects are kept centrally so you they get reused across potentially many projects you might checkout as a developer, but on device or in special cases you can use the --objects-dir parameter to use a different location.

```
$ ls objects/
8862f6feea4f6d01e28adc674285640874da19d7594dd80ed42ff7fb4dc0eea3
ad6da30bb62fae51c770574a5ca33c5e8e4bbc67fd6c5e7c094c34ad52a28e4d
d0365cf6153143a414cccaca9260bc614593feba9fe0379d0ffb7a1178470499
d9206603679fcf0a10bf4e88bf880222b05b828749ea1e2874559016ff0f5230
```

## Commands

### pvr init

```
$ pvr init
```

Observe how the repo json got created in a subdirectory:

```
$ cat .pvr/json
{
	"#spec": "pantavisor-multi-platform@1"
}
```

You would now continue editing this directory as it pleases you and you can refer to any file you put here in your configs just using the path as key (e.g. `systemc.json": {}` would create a systemc.json file on checkout).

### pvr add [file1 file2 ...]

`prv add` will put a file that exists in working directory under management of pvr. This means that the file will be honored on future `pvr diff` and `pvr commit` operations.

Example: If you bring in a basic platform to your system you simply copy them into the working dir and put them under pvr management:

```
$ cp /from/somewhere.conf lxc-platform.conf
$ cp /from/somewhere.json lxc-platform.json
$ pvr add lxc-platform.json pxc-platform.conf
```

These files will then be part of the next commit.

### pvr diff

You can look at your current changes to working directory using the diff command to get RFCXXXX json patch format:

```
$ pvr diff
{
	"lxc-platform.conf": "sha1:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	"lxc-platform.json": { ... }
}
```

### pvr commit

Committing your pvr will update the .pvr directory so it can be pushed to pantahub.

```
$ pvr commit
Adding file xxx
Changing File yyy
Commit Done
```

You can then continue editing and see your changes compared to the committed baseline using `pvr diff` again.

### pvr put <destination>

Put local:

```
$ pvr put /some/local/repopath

$ find /some/local/repopath
/some/local/repopath/json
/some/local/repopath/objects/xxxxxxxxxxxxxxxxxxxxxx
/some/local/repopath/objects/yyyyyyyyyyyyyyyyyyyyyy
```

### pvr post <remote-device-ep>

Post local pvr as a new revision to your device endpoint:

```
$ pvr post https://api.pantahub.com/trails/<YOURDEVICE>
...
```

### pvr clone <LOCATION>

you can clone a remote device state as follows:

```
$ pvr clone https://api.pantahub.com/trails/<YOURDEVICE>
...
```

Alternatively you can get a specific revision:

```
$ pvr clone https://api.pantahub.com/trails/<YOURDEVICE>/steps/<REV>
...
```

### pvr fastcopy <SOURCE> <DESTINATION>

You can copy device state or subelements from one device to another without
downloading the bits to your local machine.

This only works for source devices that the user has access to. Right now
thats the case for Public devices as well as for devices owned by the
user itself.

To fastcopy a complete device experience you would simply use:

```
pvr fastcopy -m "your commit message to remember" \
	https://pvr.pantahub.com/pantahub-ci/rpi4_initial_latest \
	https://pvr.pantahub.com/yournick/yourdevice
```

NOTE: this will delete app apps and bsp/ entries before replacing them. If you dont want that you have to use
#fragment encoded in the URL to select specific source elements to copy and replace.


To fastcopy a specific folder of your source device and replace the matching folder you coudl simply use:

```
pvr fastcopy -m "your commit message (copying #fragement)" \
	https://pvr.pantahub.com/pantahub-ci/rpi4_initial_latest#fragment \
	https://pvr.pantahub.com/yournick/yourdevice
```

Example (copy just the bsp/):

```
pvr fastcopy -m "your commit message to file for the new revision" \
	https://pvr.pantahub.com/pantahub-ci/rpi4_initial_latest#bsp \
	https://pvr.pantahub.com/yournick/yourdevice
```

Similarly you can copy apps this way:

```
pvr fastcopy -m "copy pvr-sdk from source device to yourdevice" \
	https://pvr.pantahub.com/pantahub-ci/rpi4_initial_latest#pvr-sdk \
	https://pvr.pantahub.com/yournick/yourdevice
```

# PVR Device Commands

PVR Devices commands provide convenience for developers and individuals
that operate their very own pantavisor enabled solutions.

These commands are designed for developers that want to interface with devices
in their own local network, but not does not replace a fleet management
solution.

## pvr device scan

Scan for pantavisor enabled devices on local network.

Info about how to interface and claim devices that dont have an owner yet
are printed on console.

```
example1: $ pvr device scan
Scanning ...
        ID: 5b0aa4363c6f7200095b2566 (owned)
        Host: linux.local.
        IPv4: [192.168.178.97]
        IPv6: [2a02:2028:713:3001:602:a2ff:feb3:d4e8]
        Port: 22
        Pantahub WWW: https://hub.pantacor.com/u/_/devices/5b0aa4363c6f7200095b2566
        PVR Clone: https://api.pantahub.com:443/trails/5b0aa4363c6f7200095b2566
```

# References

## Example pvr json

```
{
	"#spec": "pantavisor-multi-platform@1",
	"0base.cpio.xz": "d58791088d7e6be67b43b927f06b2deee3bf0ab0a73509852d3c1e47d0e09296",
	"alpine-mini.json": {
		"configs": [
			"lxc-alpine.config"
		],
		"exec": "/sbin/init",
		"name": "alpine-mini",
		"share": [
			"NETWORK",
			"UTS",
			"IPC"
		],
		"type": "lxc"
	},
	"alpine-mini.squashfs": "219e14651a6f2158bead0bcf37c9efa7dca2b9a96f3661d9d78e1f7d4118e7a1",
	"firmware.squashfs": "dfbfa0ffebf8fd75d0e07eb4ee8228b167b928831449f66e511182da6e3027dd",
	"kernel.img": "fec9b1db203e4ceb3b45d6bf09b6d1c971d9db0e90498b9142ee53c578269497",
	"lxc-alpine.config": "de878a7e0a3b4f23ea5b47520c8105d569f543d76c49ab0b2f6b3a5472cd5162",
	"modules.squashfs": "abe82a1b95c7314355da396ee7a25459aace231ad3057692572d90c6799d432b",
	"pantavisor.json": {
		"firmware": "/volumes/firmware.squashfs",
		"initrd": [
			"0base.cpio.gz"
		],
		"linux": "kernel.img",
		"platforms:": [
			"alpine-mini"
		],
		"volumes": [
			"alpine-mini.squashfs",
			"firmware.squashfs",
			"modules.squashfs"
		]
	}
}
```

## pvr claim -c <DEVICE_NICK> https://api.pantahub.com:443/devices/<DEVICE_ID>

pvr claim: Claims a new device

```
example1$ pvr claim -c seemingly-rich-mastodon https://api.pantahub.com:443/devices/5d0a2b4e12734a0008363a9a

Login (/type [R] to register) @ https://api.pantahub.com/auth (realm=pantahub services) ***
Username: youremail@gmail.com
Password: *****
```

## pvr device create [DEVICE_NICK]

pvr device create creates a new device from an existing device diretory usings its same state values.

```
example1$ pvr device create mydevice1
{
    "device-meta": {},
    "garbage": false,
    "id": "5d7645019061a500098617bd",
    "nick": "mydevice1",
    "owner": "prn:::accounts:/5bf2ac9e41b2dd0009a96c97",
    "prn": "prn:::devices:/5d7645019061a500098617bd",
    "public": false,
    "secret": "mznzxle72fjf6w2",
    "time-created": "2019-09-09T12:26:41.289253663Z",
    "time-modified": "2019-09-09T12:26:41.289253663Z",
    "user-meta": {}
}
Device Created Successfully
```

## pvr device get <DEVICE_NICK|ID>

pvr device get : Get Device details

```
example1$ pvr device get 5db2f6878b693e0009f1b29f
{
    "device-meta": {},
    "garbage": false,
    "id": "5db2f6878b693e0009f1b29f",
    "nick": "heavily_strong_pika",
    "owner": "prn:::accounts:/5bf2ac9e41b2dd0009a96c97",
    "owner-nick": "sirinibin",
    "prn": "prn:::devices:/5db2f6878b693e0009f1b29f",
    "public": false,
    "secret": "mznzxle72fjf6w2",
    "time-created": "2019-10-25T13:20:07.842Z",
    "time-modified": "2019-10-25T13:20:07.842Z",
    "user-meta": {}
}
example2$ pvr device get heavily_strong_pika
{
    "device-meta": {},
    "garbage": false,
    "id": "5db2f6878b693e0009f1b29f",
    "nick": "heavily_strong_pika",
    "owner": "prn:::accounts:/5bf2ac9e41b2dd0009a96c97",
    "owner-nick": "sirinibin",
    "prn": "prn:::devices:/5db2f6878b693e0009f1b29f",
    "public": false,
    "secret": "mznzxle72fjf6w2",
    "time-created": "2019-10-25T13:20:07.842Z",
    "time-modified": "2019-10-25T13:20:07.842Z",
    "user-meta": {}
}
```

## pvr device set <DEVICE_NICK|ID> <KEY1>=<VALUE1> [KEY2]=[VALUE2]...[KEY-N]=[VALUE-N]

pvr device set : Set or Update device user-meta & device-meta fields (Note:If you are logged in as USER then you can update user-meta field but if you are logged in as DEVICE then you can update device-meta field)

```
example1$ pvr device set 5df243ff0be81900099855e6 a=1 b=2
{
    "a": "1",
    "b": "2"
}
user-meta field Updated Successfully

```

```
example2$ pvr device set 5df243ff0be81900099855e6 a=1 b=2
{
    "a": "1",
    "b": "2"
}
device-meta field Updated Successfully

```

## pvr device logs [--template=<short|json|<gotemplate>] <deviceid|devicenick>[/source][@level][#platform]

pvr device logs list the logs with filter options of device,source,level & platform

```
example1$ pvr device logs 5d555d5e80123b31faa3cff2/pantavisor.log@INFO#windows
2020-01-06T12:54:17Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:43Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:53Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 3 to remember from device:5d555d5e80123b31faa3cff2

```

pvr device logs list the logs with filter options of multiple device,source,level &platform

```
example2$ pvr device logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2#windows,linux
2020-01-06T12:54:17Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:43Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 1 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:54:53Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:55:03Z 5e0f4ede:windows(pantavisor.log):INFO       My log line 2 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:55:17Z 5e0f4ede:linux(pantavisor2.log):INFO2       My log line 3 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:55:43Z 5e0f4ede:linux(pantavisor2.log):INFO2       My log line 3 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:55:53Z 5e0f4ede:linux(pantavisor2.log):INFO2       My log line 4 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:56:03Z 5e0f4ede:linux(pantavisor2.log):INFO2       My log line 4 to remember from device:5d555d5e80123b31faa3cff5
```

```
pvr device logs --from=2020-01-06 --to=2020-01-07 list the logs within a given date range

```

```
example3\$ pvr device logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2
2020-01-06T12:54:17Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:43Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:54:53Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:03Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:17Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:43Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:53Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:56:03Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff5
```

```
pvr device logs --from=2020-01-06T12:54:10--to=2020-01-06T12:54:20 list the logs within a given date time range
```

```
example4\$ pvr device logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:15Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:15Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:56:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff5

```

```

pvr device logs --from=2020-01-06T12:54:10+05:30 --to=2020-01-06T12:54:20+05:30 list the logs within a given date time range having timezone: +05:30(IST) ,Note:Timezone is optional

```

```
example5\$ pvr device logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:15Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:15Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:56:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff5
```

```
pvr device logs --from=P10D --to=P5D list the logs from last 10 days to last 5 days
Note: --from & --too flags support the ISO 8601 Duration strings
```

```
example6\$ pvr device logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:54:10Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:15Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:15Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff2
2020-01-07T12:55:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff5
2020-01-07T12:55:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:56:20Z 5e0f4ede:pantavisor2.log:INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff5
```

pvr device logs list the logs with filter options of multiple platforms

```
example7\$ pvr device logs --platform=linux,windows
5ea97febfb( ) May 1 20:28:28 linux(pantavisor.log ): My log line to remember
5ea97febfb( ) May 1 20:28:29 windows(pantavisor.log ): My log line to remember
5ea97febfb( ) May 1 20:28:30 linux(pantavisor.log ): My log line to remember

```

## pvr export <FILENAME.tar.gz>

pvr export : Exports repo into single file (tarball)

```

example1$ pvr export device.tar.gz
$ ls
alpine-hotspot bsp download-layer nginx-app pvr-sdk
app2 device.tar.gz network-mapping.json pv-avahi storage-mapping.json

```

## pvr import <FILENAME.tar.gz>

pvr import : import repo tarball (like the one produced by 'pvr export') into pvr in current working dir.It can import files with.gz or .tgz extension as well as plain .tar. Will not do pvr checkout, so working directory stays untouched.

```

example1\$ pvr import device.tar.gz

```

# PVR Pantahub Commands

Since version 006 PVR also provides convenience commands for interacting with pantahub
regardless beyond publishing pvr repositories to pantahub trails.

## pvr ps

`WARNING:`This command is DEPRECATED, please use `pvr device ps` instead

`pvr ps` gets a list of devices like below:

```

\$ pvr ps
ID NICK REV STATUS STATE SEEN MESSAGE
5a21cefc tops_urchin 20 NEW xxxx 7 months ago message....
5af32b42 verified_cicada 5 NEW xxxx 5 months ago message....
5af4ca2c classic_crappie 0 DONE xxxx 5 months ago message....
5b07f476 resolved_mule 0 DONE xxxx 4 months ago message....
5b07ff81 right_vervet 0 DONE xxxx 4 months ago message....
5b08464f helped_aphid 3 NEW xxxx about 23 hours ago message....

```

## pvr logs <deviceid|devicenick>[/source][@level][#platform]

`WARNING:`This command is DEPRECATED, please use `pvr device logs` instead

pvr logs list the logs with filter options of device,source & level

```

example1\$ pvr logs 5d555d5e80123b31faa3cff2/pantavisor.log@INFO
2020-01-06T12:54:17Z 5e0f4ede:pantavisor.log:INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:43Z 5e0f4ede:pantavisor.log:INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:53Z 5e0f4ede:pantavisor.log:INFO My log line 3 to remember from device:5d555d5e80123b31faa3cff2

```

pvr logs list the logs with filter options of multiple device,source & level

```

example2\$ pvr logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2#linux,windows
2020-01-06T12:54:17Z 5e0f4ede:windows(pantavisor.log):INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:54:43Z 5e0f4ede:windows(pantavisor.log):INFO My log line 1 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:54:53Z 5e0f4ede:windows(antavisor.log):INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:55:03Z 5e0f4ede:linux(pantavisor.log):INFO My log line 2 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:55:17Z 5e0f4ede:linux(pantavisor2.log):INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:55:43Z 5e0f4ede:linux(pantavisor2.log):INFO2 My log line 3 to remember from device:5d555d5e80123b31faa3cff5
2020-01-06T12:55:53Z 5e0f4ede:linux(pantavisor2.log):INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff2
2020-01-06T12:56:03Z 5e0f4ede:linux(pantavisor2.log):INFO2 My log line 4 to remember from device:5d555d5e80123b31faa3cff5

```

pvr logs list the logs with filter options of multiple platforms

```
example3\$ pvr logs --platform=linux,windows
5ea97febfb( ) May 1 20:28:28 linux(pantavisor.log ): My log line to remember
5ea97febfb( ) May 1 20:28:29 windows(pantavisor.log ): My log line to remember
5ea97febfb( ) May 1 20:28:30 linux(pantavisor.log ): My log line to remember

```

### pvr logs --template=<json|short|TEMPLATE>

Log output can be formatted using `--template` flag.

Valid values are:
 * `json`: json format (all fields)
 * `short`: short format (default)
 * `TEMPLATE`: golang template (see below)

Golang template syntax is documented in official golang docs: https://golang.org/pkg/text/template/

The object passed to the parse is the entire LogsEntry struct. Hence you can refer to any field
inside your template:

```
type Entry struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Device      string             `json:"dev,omitempty" bson:"dev"`
	Owner       string             `json:"own,omitempty" bson:"own"`
	TimeCreated time.Time          `json:"time-created,omitempty" bson:"time-created"`
	LogTSec     int64              `json:"tsec,omitempty" bson:"tsec"`
	LogTNano    int64              `json:"tnano,omitempty" bson:"tnano"`
	LogRev      string             `json:"rev,omitempty" bson:"rev"`
	LogPlat     string             `json:"plat,omitempty" bson:"plat"`
	LogSource   string             `json:"src,omitempty" bson:"src"`
	LogLevel    string             `json:"lvl,omitempty" bson:"lvl"`
	LogText     string             `json:"msg,omitempty" bson:"msg"`
}
```

Also we offer the following template functions:
  * sprig funcs: https://github.com/Masterminds/sprig
  * `sprint FORMAT VALUE`: single value string formatter using strformat syntax (e.g. {{ "test1234" | sprintf "%4s" }} => test)
  * `basename PATH`: basename of a given path (e.g. `{{ "/path/one" | basename }} => one` )
  * `timeformat TIME`: time format for a time field like TimeCreated (e.g. `{{ .TimeCreated | timeformat "Stamp" }} => one` )
    * Any golang time constant is supported, see https://golang.org/pkg/time/#pkg-constants (e.g. ANSIC, UnixDate, ... )


## pvr login [END_POINT]

Login to pantahub with your username & password with an optional end point
Note:Default endpoint is https://api.pantahub.com/auth/auth_status

```

example1\$ pvr login
**_ Login (/type [R] to register) @ https://api.pantahub.com/auth (realm=pantahub services) _**
Username: sirinibin2006@gmail.com
Password: **\***
Response of GET https://api.pantahub.com/auth/auth_status
{
"exp": 1568206179,
"id": "sirinibin2006@gmail.com",
"nick": "sirinibin",
"orig_iat": 1568205279,
"prn": "prn:::accounts:/5bf2ac9e41b2dd0009a96c97",
"roles": "user",
"scopes": "prn:pantahub.com:apis:/base/all",
"type": "USER"
}
LoggedIn Successfully!

```

```

example2\$ pvr login https://api2.pantahub.com/auth
**_ Login (/type [R] to register) @ https://api2.pantahub.com/auth (realm=pantahub services) _**
Username: sirinibin2006@gmail.com
Password: **\***
Response of GET https://api2.pantahub.com/auth/auth_status
{
"exp": 1568206179,
"id": "sirinibin2006@gmail.com",
"nick": "sirinibin",
"orig_iat": 1568205279,
"prn": "prn:::accounts:/5bf2ac9e41b2dd0009a96c97",
"roles": "user",
"scopes": "prn:pantahub.com:apis:/base/all",
"type": "USER"
}
LoggedIn Successfully!

```

## pvr whoami

pvr whoami : List the loggedin details with pantahub instances

```

example1\$ pvr whoami
sirinibin(prn:::accounts:/5bf2ac9e41b2dd0009a96c97) at https://api.pantahub.com/auth

```

## pvr get [remote-or-local-repository[#part1,part2,-unpart]] [target-repository]

Update target-repository from remote or local repository

```
# get latest from remembered repository
example1$ $ pvr get
33aefd8dbf46f05 [OK]
686a026cd606613 [OK]
ff2a85a62cd09f1 [OK]
76d1d085d44fd3f [OK]
b30e6b64d3e1ecb [OK]
66ae7cac57d6d05 [OK]
e8305714eaadc57 [OK]
989647b3d5b7fde [OK]
612408620229de6 [OK]
f1b441cb2721355 [OK]
d49d56059ba219c [OK]
3f4889e5eed2252 [OK]
b56390fbeb5e46f [OK]
70075ca4496451e [OK]
```

or

```
# get latest from remembered repository
example1$ $ pvr get pantahub-ci/rpi3_initial_latest
fe4d959c5541950 [OK]
5a544d45a44cf3e [OK]
834d31840a923e0 [OK]
763030a2c49b8e8 [OK]
5090f4922c2492e [OK]
018e41b74500fa7 [OK]
237277bb3586033 [OK]
7bdd13f15773ab1 [OK]
62f92986988541a [OK]
13cef79b5098b82 [OK]
dee31a19ce47772 [OK]
67b1ce399971e30 [OK]
12c8d468094e3d0 [OK]
```

You can also retrieve just one part (e.g. 'bsp' or $appname) using the fragment (#) notation:

```
pvr get pantahub-ci/rpi3_initial_latest#bsp
67b1ce399971e30 [OK]
018e41b74500fa7 [OK]
fe4d959c5541950 [OK]
dee31a19ce47772 [OK]
```

You can also retrieve two mor more parts with , separated (e.g. 'bsp' and $appname) using the fragment (#) notation:

```
pvr get pantahub-ci/rpi3_initial_latest#bsp,$appname
...
```

Further can also remove one or multiple parts from repo in the same operation by prefixing the part with a '-', e.g.
the following would get the latest bsp but also remove $appname from the pvr repo

```
pvr get pantahub-ci/rpi3_initial_latest#bsp,-$appname
...
```

You can also get from a tarball produced with ```pvr export```:

```
pvr get /tmp/myexport.tar.gz#bsp
pulling objects file /tmp/pvr-tmprepo-544439698/objects/dee31a19ce4777282e8f388ee174ec214d60d2682cd2c09537f5656b1106bf3f-> /home/asac/.pvr/objects/dee31a19ce4777282e8f388ee174ec214d60d2682cd2c09537f5656b1106bf3f.new
pulling objects file /tmp/pvr-tmprepo-544439698/objects/018e41b74500fa7109390f0505180cfb612146e7c1dcac1669b4df226110aa90-> /home/asac/.pvr/objects/018e41b74500fa7109390f0505180cfb612146e7c1dcac1669b4df226110aa90.new
pulling objects file /tmp/pvr-tmprepo-544439698/objects/fe4d959c5541950fe8fb10fd13eaeee0cfa2a9bf3fcae37c825daeb12a5366d2-> /home/asac/.pvr/objects/fe4d959c5541950fe8fb10fd13eaeee0cfa2a9bf3fcae37c825daeb12a5366d2.new
pulling objects file /tmp/pvr-tmprepo-544439698/objects/67b1ce399971e304b02aa4ad11049ae78a7c7a44652d89ef44a60a04b2b541b6-> /home/asac/.pvr/objects/67b1ce399971e304b02aa4ad11049ae78a7c7a44652d89ef44a60a04b2b541b6.new
``` 

Rememember that ```pvr get``` will update the pristine state only, but not the working copy.

You would usually introspect retrieved changes first using:

```
pvr status
```

and then replace working copy with pristine state using pvr checkout:

```
pvr checkout
```

## pvr global-config

pvr global-config :Get Global Configuration details of the repo.

```

example1\$ pvr global-config
{
"Spec": "1",
"AutoUpgrade": false,
"DistributionTag": "develop"
}

```

## pvr merge [remote-or-local-repository[#part]] [target-repository]

pvr merge : Merge content of repository into target-directory.Default target-repository is the local .pvr one. If not <repository> is provided the last one is used.

```

example1\$ pvr merge
e8305714eaadc57 [OK]
3f4889e5eed2252 [OK]
76d1d085d44fd3f [OK]
b56390fbeb5e46f [OK]
d49d56059ba219c [OK]
686a026cd606613 [OK]
70075ca4496451e [OK]
66ae7cac57d6d05 [OK]
989647b3d5b7fde [OK]
33aefd8dbf46f05 [OK]
612408620229de6 [OK]
f1b441cb2721355 [OK]
b30e6b64d3e1ecb [OK]
ff2a85a62cd09f1 [OK]
```

Same syntax for retrieving remote or local or just a part of a repository as for ```pvr get``` do apply.

## pvr putobjects <OBJECTS_ENDPOINT>

pvr putobjects : put objects from local repository to objects-endpoint

```
example1\$ pvr putobjects https://api.pantahub.com/objects
alpine-hotsp [OK]
bsp/kernel.i [OK]
pvr-sdk/root [OK]
bsp/firmware [OK]
pvr-sdk/root [OK]
alpine-hotsp [OK]
bsp/pantavis [OK]
pv-avahi/roo [OK]
bsp/pantavis [OK]
alpine-hotsp [OK]
pv-avahi/lxc [OK]
bsp/modules. [OK]
pvr-sdk/lxc. [OK]
```


## pvr self-upgrade

pvr self-upgrade : Update pvr command to the latest version

```

example1$ $ sudo pvr self-upgrade
[sudo] password for nintriva:
Starting update PVR using Docker latest tag (sha256:0d6e747e75758535bdee89657902a1499e449db9510d688e0ef16d3171203975)

Downloading layers 8 ...
Done with [3/8] from repository
Done with [2/8] from repository
Done with [7/8] from repository
Done with [8/8] from repository
Done with [5/8] from repository
Done with [6/8] from repository
Done with [1/8] from repository
Done with [4/8] from repository

Extracting layers 8 ...

Pvr installed on /bin/pvr

Docker layers are going to be cache on: /root/.pvr/cache

PVR has been updated!

```

## pvr checkout|reset

pvr checkout|reset : checkout/reset working directory to match the repo stat.reset/checkout also forgets about added files; pvr status and diff will yield empty

```

example1\$ pvr checkout

```

## pvr register [API_URL] -u \<USERNAME\> -p \<PASSWORD\> -e \<EMAIL\>

pvr register : register new user account with pantahub

```

example1$ $ pvr register https://api.pantahub.com -e jogn123@gmail.com -u john123 -p 123

Your registration process needs to be complete two steps
1.- Confirm you aren't a bot
2.- Confirm your email address

Follow this link to continue and after that come back and continue

```

# PVR App Commands

Since 022 pvr app commands are docker multi platform ready.

By default pvr will look for a priority ordered list of platforms
that the device supports running natively in `_hostconfig/docker/config.json`.

The file structure is simple:

```
cat _hostconfig/pvr/docker.json
{
    "platforms": [
        "linux/arm",
        "linux/arm/v6"
    ]
}
```

If no matching architecture is found, pvr continues and falls back to the
default that the docker_tag references.

To overwrite the defaults provide in the `_hostconfig/pvr/docker.json` file
in the pvr repo, one can pass the --plaform=linux/XXX option to pvr app add:

```
pvr app add --platform=linux/arm/v5 --from=nginx:latest nginx
```

Similar to add, developers can conveniently change platform selection
through the same `--platform` argument when using the `pvr app update`
operation.

## pvr app add <APP_NAME> --from=<DOCKER_IMAGE> --source=[remote|local],[remote|local]

pvr app add creates a new application and generates files by pulling layers from a given docker image in either remote or local docker repo's.
By default it will first look in remote repo. when not found it will pull from local docker repo,the priority can be changed using the --source flag(default:remote,local).

```

example1\$ pvr app add nginx-app --from=nginx --source=remote,local
Generating squashfs...
Downloading layers...
Layer 0 downloaded(cache)
Layer 1 downloaded(cache)
Layer 2 downloaded(cache)
Extracting layers...
Extracting layer 0
Extracting layer 1
Extracting layer 2
Stripping qemu files...
Deleted /home/nintriva/work/gitlab.com/pantacor/devices/10/nginx-app/download-layer/rootfs/usr/bin/qemu-arm-static file
Generating squashfs file
Generating squashfs digest
Application added

```

## pvr app info <APP_NAME>

pvr app info <appname> :output info and state of appname

```

example1$ $ pvr app info nginx-app
{
"#spec": "service-manifest-src@1",
"args": {},
"config": {},
"docker_digest": "sha256:231d40e811cd970168fb0c4770f2161aa30b9ba6fe8e68527504df69643aa145",
"docker_name": "nginx",
"docker_source": "remote,local",
"docker_tag": "latest",
"persistence": {},
"template": "builtin-lxc-docker"
}

```

## pvr app ls

pvr app ls :list applications in pvr checkout

```

example1$ $ pvr app ls
alpine-hotspot
app1
app2
nginx-app
pv-avahi
pvr-sdk

```

## pvr app rm <APP_NAME>

pvr app rm <appname> : remove app from pvr checkout

```

example1$ $ pvr app rm app1
\$ pvr app ls
alpine-hotspot
app2
nginx-app
pv-avahi
pvr-sdk

```

## pvr app update <APP_NAME>

pvr app update :update an existing application.

```

example1$ $ pvr app update nginx-app
Application updated

```


## pvr deploy <deploy-dir> [source-repos]+


With pvr deploy you can deploy one to many source repos to a deployment directory such as the
one you find on the pantavisor device for each revision.

This command can be used to modify a rootfs and change/replace/update apps.

Example, will deploy the 'os' container from a local repository, the 'bsp' container from a pvr
export to a factory revision (trails/0) in a pantavisor enabled rootfs.

```
pvr deploy trails/0  /path/to/repo/.pvr#os /tmp/export.tgz#bsp
```

This command will create hardlinks of the objects to the objects pool so do not
use this on a host where you intend the checkout to be edited.


# PVR sig commands

PVR sig commands offers support for maintaining pvs signatures inside your tree.

For details see README.pvs.md

Commands currently supported are:

 * pvr sig add - adds a new signature to the _pvs/ hierarchy of the state
 * pvr sig update - updates a committed signature from the _pvs/ hierarchy to be validate against committed state
 * pvr sig ls - list files covered by signatures in _sigs/ hieararchy; by default sig ls will show signature info while considering _all_ signatures in the system state


## PVR sig with CA commands

```
commit c2d6f1450422b89d90948499c7cd6dd6949e5df3 (HEAD -> feature/pvs-ca, origin/feature/pvs-ca)
Author: Alexander Sack <asac@pantacor.com>
Date:   Wed Oct 20 17:58:50 2021 +0200

    add support for using x509 cert chains using x5c jws header to determine trust in pvr signatures
    
    * introduce new --x5c argument pvr app sig command to provide the chain to include in pvr sig add and update commands
    * introduce --cacerts argument to pvr sig commands to allow to post a trust CACERTS file to use to validate in pvr app ls;
      using special value _system will use the system cert store to validate ca chain
    * pubkey validation now allows to have multiple trusted pubkeys in the file referenced by --pubkey
    * document this feature in README.md
    
    Example 1: "add signature with trust ca chain"
    
    Below statement injects the myKey.crt as the trust chain into the jws.
    If you have intermediates your .crt file would need to include those
    also in reverse order.
    
    ```
    pvr sig --x5c ../ca/myKey.crt --key ../ca/myKey.key add --part nginx
    ```
    
    Example 2: "update signatures with trustchain"
    
    Below will refresh the nginx.json signature and attach myKey.crt as
    the trust ca cert chain to validate against root certificates
    
    ```
    pvr sig --x5c ../ca/myKey.crt --key ../ca/myKey.key update _sigs/nginx.json
    ```
    
    Example 3: "validate signatures with cert pool in file"
    
    Below you can see how to validate signature with ca cert pool in file myCA.pem.
    
    ```
    pvr sig --cacerts ../ca/myCA.pem ls --part _sigs/nginx.json
    ```
    
    Example 4: use system ca cert pool to validate signature
    
    For this you have to put your myCA.pem into one of the system folders for
    trusted certificates. e.g. /etc/ssl/certs
    
    ```
    pvr sig ls --part _sigs/nginx.json
    ```
```

