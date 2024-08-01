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
	"fmt"
	"strings"
)

const UnikraftUnikernel UnikernelType = "unikraft"

type Unikraft struct {
	AppName string
	Command string
	Net     UnikraftNet
	VFS     UnikraftVFS
}

type UnikraftNet struct {
	Address string
	Mask    string
	Gateway string
}

type UnikraftVFS struct {
	RootFS string
}

func (u *Unikraft) CommandString() (string, error) {
	return fmt.Sprintf("%s %s %s %s %s -- %s", u.AppName,
		u.Net.Address,
		u.Net.Gateway,
		u.Net.Mask,
		u.VFS.RootFS,
		u.Command), nil
}

func (u *Unikraft) SupportsBlock() bool {
	return false
}

func (u *Unikraft) SupportsFS(_ string) bool {
	return false
}

func (u *Unikraft) Init(data UnikernelParams) error {
	// if there are no spaces in the command line, then
	// we assume that there was one word (appname) in the command line
	// Otherwise, we use the first word as the name of the app
	appName := u.Command
	firstSpace := strings.Index(data.CmdLine, " ")
	if firstSpace > 0 {
		appName = data.CmdLine[:firstSpace]
		u.Command = strings.TrimLeft(data.CmdLine, appName)
	}
	u.AppName = appName

	u.Net.Address = "netdev.ipv4_addr=" + data.EthDeviceIP
	u.Net.Gateway = "netdev.ipv4_gw_addr=" + data.EthDeviceGateway
	u.Net.Mask = "netdev.ipv4_subnet_mask=" + data.EthDeviceMask

	// TODO: We need to add support for actual block devices (e.g. virtio-blk)
	// and sharedfs or any other Unikraft related ways to pass data to guest.
	switch data.RootFSType {
	case "initrd":
		u.VFS.RootFS = "vfs.rootfs=" + "initrd"
	default:
		u.VFS.RootFS = ""
	}

	return nil
}

func newUnikraft() *Unikraft {
	unikraftStruct := new(Unikraft)
	return unikraftStruct
}
