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

// There is no need for any changes here yet.
func (l *Linux) MonitorNetCli(_ string) string {
	return ""
}

// We have not managed to make Unikraft run with block yet.
func (l *Linux) MonitorBlockCli(_ string) string {
	return ""
}

// There are no generic CLI hypervisor options for Unikraft yet.
func (l *Linux) MonitorCli(_ string) string {
	return ""
}


func (l *Linux) CommandString() (string, error) {
	//return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw loglevel=15 nokaslr init=/guest_start.sh %s %s %s",
	//	l.Net.Address,
	//	l.Net.Gateway,
	//	l.Command), nil
	//return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw quiet loglevel=0 nokaslr init=%s",
	return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw loglevel=15 nokaslr ip=%s::%s:%s:urunc:eth0:off init=%s",
		l.Net.Address,
		l.Net.Gateway,
		l.Net.Mask,
		l.Command), nil
	//return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw loglevel=14 nokaslr init=%s",
	//	l.Command), nil
}

func (l *Linux) SupportsBlock() bool {
	return true
}

func (l *Linux) SupportsFS(_ string) bool {
	return true
}

func (l *Linux) Init(data UnikernelParams) error {
	// if there are no spaces in the command line, then
	// we assume that there was one word (appname) in the command line
	// Otherwise, we use the first word as the name of the app
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
