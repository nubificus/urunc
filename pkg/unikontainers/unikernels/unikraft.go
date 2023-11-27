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
)

const UnikraftUnikernel UnikernelType = "unikraft"

type UnikraftCliOpts struct {
	Command string
	Net     UnikraftNet
	Blk     UnikraftBlk
}

type UnikraftNet struct {
	Address string
	Mask    string
	Gateway string
}

type UnikraftBlk struct {
	RootFs string
	DevTag string // Will be used for the 9pfs
}

func newUnikraftCli(data UnikernelParams) (string, error) {
	var cli_opts UnikraftCliOpts

	cli_opts.Command = data.CmdLine

	cli_opts.Net.Address = "netdev.ipv4_addr=" + data.EthDeviceIP
	cli_opts.Net.Gateway = "netdev.ipv4_gw_addr=" + data.EthDeviceGateway
	cli_opts.Net.Mask = "netdev.ipv4_subnet_mask=" + data.EthDeviceMask

	// TODO: We need to add support for actual block devices (e.g. virtio-blk)
	// and sharedfs or any other Unikraft related ways to pass data to guest.
	cli_opts.Blk.RootFs = "vfs.rootfs=" + "initrd"
	cli_opts.Blk.DevTag = ""

	return fmt.Sprintf("%s %s %s %s %s -- %s", cli_opts.Net.Address,
						   cli_opts.Net.Gateway,
						   cli_opts.Net.Mask,
						   cli_opts.Blk.RootFs,
						   cli_opts.Blk.DevTag,
						   cli_opts.Command), nil
}
