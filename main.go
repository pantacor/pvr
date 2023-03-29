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

package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/go-resty/resty"
	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func main() {

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "pvr"
	app.Usage = "PantaVisor Repo"
	app.Version = VERSION
	home := os.Getenv("HOME")

	if home == "" {
		usr, err := user.Current()
		if err != nil {
			euid := syscall.Geteuid()
			usr, err = user.LookupId(fmt.Sprintf("%d", euid))
		}
		if err != nil {
			panic(err)
		}
		home = usr.HomeDir
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "access-token, a",
			Usage:  "Use `ACCESS_TOKEN` for authorization with core services",
			EnvVar: "PVR_ACCESSTOKEN",
		},
		cli.StringFlag{
			Name:   "baseurl, b",
			Usage:  "Use `BASEURL` for resolving prn URIs to core service endpoints",
			EnvVar: "PVR_BASEURL",
			Value:  "https://api.pantahub.com",
		},
		cli.StringFlag{
			Name:   "http-proxy",
			Usage:  "Use `PVR_HTTP_PROXY` to configure the proxy mode. Values 'no' (disable), '<URL>' (use proxy by url), 'system' (use system settings - default)",
			EnvVar: "PVR_HTTP_PROXY",
			Value:  "system",
		},
		cli.StringFlag{
			Name:   "repo-baseurl, r",
			Usage:  "Use `PVR_REPO_BASEURL` for resolving PVR repositories like docker through user/name syntax.",
			EnvVar: "PVR_REPO_BASEURL",
			Value:  "https://pvr.pantahub.com",
		},
		cli.StringFlag{
			Name:   "config-dir, c",
			Usage:  "Use `PVR_CONFIG_DIR` for using a custom global config directory (used to store auth.json etc.).",
			EnvVar: "PVR_CONFIG_DIR",
			Value:  filepath.Join(home, ".pvr"),
		},
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "enable debugging output for rest calls",
			EnvVar: "PVR_DEBUG",
		},
		cli.BoolFlag{
			Name:   "disable-self-upgrade",
			Usage:  "disable self-upgrades",
			EnvVar: "PVR_DISABLE_SELF_UPGRADE",
		},
		cli.BoolFlag{
			Name:   "insecure, i",
			Usage:  "skip tls verify",
			EnvVar: "PVR_INSECURE",
		},
	}

	app.Before = func(c *cli.Context) error {
		libpvr.IsDebugEnabled = c.GlobalBool("debug")
		resty.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: c.GlobalBool("insecure")})
		resty.SetDebug(libpvr.IsDebugEnabled)

		c.App.Metadata["PVR_AUTH"] = c.GlobalString("access-token")

		if c.GlobalString("baseurl") != "" {
			c.App.Metadata["PVR_BASEURL"] = c.GlobalString("baseurl")
		} else {
			c.App.Metadata["PVR_BASEURL"] = "https://api.pantahub.com"
		}

		baseURL, err := url.Parse(c.App.Metadata["PVR_BASEURL"].(string))
		if err != nil {
			return err
		}
		c.App.Metadata["PVR_BASEURL_url"] = baseURL

		if c.GlobalString("repo-baseurl") != "" {
			c.App.Metadata["PVR_REPO_BASEURL"] = c.GlobalString("repo-baseurl")
		} else {
			c.App.Metadata["PVR_REPO_BASEURL"] = "https://pvr.pantahub.com"
		}

		repoBaseURL, err := url.Parse(c.App.Metadata["PVR_REPO_BASEURL"].(string))
		if err != nil {
			return err
		}
		c.App.Metadata["PVR_REPO_BASEURL_url"] = repoBaseURL

		if c.GlobalString("config-dir") != "" {
			c.App.Metadata["PVR_CONFIG_DIR"] = c.GlobalString("config-dir")
		} else {
			c.App.Metadata["PVR_CONFIG_DIR"] = filepath.Join(home, ".pvr")
		}

		if !c.GlobalBool("disable-self-upgrade") {
			c.App.Metadata["PVR_SELF_UPGRADE"] = "yes"
		} else {
			c.App.Metadata["PVR_SELF_UPGRADE"] = nil
		}

		transport := http.DefaultTransport.(*http.Transport)

		// --http-proxy=no -> disable proxy even if http_proxy env is set
		if c.GlobalString("http-proxy") == "system" {
			transport.Proxy = http.ProxyFromEnvironment
		} else if c.GlobalString("http-proxy") == "no" {
			transport.Proxy = nil
		} else {
			u, err := url.Parse(c.GlobalString("http-proxy"))
			if err != nil {
				return errors.New("ERROR: http-proxy not a valid url")
			}
			transport.Proxy = http.ProxyURL(u)
		}

		resty.SetTransport(transport)

		libpvr.UpdateIfNecessary(c)

		return nil
	}

	app.Commands = []cli.Command{
		CommandInit(),
		CommandAdd(),
		CommandJson(),
		CommandClaim(),
		CommandDiff(),
		CommandStatus(),
		CommandCommit(),
		CommandSig(),
		CommandPut(),
		CommandPost(),
		CommandGet(),
		CommandInspect(),
		CommandRemoteInfo(),
		CommandStepInfo(),
		CommandDeploy(),
		CommandMerge(),
		CommandReset(),
		CommandClone(),
		CommandFastCopy(),
		CommandPutObjects(),
		CommandExport(),
		CommandImport(),
		CommandRegister(),
		CommandScanDeprecated(),
		CommandPsDeprecated(),
		CommandLogsDeprecated(),
		CommandApp(),
		CommandSelfUpdate(),
		CommandGlobalConfig(),
		CommandWhoami(),
		CommandLogin(),
		CommandDevice(),
		CommandCompletion(),
		CommandDmApply(),
		CommandDmConvert(),
		CommandVersion(),
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}
