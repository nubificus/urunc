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
	"strings"
)

const LinuxUnikernel string = "linux"

type Linux struct {
	App          string
	Command      string
	Net          LinuxNet
	RootFsType   string
}

type LinuxNet struct {
	Address string
	Gateway string
	Mask    string
}

func (l *Linux) CommandString() (string, error) {
	rdinit := ""
	bootParams := "panic=-1 console=ttyS0 nokaslr loglevel=15"
	if l.RootFsType == "block" {
		rootParams := "root=/dev/vda rw"
		bootParams += " " + rootParams
	} else if l.RootFsType == "initrd" {
		rootParams := "root=/dev/ram0 rw"
		rdinit = "rd"
		bootParams += " " + rootParams
	}
	if l.Net.Address != "" {
		netParams := fmt.Sprintf("ip=%s::%s:%s:urunc:eth0:off",
			l.Net.Address,
			l.Net.Gateway,
			l.Net.Mask)
		bootParams += " " + netParams
	}
	if l.App != "" {
		initParams := rdinit + "init=" + l.App + " -- " + l.Command
		bootParams += " " + initParams
	}

	return bootParams, nil
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
		bcli := " -device virtio-blk-pci,id=blk0,drive=hd0"
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
	// we use the first argument in the cli args as the app name and the
	// rest as its arguments.
	if len(data.CmdLine) == 0 {
		return fmt.Errorf("No init was specified")
	} else if len(data.CmdLine) == 1 {
		l.App = data.CmdLine[0]
		l.Command = ""
	} else {
		l.App = data.CmdLine[0]
		l.Command = strings.Join(data.CmdLine[1:], " ")
	}

	l.Net.Address = data.EthDeviceIP
	l.Net.Gateway = data.EthDeviceGateway
	l.Net.Mask = data.EthDeviceMask

	l.RootFsType = data.RootFSType
	return nil
}

func newLinux() *Linux {
	linuxStruct := new(Linux)
	return linuxStruct
}
