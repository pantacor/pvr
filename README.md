PVR operates on two types of entities:

Sumodules:
 * PVR Repository
 * PVR Pantahub
 * PVR Devel

# PVR Repository

## Features

 * simple repository in json format with object store
 * checkout and commit any working directory to repository
 * look at working tree diffs with `status` and `diff` commands
 * get, push and apply patches as incremental updates
 * all repo operations are atomic and can be recovered
 * object store can be local or in cloud/CDN

## Requirement

 * a server backend implementing state CRUD primitives like offered by pantahub "trails"

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
example1$ pvr init
```

However, more likely is that you want to consume an existing pvr made by you
or someone else and maybe change things against it:

```
pvr clone /path/to/pvr/repo example2
 -> mkdir example2
 -> pvr init
 -> pvr get /path/to/pvr/repo
 -> pvr checkout
```

While working on changes to your local checkout, you can use `status` and `diff`
to observe your current changes:

```
example2$ pvr status
A newfile.txt
D deleted.txt
C some.json
C working.txt
```

This means that `newfile.txt` is new, working.txt and some.json changed and
`deleted.txt` got removed from your working directory.

You can introspect the changes through the `diff` command:

```
example2$ pvr diff
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
example2$ pvr commit
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
example2$ pvr push /tmp/myrepo
```

You can always get a birds view on things in your repo by dumping the complete
current json:
```
example2$ pvr json
{...}
```

You can also push your repository to a pvr compliant REST backend. In this
case to a device trails (replace device id with your device)

```
example2: pvr post https://api.pantahub.com/trails/<DEVICEID>
```

You can later clone that very repo to use it as a starting point or get 
its content to update another repo.

## Internals

The pvr repository has the following structure in v1:

objects/:
```
ls objects/
8862f6feea4f6d01e28adc674285640874da19d7594dd80ed42ff7fb4dc0eea3
ad6da30bb62fae51c770574a5ca33c5e8e4bbc67fd6c5e7c094c34ad52a28e4d
d0365cf6153143a414cccaca9260bc614593feba9fe0379d0ffb7a1178470499
d9206603679fcf0a10bf4e88bf880222b05b828749ea1e2874559016ff0f5230
```

json:
```
cat json
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

## Commands

### pvr init
```
$ pvr init
$ cat .pvr/json 

{
	"#spec": "pantavisor-multi-platform@1"
}
```

You would now continue editing this directory as it pleases you. and you can refer to any file you put here in your configs just using the absolute path (e.g. /systemc.json).

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

# PVR Device Commands

PVR Devices commands provide convenience for developers and individuals
that operate their very own pantavisor enabled solutions.

These commands are designed for developers that want to interface with devices
in their own local network, but not does not replace a fleet management
solution.

## pvr scan

Scan for pantavisor enabled devices on local network.

Info about how to interface and claim devices that dont have an owner yet
are printed on console.

```
> .\pvr.exe scan
Scanning ...
        ID: 5b0aa4363c6f7200095b2566 (owned)
        Host: linux.local.
        IPv4: [192.168.178.97]
        IPv6: [2a02:2028:713:3001:602:a2ff:feb3:d4e8]
        Port: 22
        Pantahub WWW: https://www.pantahub.com/u/_/devices/5b0aa4363c6f7200095b2566
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

# PVR Pantahub Commands

Since version 006 PVR also provides convenience commands for interacting with pantahub
regardless beyond publishing pvr repositories to pantahub trails.

## pvr ps

```pvr ps``` gets a list of devices like below:

```
$ pvr ps
     ID               NICK             REV   STATUS   STATE          SEEN            MESSAGE
  5a21cefc   tops_urchin                20   NEW      xxxx    7 months ago         message....
  5af32b42   verified_cicada             5   NEW      xxxx    5 months ago         message....
  5af4ca2c   classic_crappie             0   DONE     xxxx    5 months ago         message....
  5b07f476   resolved_mule               0   DONE     xxxx    4 months ago         message....
  5b07ff81   right_vervet                0   DONE     xxxx    4 months ago         message....
  5b08464f   helped_aphid                3   NEW      xxxx    about 23 hours ago   message....
```

## pvr logs <deviceid|devicenick>[/source][@Level]

```pvr logs list the logs with filter options of device,source & level

```
example1$ pvr logs 5d555d5e80123b31faa3cff2/pantavisor.log@INFO
	{
    "start": 0,
    "page": 1,
    "count": 1,
    "entries":[
        {
            "id": "5d12142980123bb4654d11fc",
            "dev": "prn:::devices:/5d555d5e80123b31faa3cff2",
            "own": "prn:pantahub.com:auth:/user1",
            "time-created": "2019-06-25T12:31:37.178Z",
            "tsec": 1496532292,
            "tnano": 802110514,
            "src": "pantavisor.log",
            "lvl": "INFO",
            "msg": "My log line to remember"
        }
	]		
}
```
```pvr logs list the logs with filter options of multiple device,source & level

```
example2$ pvr logs 5d555d5e80123b31faa3cff2,5d555d5e80123b31faa3cff5/pantavisor.log,pantavisor2.log@INFO,INFO2
{
    "start": 0,
    "page": 1,
    "count": 2,
    "entries":[
        {
            "id": "5d12142980123bb4654d11fc",
            "dev": "prn:::devices:/5d555d5e80123b31faa3cff2",
            "own": "prn:pantahub.com:auth:/user1",
            "time-created": "2019-06-25T12:31:37.178Z",
            "tsec": 1496532292,
            "tnano": 802110514,
            "src": "pantavisor.log",
            "lvl": "INFO",
            "msg": "My log line to remember"
        },
		{
            "id": "5d12142980123bb4654d11fc",
            "dev": "prn:::devices:/5d555d5e80123b31faa3cff5",
            "own": "prn:pantahub.com:auth:/user1",
            "time-created": "2019-06-25T12:31:37.178Z",
            "tsec": 1496532292,
            "tnano": 802110514,
            "src": "pantavisor2.log",
            "lvl": "INFO2",
            "msg": "My log line to remember"
        }
	]	
} 	
```

## pvr login [END_POINT]

```
Login to pantahub with your username & password with an optional end point
Note:Default endpoint is https://api.pantahub.com/auth/auth_status


```
example1$  pvr login
*** Login (/type [R] to register) @ https://api.pantahub.com/auth (realm=pantahub services) ***
Username: sirinibin2006@gmail.com
Password: *****
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
example2$  pvr login https://api2.pantahub.com/auth
*** Login (/type [R] to register) @ https://api2.pantahub.com/auth (realm=pantahub services) ***
Username: sirinibin2006@gmail.com
Password: *****
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