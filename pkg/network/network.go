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
	"errors"
	"fmt"
	"net"
	"os/user"
	"strconv"
	"strings"

	"github.com/jackpal/gateway"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	DefaultInterface = "eth0" // FIXME: Discover the veth endpoint name instead of using default "eth0". See: https://github.com/nubificus/urunc/issues/14
	DefaultTap       = "tapX_urunc"
)

var netlog = logrus.WithField("subsystem", "network")

type UnikernelNetworkInfo struct {
	TapDevice string
	EthDevice Interface
}
type Manager interface {
	NetworkSetup() (*UnikernelNetworkInfo, error)
}

type Interface struct {
	IP             string
	DefaultGateway string
	Mask           string
	Interface      string
	MAC            string
}

func NewNetworkManager(networkType string) (Manager, error) {
	switch networkType {
	case "static":
		return &StaticNetwork{}, nil
	case "dynamic":
		return &DynamicNetwork{}, nil
	default:
		return nil, fmt.Errorf("network manager %s not supported", networkType)

	}
}

func getTapIndex() (int, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}
	tapCount := 0
	for _, iface := range ifaces {
		if strings.Contains(iface.Name, "tap") {
			tapCount++
		}
	}
	if tapCount > 255 {
		return tapCount, fmt.Errorf("TAP interfaces count higher than 255")
	}
	return tapCount, nil
}

func createTapDevice(name string, mtu int, ownerUID, ownerGID int) (netlink.Link, error) {
	tapLinkAttrs := netlink.NewLinkAttrs()
	tapLinkAttrs.Name = name
	tapLink := &netlink.Tuntap{
		LinkAttrs: tapLinkAttrs,

		// We want a tap device (L2) as opposed to a tun (L3)
		Mode: netlink.TUNTAP_MODE_TAP,

		// Firecracker does not support multiqueue tap devices at this time:
		// https://github.com/firecracker-microvm/firecracker/issues/750
		Queues: 1,
		//Queues: 2,

		Flags:  netlink.TUNTAP_NO_PI |
			//netlink.TUNTAP_MULTI_QUEUE_DEFAULTS |
			netlink.TUNTAP_VNET_HDR, // parse vnet headers added by the vm's virtio_net implementation
	}

	err := netlink.LinkAdd(tapLink)
	if err != nil {
		return nil, fmt.Errorf("failed to create tap device: %w", err)
	}

	for _, tapFd := range tapLink.Fds {
		err = unix.IoctlSetInt(int(tapFd.Fd()), unix.TUNSETOWNER, ownerUID)
		if err != nil {
			return nil, fmt.Errorf("failed to set tap %s owner to uid %d: %w", name, ownerUID, err)
		}

		err = unix.IoctlSetInt(int(tapFd.Fd()), unix.TUNSETGROUP, ownerGID)
		if err != nil {
			return nil, fmt.Errorf("failed to set tap %s group to gid %d: %w", name, ownerGID, err)
		}
	}

	err = netlink.LinkSetMTU(tapLink, mtu)
	if err != nil {
		return nil, fmt.Errorf("failed to set tap device MTU to %d: %w", mtu, err)
	}

	return tapLink, nil
}

// ensureEth0Exists checks all network interfaces in current netns and returns
// nil if eth0 is present or ErrEth0NotFound if not
func ensureEth0Exists() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, iface := range ifaces {
		if iface.Name == DefaultInterface {
			return nil
		}
	}
	return errors.New("eth0 device not found")
}

func getInterfaceInfo(iface string) (Interface, error) {
	ief, err := net.InterfaceByName(iface)
	if err != nil {
		return Interface{}, err
	}
	IfMAC := ief.HardwareAddr.String()
	if IfMAC == "" {
		return Interface{}, fmt.Errorf("failed to get MAC address of %q", ief)
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return Interface{}, err
	}
	ipAddress := ""
	mask := ""
	netMask := net.IPMask{}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			ipAddress = ipNet.IP.String()
			// hexadecimal notation
			mask = ipNet.Mask.String()
			netMask = ipNet.Mask
			break
		}
	}
	if mask == "" {
		return Interface{}, fmt.Errorf("failed to find mask for %q", DefaultInterface)
	}
	// convert to decimal notation
	decimalParts := make([]string, len(netMask))
	for i, part := range netMask {
		decimalParts[i] = fmt.Sprintf("%d", part)
	}
	mask = strings.Join(decimalParts, ".")
	if ipAddress == "" {
		return Interface{}, fmt.Errorf("failed to find IPv4 address for %q", DefaultInterface)
	}
	gateway, err := gateway.DiscoverGateway()
	if err != nil {
		return Interface{}, err
	}
	return Interface{
		IP:             ipAddress,
		DefaultGateway: gateway.String(),
		Mask:           mask,
		Interface:      DefaultInterface,
		MAC:            IfMAC,
	}, nil
}

func addIngressQdisc(link netlink.Link) error {
	ingress := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}
	return netlink.QdiscAdd((ingress))
}

func addRedirectFilter(source netlink.Link, target netlink.Link) error {
	return netlink.FilterAdd(&netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: source.Attrs().Index,
			Parent:    netlink.MakeHandle(0xffff, 0),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_STOLEN,
				},
				MirredAction: netlink.TCA_EGRESS_REDIR,
				Ifindex:      target.Attrs().Index,
			},
		},
	})
}

func networkSetup(tapName string, ipAddress string, redirectLink netlink.Link, addTCRules bool) (netlink.Link, error) {
	err := ensureEth0Exists()
	// if eth0 does not exist in the namespace, the unikernel was spawned using ctr, so we skip the network setup
	if err != nil {
		netlog.Info("eth0 interface not found, assuming unikernel was spawned using ctr")
		return nil, nil
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
	newTapDevice, err := createTapDevice(tapName, redirectLink.Attrs().MTU, uid, gid)
	if err != nil {
		return nil, err
	}
	if addTCRules {
		err = addIngressQdisc(newTapDevice)
		if err != nil {
			return nil, err
		}
		err = addIngressQdisc(redirectLink)
		if err != nil {
			return nil, err
		}
		err = addRedirectFilter(newTapDevice, redirectLink)
		if err != nil {
			return nil, err
		}
		err = addRedirectFilter(redirectLink, newTapDevice)
		if err != nil {
			return nil, err
		}
	}
	ipn, err := netlink.ParseAddr(ipAddress)
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
	return newTapDevice, nil
}

func Cleanup(tapDevice string) error {
	netlog.Info("net cleanup called")
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, iface := range ifaces {
		netlog.Debugf("Discovered device %s", iface.Name)
	}
	tapLink, err := netlink.LinkByName(tapDevice)
	if err != nil {
		netlog.Errorf("Failed to get link %s by name: %v", tapDevice, err)
		return nil
	}
	err = deleteAllTCFilters(tapLink)
	if err != nil {
		netlog.Errorf("Failed to delete all TC filters: %v", err)
		return err
	}
	err = deleteAllQDiscs(tapLink)
	if err != nil {
		netlog.Errorf("Failed to delete all qdiscs: %v", err)
		return err
	}
	err = deleteTapDevice(tapLink)
	if err != nil {
		netlog.Errorf("Failed to delete link %s: %v", tapDevice, err)
	}
	return nil
}

func deleteIngressQdisc(link netlink.Link) error {
	qdiscs, err := netlink.QdiscList(link)
	if err != nil {
		return err
	}
	for _, qdisc := range qdiscs {
		if qdisc.Attrs().Parent == netlink.HANDLE_INGRESS && qdisc.Attrs().LinkIndex == link.Attrs().Index {
			err = netlink.QdiscDel(qdisc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteAllQDiscs(device netlink.Link) error {
	err := deleteIngressQdisc(device)
	if err != nil {
		return err
	}
	device, err = netlink.LinkByName(DefaultInterface)
	if err != nil {
		return err
	}
	err = deleteIngressQdisc(device)
	if err != nil {
		return err
	}
	return nil
}

func deleteAllTCFilters(device netlink.Link) error {
	var allFilters []netlink.Filter
	parent := uint32(netlink.HANDLE_ROOT)
	tapFilters, err := netlink.FilterList(device, parent)
	if err != nil {
		return nil
	}
	allFilters = append(allFilters, tapFilters...)

	device, err = netlink.LinkByName(DefaultInterface)
	if err != nil {
		return err
	}
	ethFilters, err := netlink.FilterList(device, parent)
	if err != nil {
		return err
	}
	allFilters = append(allFilters, ethFilters...)
	for _, filter := range allFilters {
		err = netlink.FilterDel(filter)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteTapDevice(device netlink.Link) error {
	err := netlink.LinkSetDown(device)
	if err != nil {
		netlog.Errorf("Failed to set link down: %v", err)
		return err
	}
	err = netlink.LinkDel(device)
	if err != nil {
		netlog.Errorf("Failed to delete link: %v", err)
		return err
	}
	return nil
}
