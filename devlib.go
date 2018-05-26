//
// Copyright 2018  Pantacor Ltd.
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
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

type PVZeroConf struct {
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
			}
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = resolver.Browse(ctx, "_ssh._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
	fmt.Println("Scanning ...")

	<-ctx.Done()

	if len(devices) > 0 {
		fmt.Println("Pantavisor Devices found:")
		for i, v := range devices {
			fmt.Printf("%d.\tID: %s\n", i+1, v)
			fmt.Printf("\tClone: %s\n", v.Pantahub+"/trails/"+v.DeviceId)
			fmt.Printf("\tWWW: %s\n", "https://www.pantahub.com/u/_/devices/"+v.DeviceId)
			if v.Challenge != "" {
				fmt.Printf("\tClaim Cmd: %s\n", v.ClaimCmd())
			}
		}
	} else {
		fmt.Println("No Devices found. Please try again elsewhere ...")
	}
}
