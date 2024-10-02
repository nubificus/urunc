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

package network

import (
	"fmt"
	"strings"

	"github.com/nubificus/urunc/internal/constants"
	"github.com/vishvananda/netlink"
)

var StaticIPAddr = fmt.Sprintf("%s/24", constants.StaticNetworkTapIP)

type StaticNetwork struct {
}

func (n StaticNetwork) NetworkSetup() (*UnikernelNetworkInfo, error) {
	newTapName := strings.ReplaceAll(DefaultTap, "X", "0")
	addTCRules := false
	redirectLink, err := netlink.LinkByName(DefaultInterface)
	if err != nil {
		netlog.Errorf("failed to find %s interface", DefaultInterface)
		return nil, err
	}
	newTapDevice, err := networkSetup(newTapName, StaticIPAddr, redirectLink, addTCRules)
	if err != nil {
		return nil, err
	}
	return &UnikernelNetworkInfo{
		TapDevice: newTapDevice.Attrs().Name,
		EthDevice: Interface{
			IP:             constants.StaticNetworkUnikernelIP,
			DefaultGateway: constants.StaticNetworkTapIP,
			Mask:           "255.255.255.0",
			Interface:      DefaultInterface, // or tap0_urunc?
			MAC:            redirectLink.Attrs().HardwareAddr.String(),
		},
	}, nil
}
