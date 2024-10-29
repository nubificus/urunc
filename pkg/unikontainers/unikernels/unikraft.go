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
	"errors"
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
)

const UnikraftUnikernel string = "unikraft"
const UnikraftCompatVersion string = "0.16.1"

var ErrUndefinedVersion = errors.New("version is undefined, using default version")
var ErrVersionParsing = errors.New("failed to parse provided version, using default version")

type Unikraft struct {
	AppName string
	Command string
	Net     UnikraftNet
	VFS     UnikraftVFS
	Version string
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
	u.Command = strings.TrimSpace(data.CmdLine)
	firstSpace := strings.Index(u.Command, " ")
	if firstSpace > 0 {
		u.AppName = u.Command[:firstSpace]
		u.Command = strings.TrimLeft(u.Command, u.AppName)
	} else {
		u.AppName = "unikraft"
	}
	u.Version = data.Version

	// TODO: We need to add support for actual block devices (e.g. virtio-blk)
	// and sharedfs or any other Unikraft related ways to pass data to guest.
	switch data.RootFSType {
	case "initrd":
		u.VFS.RootFS = "vfs.rootfs=" + "initrd"
	default:
		u.VFS.RootFS = ""
	}
	return u.configureNet(data.EthDeviceIP, data.EthDeviceGateway, data.EthDeviceMask)
}

func (u *Unikraft) configureNet(ethDeviceIP, ethDeviceGateway, ethDeviceMask string) error {
	setCompatNetConfig := func() {
		u.Net.Address = "netdev.ipv4_addr=" + ethDeviceIP
		u.Net.Gateway = "netdev.ipv4_gw_addr=" + ethDeviceGateway
		u.Net.Mask = "netdev.ipv4_subnet_mask=" + ethDeviceMask
	}

	setCurrentNetConfig := func() {
		u.Net.Address = "netdev.ip=" + ethDeviceIP + "/24:" + ethDeviceGateway + ":8.8.8.8"
	}

	if u.Version == "" {
		setCurrentNetConfig()
		return ErrUndefinedVersion
	}

	unikernelVersion, err := version.NewVersion(u.Version)
	if err != nil {
		setCurrentNetConfig()
		return ErrVersionParsing
	}

	targetVersion, err := version.NewVersion(UnikraftCompatVersion)
	if err != nil {
		return fmt.Errorf("failed to parse default version: %w", err)
	}

	if unikernelVersion.GreaterThanOrEqual(targetVersion) {
		setCurrentNetConfig()
	} else {
		setCompatNetConfig()
	}
	return nil
}

func newUnikraft() *Unikraft {
	unikraftStruct := new(Unikraft)
	return unikraftStruct
}
