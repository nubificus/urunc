// Copyright (c) 2023-2024, Nubificus LTD
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
)

const LinuxUnikernel string = "linux"

type Linux struct {
	Command string
	Net     LinuxNet
}

type LinuxNet struct {
	Address string
	Gateway string
	Mask string
}

func (l *Linux) CommandString() (string, error) {
	return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw loglevel=15 nokaslr ip=%s::%s:%s:urunc:eth0:off init=%s",
		l.Net.Address,
		l.Net.Gateway,
		l.Net.Mask,
		l.Command), nil
}

func (l *Linux) SupportsBlock() bool {
	return true
}

func (l *Linux) SupportsFS(_ string) bool {
	return true
}

func (l *Linux) MonitorNetCli(_ string) string {
	return ""
}

// Mewz does not seem to support virtio block or anu other kind of block/fs.
func (l *Linux) MonitorBlockCli(monitor string) string {
	switch monitor {
	case "qemu":
		bcli := " -device virtio-blk-pci,id=blk0,drive=hd0,scsi=off"
		bcli += " -drive format=raw,if=none,id=hd0,file="
		return bcli
	default:
		return ""
	}
}

// Mewz does not require any monitor specific cli option
func (l *Linux) MonitorCli(monitor string) string {
	switch monitor {
	case "qemu":
		return " -no-reboot -serial stdio -nodefaults"
	default:
		return ""
	}
}

func (l *Linux) Init(data UnikernelParams) error {
	l.Command = data.CmdLine

	l.Net.Address = data.EthDeviceIP
	l.Net.Gateway = data.EthDeviceGateway
	l.Net.Mask = data.EthDeviceMask

	return nil
}

func newLinux() *Linux {
	linuxStruct := new(Linux)
	return linuxStruct
}
