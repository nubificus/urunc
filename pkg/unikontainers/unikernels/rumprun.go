// Copyright 2023 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikernels

import (
	"encoding/json"
	"fmt"
)

const RumprunUnikernel UnikernelType = "rumprun"

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

func (r Rumprun) CommandString() (string, error) {
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

func newRumprun(data UnikernelParams) (Rumprun, error) {
	tempBlk := RumprunBlk{
		Source:     "etfs",
		Path:       "/dev/ld0a",
		FsType:     "blk",
		Mountpoint: "/data",
	}

	// if EthDeviceMask is empty, there is no network support
	if data.EthDeviceMask == "" {
		return Rumprun{
			Command: data.CmdLine,
			Net:     RumprunNet{Mask: ""},
			Blk:     tempBlk,
		}, nil
	}

	// FIXME: in the case of rumprun & k8s, we need to explicitly set the mask
	// to an inclusive value (eg 1 or 0), as NetBSD complains and does not set the default gw
	// if it is not reachable from the IP address directly.
	mask, err := subnetMaskToCIDR(data.EthDeviceMask)
	if err != nil {
		return Rumprun{}, err
	}
	tempNet := RumprunNet{
		Interface: "ukvmif0",
		Cloner:    "True",
		Type:      "inet",
		Method:    "static",
		Address:   data.EthDeviceIP,
		Mask:      fmt.Sprintf("%d", mask),
		Gateway:   data.EthDeviceGateway,
	}
	return Rumprun{
		Command: data.CmdLine,
		Net:     tempNet,
		Blk:     tempBlk,
	}, nil
}
