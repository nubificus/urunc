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
	AppName string
	Command string
	Net     LinuxNet
	VFS     LinuxVFS
	Version string
}

type LinuxNet struct {
	Address string
	Mask    string
	Gateway string
}

type LinuxVFS struct {
	RootFS string
}

func (l *Linux) CommandString() (string, error) {
	return fmt.Sprintf("panic=-1 console=ttyS0 root=/dev/vda rw loglevel=15 nokaslr init=%s",
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

	//l.Net.Address = "netdev.ip=" + data.EthDeviceIP + "/24:" + data.EthDeviceGateway + ":8.8.8.8"
	//// TODO: We need to add support for actual block devices (e.g. virtio-blk)
	//// and sharedfs or any other Linux related ways to pass data to guest.
	//if data.RootFSType == "initrd" {
	//	// TODO: This needs better handling. We need to revisit this
	//	// when we better understand all the available options for
	//	// passing info inside linux unikernels.
	//	l.VFS.RootFS = "vfs.fstab=[ \"initrd0:/:extract:::\" ]"
	//} else {
	//	l.VFS.RootFS = ""
	//}

	return nil
}

func newLinux() *Linux {
	linuxStruct := new(Linux)
	return linuxStruct
}
