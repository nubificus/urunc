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
	Command string
	Net     LinuxNet
}

type LinuxNet struct {
	Address string
	Gateway string
}

func (l *Linux) CommandString() (string, error) {
	return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw loglevel=15 nokaslr init=/guest_start.sh %s %s %s",
		l.Net.Address,
		l.Net.Gateway,
		l.Command), nil
}

func (l *Linux) SupportsBlock() bool {
	return true
}

func (l *Linux) SupportsFS(_ string) bool {
	return false
}

func (l *Linux) Init(data UnikernelParams) error {
	// if there are no spaces in the command line, then
	// we assume that there was one word (appname) in the command line
	// Otherwise, we use the first word as the name of the app
	l.Command = data.CmdLine

	l.Net.Address = strings.ReplaceAll(data.EthDeviceIP, ".", " ") + " 24"
	l.Net.Gateway = strings.ReplaceAll(data.EthDeviceGateway, ".", " ")

	return nil
}

func newLinux() *Linux {
	linuxStruct := new(Linux)
	return linuxStruct
}
