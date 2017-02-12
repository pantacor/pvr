# The PiVo Repo

> "*When you are in scalable REST world, there is not much else than JSON with binary blobs being published through dumb CDNs.
> 
> With PVR, consuming and sharing file trees in json format becomes a joy for REST environments.*"

# Features

 * repository format with single json file and object store
 * create file tree from repo
 * commit file tree to repo
 * json file is a path to object map; json content inlined
 * json diffs describe all changes to file tree
 * json diffs can be patched in as incremental updates
 * all repo operations are atomic and can be recovered from
 * object store can be local or in cloud/CDN

# Install

1. Get latest pvr from your distribution, e.g.

    ```
    $ apt-get install pvr
    ```

2. Download latest binary matching your architecture:

    ```
    $ wget http://downloads.pantahub.com/x86-64/pvr
    $ chmod a+x pvr
    ```

# Get Started

pvr is about transforming a directory structure that contains files as well as json files into
a single managable, diffable and mergeable json file that unambiguously defines directory state.

To leverage the features of json diff/merge etc. all json files found in a directory tree will
get inlined while all other objects will get a sha reference entry with the files themselves
getting stored in a flat objects directory.

# pvr basics

You can checkout a pvr directory to your working directory

```
# by default it will expect a pvr repo at .pvr/ in cwd
pvr checkout

# alternatively you can also just refer to a directory elsewhere
pvr checkout $HOME/my/pvrrepo/

```

# Internals

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

# Commands

## pvr init <SPEC>
```
$ pvr init pantavisor-multi-platform@1
Created empty pantavisor-multi-platform@1 project.

$ cat .pvr/system.json
{
	"#spec": "pantavisor-multi-platform@1",
	"systemc.json" {
		"linux": "",
		"initrd": [
			"",
		],
		"platforms:": [],
		"volumes": {},
	}
}
```

As you can see the init program has created a template for you
that needs filling up.

```
$ ls -a
.
..
.pvr
systemc.json
```

You would now continue editing this directory as it pleases you. and you can refer to any file you put here in your configs just using the absolute path (e.g. /systemc.json).

## pvr add [file1 file2 ...]

`prv add` will put a file that exists in working directory under management of pvr. This means that the file will be honored on future `pvr diff` and `pvr commit` operations.

Example: If you bring in a basic platform to your system you simply copy them into the working dir and put them under pvr management:

```
$ cp /from/somewhere.conf lxc-platform.conf
$ cp /from/somewhere.json lxc-platform.json
$ pvr add lxc-platform.json pxc-platform.conf
```

These files will then be part of the next commit.

## pvr diff
You can look at your current changes to working directory using the diff command to get RFCXXXX json patch format:

```
$ pvr diff
{
	"lxc-platform.conf": "sha1:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	"lxc-platform.conf": "sha1:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
}
```

## pvr commit
Committing your pvr will update the .pvr directory so it can be pushed to pantahub.

```
$ pvr commit -m "my commit message"
Adding file xxx
Changing File yyy
Commit Done
```

You can then continue editing and see your changes compared to the committed baseline using `pvr diff` again.

## pvr push <destination>

Push local:
```
$ pvr push /some/local/repopath
[=============================================] 100%
$ find /some/local/repopath
/some/local/repopath/json
/some/local/repopath/rev
/some/local/repopath/commitmsg
/some/local/repopath/objects/sha1:xxxxxxxxxxxxxxxxxxxxxx
/some/local/repopath/objects/sha1:yyyyyyyyyyyyyyyyyyyyyy
```

Push to your device:
```
$ pvr push https://localhost:12366/pvr/api/v1/DEVICEID/system
DONE [=============================================] 100%
```

Talk about your device on panta blog:

```
$ pvr post -h https://plog.pantahub.com/ricmm/
Title: My first raspberry pi system!
Tags: rpi3 yocto minimal system
Summary: Install this following the instructions and
  post your comments through disqus.
Instructions:
# MD Format Instructions
here.
<EMPTYLINE>
<EMPTYLINE>
Posted: https://plog.pantahub.com/ricmm/my-first-raspberry-pi-system
$
```

## pvr get <LOCATION> (Blog)

A reader wants to use your first raspberry pi system and apply it to one of his rpi3s

```
$ pvr get https://plog.pantahub.com/ricmm/my-first-raspberry-pi-system
DONE [=============================================] 100%
$ pvr push https://localhost:12366/pantavisor/api/v1/DEVICEID/system
DONE [=============================================] 100%
```

## pvr get <LOCATION> (Device)

A developer or admin or app wants to get state from a device instead of a post. He can also do that using `pvr get` primitive:
```
$ pvr get https://localhost:12366/pantavisor/api/v1/DEVICEID/system
```

```
$ pvr get https://localhost:12366/pantavisor/api/v1/DEVICEID1/system#latest
DONE [=============================================] 100%
$ pvr push https://localhost:12366/pantavisor/api/v1/DEVICEID2/system
DONE [=============================================] 100%
```


# References

## Example system pvr json

```
"system.json":
{
	"#spec": "pantavisor-multi-platform@1",
	"myvideo.blob": "sha:xxxxxxxxxxxxxxxxxxxxxxxxxxx",
	"config.json" {
	  chipalo: "run quick and better"
	},
	"conf": {
		"lxc-owrt-mips.conf": "sha:tttttttttttttttttttttt",
		"lxc-ble-gw1.conf": "sha:rrrrrrrrrrrrrrrrrrrr"
	},
	"systemc.json" {
		"spec": "pantavisor-systemc@1",
		"linux": "/kernel.img",
		"initrd": [
			"/0base.cpio.gz",
			"/asacrd.cpio.gz",
		],
		"platforms:": ["lxc-owrt-mips.json"]
		"volumes": []
	},
	"lxc-owrt-mips.json":
	{
		"spec": "pantavisor-lxc-runner@1",
		"parent": null,
		"lxc-config": "/owrt.json",
		"lxc-shares": [ NETWORK, UTS, IPC ],
		"lxc-exec": "/init"
	},
	"lxc-azure-ble-gw1.json":
	{
		"parent": lxc-owrt-mips,
		"runner": "lxc",
		"lxc-config": "/files/lxc-ble-gw1.conf",
		"lxc-shares": [],
		"lxc-exec": "/init"
	},
}


/
/storage
/storage/trails/
/storage/trails/0.json
/storage/trails/0/chipalo.json
/storage/trails/0/systemc.json
/storage/trails/0/lxc-azure-ble-gw1.json
/storage/trails/0/kernel.img
/storage/trails/0/conf
/storage/trails/0/conf/lxc-ble-gw1.conf
/storage/trails/0/conf/lxc-owrt-mips.conf
/storage/objects/...
```

## Example trailsd config v2

```
spec: "com.pantacor.trails.state@0.2"

"state":
{
    "rev": 0,
    "kernel": "kernel.img",
    "initrd": [
      "0base.cpio.gz",
      "asacrd.cpio.gz",
    ],
    "files": [
    {
      "key": "0base.cpio.gz",
      "value": {
        "file": "5895c285a5717a4d2000001b"
      }
    },
    {
      "key": "kernel.img",
      "value": {
        "file": "2670a7bca5717a1c2900000a"
      }
    },
    {
      "key": "kernel-dbg.img",
      "value": {
        "file": "2670a7bca5717a1c2900000a"
      }
    },
    {
      "key": "linaro-minimal-lxc.conf",
      "value": {
        "file": "2670a7bca5717a1c2900000a"
      }
    },
    {
      "key": "azuredemo-blegateway-lxc.conf",
      "value": {
        "file": "2670a7bca5717a1c2900000a"
      }
    },
  ],
  "volumes": [
	{
      "key": "blegateway-config.squashfs",
      "value": {
        "type": "ro",
        "file": "5895c285a5717a4d2000001b"
      }
    },
    {
      "key": "blegateway-armv7.squashfs",
      "value": {
        "type": "ro",
        "file": "5895dbd8a5717a02b0000001"
      }
    },
    {
      "key": "linaro-armv7.squashfs",
      "value": {
        "type": "ro",
        "file": "58935de5a5717a798a000018"
      }
    },
    {
      "key": "writable-ble.ext4",
      "value": {
        "type": "rw",
        "file": "5873c7bca5717a1c2900000e"
      }
    },
    {
      "key": "writable-linaro.ext4",
      "value": {
        "type": "rw",
        "file": "5873c7bca5717a1c2900000e"
      }
    },
    {
      "key": "linaro-minimal-lxc.conf",
      "value": {
        "type": "ro",
        "file": "5923a7bca5717a1c2900000e"
      }
    },
    {
      "key": "azuredemo-blegateway-lxc.conf",
      "value": {
        "type": "ro",
        "file": "5873c7bca5717a1c2900000e"
      }
    },
  ],
  "platforms": {
    "linaro-minimal": {
      "type": "lxc",
      "parent": null,
      "config": "linaro-minimal-lxc.conf",
      "exec": "/sbin/init",
      "share": [
        "NETWORK",
        "UTS",
        "IPC",
      ]
    },
    "azuredemo-blegateway": {
      "type": "lxc",
      "parent": linaro-minimal,
      "config": "azuredemo-blegateway-lxc.conf",
      "exec": "/sbin/init",
      "share": []
    }
  },
}
```