// Copyright (c) 2023-2025, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikernels

import (
	"encoding/json"
	"fmt"
	"strings"
)

const RumprunUnikernel string = "rumprun"
const SubnetMask125 = "128.0.0.0"

type Rumprun struct {
	Command string     `json:"cmdline"`
	Net     RumprunNet `json:"net"`
	Blk     RumprunBlk `json:"blk"`
}

type RumprunNoNet struct {
	Command string     `json:"cmdline"`
	Blk     RumprunBlk `json:"blk"`
}

type RumprunCmd struct {
	Cmdline string `json:"cmdline"`
}

type RumprunNet struct {
	Interface string `json:"if"`
	Cloner    string `json:"cloner"`
	Type      string `json:"type"`
	Method    string `json:"method"`
	Address   string `json:"addr"`
	Mask      string `json:"mask"`
	Gateway   string `json:"gw"`
}

type RumprunBlk struct {
	Source     string `json:"source"`
	Path       string `json:"path"`
	FsType     string `json:"fstype"`
	Mountpoint string `json:"mountpoint"`
}

func (r *Rumprun) CommandString() (string, error) {
	// if EthDeviceMask is empty, there is no network support. omit every relevant field
	if r.Net.Mask == "" {
		tmp := RumprunNoNet{
			Command: r.Command,
			Blk:     r.Blk,
		}
		jsonData, err := json.Marshal(tmp)
		if err != nil {
			return "", err
		}
		jsonStr := string(jsonData)
		return jsonStr, nil
	}
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	jsonStr := string(jsonData)
	return jsonStr, nil
}

func (r *Rumprun) SupportsBlock() bool {
	return true
}

func (r *Rumprun) SupportsFS(fsType string) bool {
	switch fsType {
	case "ext2":
		return true
	default:
		return false
	}
}

func (r *Rumprun) MonitorNetCli(monitor string) string {
	switch monitor {
	case "hvt", "spt":
		return "--net:tap="
	default:
		return ""
	}
}

func (r *Rumprun) MonitorBlockCli(monitor string) string {
	switch monitor {
	case "hvt", "spt":
		return "--block:rootfs="
	default:
		return ""
	}
}

// Rumprun can execute only on top of Solo5 and currently there
// are no generic Solo5-specific arguments that Rumprun requires
func (r *Rumprun) MonitorCli(_ string) string {
	return ""
}

func (r *Rumprun) Init(data UnikernelParams) error {
	// if EthDeviceMask is empty, there is no network support
	if data.EthDeviceMask != "" {
		// FIXME: in the case of rumprun & k8s, we need to explicitly set the mask
		// to an inclusive value (eg 1 or 0), as NetBSD complains and does not set the default gw
		// if it is not reachable from the IP address directly.
		mask, err := subnetMaskToCIDR(SubnetMask125)
		if err != nil {
			return err
		}
		r.Net.Interface = "ukvmif0"
		r.Net.Cloner = "True"
		r.Net.Type = "inet"
		r.Net.Method = "static"
		r.Net.Address = data.EthDeviceIP
		r.Net.Mask = fmt.Sprintf("%d", mask)
		r.Net.Gateway = data.EthDeviceGateway
	}

	r.Blk.Source = "etfs"
	r.Blk.Path = "/dev/ld0a"
	r.Blk.FsType = "blk"
	r.Blk.Mountpoint = "/data"

	r.Command = strings.Join(data.CmdLine, " ")

	return nil
}

func newRumprun() *Rumprun {
	rumprunStruct := new(Rumprun)

	return rumprunStruct
}
