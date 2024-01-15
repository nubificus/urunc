// Copyright 2024 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package network

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"github.com/nubificus/urunc/internal/constants"
	"github.com/vishvananda/netlink"
)

var StaticIPAddr = fmt.Sprintf("%s/24", constants.StaticNetworkTapIP)

type StaticNetwork struct {
}

func (n StaticNetwork) NetworkSetup() (*UnikernelNetworkInfo, error) {
	err := ensureEth0Exists()
	if err != nil {
		netlog.Error("failed to find eth0 interface in current netns")
		return nil, err
	}
	redirectLink, err := netlink.LinkByName(DefaultInterface)
	if err != nil {
		netlog.Errorf("failed to find %s  interface", DefaultInterface)
		return nil, err
	}
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		return nil, err
	}
	newTapName := strings.ReplaceAll(DefaultTap, "X", "0")
	newTapDevice, err := createTapDevice(newTapName, redirectLink.Attrs().MTU, uid, gid)
	if err != nil {
		return nil, err
	}
	ipn, err := netlink.ParseAddr(StaticIPAddr)
	if err != nil {
		return nil, err
	}
	err = netlink.AddrReplace(newTapDevice, ipn)
	if err != nil {
		return nil, err
	}
	err = netlink.LinkSetUp(newTapDevice)
	if err != nil {
		return nil, err
	}
	return &UnikernelNetworkInfo{
		TapDevice: newTapDevice.Attrs().Name,
		EthDevice: Interface{
			IP:             constants.StaticNetworkUnikernelIP,
			DefaultGateway: constants.StaticNetworkTapIP,
			Mask:           "255.255.255.0",
			Interface:      "eth0", // or tap0_urunc?
			MAC:            redirectLink.Attrs().HardwareAddr.String(),
		},
	}, nil
}
