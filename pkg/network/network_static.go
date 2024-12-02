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
	"bytes"
	"fmt"
	"strings"
	"os/exec"

	"github.com/nubificus/urunc/internal/constants"
	"github.com/vishvananda/netlink"
)

var StaticIPAddr = fmt.Sprintf("%s/24", constants.StaticNetworkTapIP)

type StaticNetwork struct {
}

func setNATRule(iface string) error {
	var args []string
	var stdout, stderr bytes.Buffer

	path, err := exec.LookPath("iptables")
	if err != nil {
		return err
	}

	args = append(args, "iptables")
	args = append(args, "-t")
	args = append(args, "nat")
	args = append(args, "-A")
	args = append(args, "POSTROUTING")
	args = append(args, "-o")
	args = append(args, iface)
	args = append(args, "-j")
	args = append(args, "MASQUERADE")
	args = append(args, "--wait")
	args = append(args, "1")

	cmd := exec.Cmd {
		Path:   path,
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	err = cmd.Run()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			return fmt.Errorf("iptables command %s failed: %s", cmd.String(), stderr.String())
		default:
			return err
		}
	}

	return nil
}
// iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE --wait 1
// iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

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
	err = setNATRule(DefaultInterface)
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
