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
	"fmt"
	"strings"
)

const MewzUnikernel string = "mewz"

type Mewz struct {
	Command string
	Net     MewzNet
}

type MewzNet struct {
	Address string
	Mask    int
	Gateway string
}

func (m *Mewz) CommandString() (string, error) {
	return fmt.Sprintf("ip=%s/%d gateway=%s ", m.Net.Address, m.Net.Mask,
		m.Net.Gateway), nil
}

func (m *Mewz) SupportsBlock() bool {
	return false
}

func (m *Mewz) SupportsFS(_ string) bool {
	return false
}

func (m *Mewz) MonitorNetCli(monitor string) string {
	switch monitor {
	case "qemu":
		ncli := " -device virtio-net-pci,netdev=net0,disable-legacy=on,disable-modern=off"
		ncli += " -netdev tap,script=no,downscript=no,id=net0,ifname="
		return ncli
	default:
		return ""
	}
}

// Mewz does not seem to support virtio block or anu other kind of block/fs.
func (m *Mewz) MonitorBlockCli(_ string) string {
	return ""
}

// Mewz does not require any monitor specific cli option
func (m *Mewz) MonitorCli(monitor string) string {
	switch monitor {
	case "qemu":
		return " -no-reboot -device isa-debug-exit,iobase=0x501,iosize=2"
	default:
		return ""
	}
}

func (m *Mewz) Init(data UnikernelParams) error {
	var mask int
	if data.EthDeviceMask != "" {
		var err error
		mask, err = subnetMaskToCIDR(data.EthDeviceMask)
		if err != nil {
			return err
		}
	} else {
		mask = 24
	}
	m.Command = strings.Join(data.CmdLine, " ")
	m.Net.Address = data.EthDeviceIP
	m.Net.Gateway = data.EthDeviceGateway
	m.Net.Mask = mask

	return nil
}

func newMewz() *Mewz {
	mewzStruct := new(Mewz)
	return mewzStruct
}
