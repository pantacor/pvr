//
// Copyright 2018-2023  Pantacor Ltd.
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
package libpvr

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

type PVZeroConf struct {
	Hostname string
	AddrIPv4 []net.IP
	AddrIPv6 []net.IP
	Port     int

	Pantahub  string
	DeviceId  string
	Challenge string
}

func (p PVZeroConf) String() string {
	baseInfo := p.DeviceId
	if p.Challenge != "" {
		return baseInfo + " (unclaimed)"
	} else {
		return baseInfo + " (owned)"
	}
}

func (p PVZeroConf) ClaimCmd() string {
	if p.Challenge == "" {
		return ""
	}
	return "pvr claim -c " + p.Challenge + " " + p.Pantahub + "/devices/" + p.DeviceId
}

func Scan() {
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)

	devices := []PVZeroConf{}
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			res := PVZeroConf{}

			res.Hostname = entry.HostName
			res.AddrIPv4 = entry.AddrIPv4
			res.AddrIPv6 = entry.AddrIPv6
			res.Port = entry.Port

			for _, v := range entry.Text {
				if strings.HasPrefix(v, "pantahub=") {
					res.Pantahub = v[9:]
				} else if strings.HasPrefix(v, "challenge=") {
					res.Challenge = v[10:]
				} else if strings.HasPrefix(v, "device-id=") {
					res.DeviceId = v[10:]
				}
			}
			if res.DeviceId != "" {
				devices = append(devices, res)
				wwwURL := "https://hub.pantacor.com/u/_/devices/" + res.DeviceId
				if strings.Contains(res.Pantahub, "api2") {
					wwwURL = "https://stage.hub.pantacor.com/u/_/devices/" + res.DeviceId
				}

				fmt.Fprintf(os.Stderr, "\tID: %s\n", res)
				fmt.Fprintf(os.Stderr, "\tHost: %s\n", res.Hostname)
				fmt.Fprintf(os.Stderr, "\tIPv4: %s\n", res.AddrIPv4)
				fmt.Fprintf(os.Stderr, "\tIPv6: %s\n", res.AddrIPv6)
				fmt.Fprintf(os.Stderr, "\tPort: %d\n", res.Port)
				if res.Challenge != "" {
					fmt.Fprintf(os.Stderr, "\tClaim Cmd: %s\n", res.ClaimCmd())
				} else {
					fmt.Fprintf(os.Stderr, "\tPantahub WWW: %s\n", wwwURL)
					fmt.Fprintf(os.Stderr, "\tPVR Clone: %s\n", res.Pantahub+"/trails/"+res.DeviceId)
				}
			}
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = resolver.Browse(ctx, "_ssh._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
	fmt.Fprintln(os.Stderr, "Scanning ...")

	<-ctx.Done()

	fmt.Fprintf(os.Stderr, "Pantavisor devices detected in network: %d (see above for details)\n", len(devices))
}
