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
	// DefaultInterface is the default network interface created inside the network namespace
	//
	// FIXME: Discover the veth endpoint name instead of using default "eth0". See: https://github.com/nubificus/urunc/issues/14
	DefaultInterface = "eth0"
	// DefaultTap is the default name for the tap device used by urunc.
	DefaultTap = "tapX_urunc"
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

// NetInterfaceFetcher defines an interface for fetching network interfaces.
type NetInterfaceFetcher interface {
	Interfaces() ([]net.Interface, error)
}

// realNetInterfaceFetcher is a real implementation of NetInterfaceFetcher.
type realNetInterfaceFetcher struct{}

// Interfaces retrieves the list of network interfaces using net.Interfaces().
func (r realNetInterfaceFetcher) Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// NewNetworkManager returns a new instance of a network manager based on the specified networkType.
// It supports two types of network managers: "static" and "dynamic".
// Returns a Manager interface and nil error if networkType is "static" or "dynamic".
// Returns nil and an error if networkType is not supported.
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

// getTapIndex counts and returns the number of TAP interfaces present in the current network namespace.
// It uses net.Interfaces() to fetch all network interfaces and counts those whose names contain "tap".
// If an error occurs while fetching interfaces, it returns 0 and the error.
// If the number of TAP interfaces exceeds 255, it returns the count and an error indicating the limit exceeded.
func getTapIndex(fetcher NetInterfaceFetcher) (int, error) {
	ifaces, err := fetcher.Interfaces()
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

// createTapDevice creates a TAP (L2) network device with the specified name, MTU, owner UID, and owner GID.
// It sets up a single queue tap device with vnet header parsing enabled.
//
// Returns the created netlink.Link representing the TAP device upon success, or an error if any operation fails.
func createTapDevice(name string, mtu int, ownerUID, ownerGID int) (netlink.Link, error) {
	netlinkProvider := new(realNetLink)
	tapLinkAttrs := netlinkProvider.NewLinkAttrs()
	tapLinkAttrs.Name = name
	tapLink := &netlink.Tuntap{
		LinkAttrs: tapLinkAttrs,

		// We want a tap device (L2) as opposed to a tun (L3)
		Mode: netlink.TUNTAP_MODE_TAP,

		// Firecracker does not support multiqueue tap devices at this time:
		// https://github.com/firecracker-microvm/firecracker/issues/750
		Queues: 1,

		Flags: netlink.TUNTAP_ONE_QUEUE | // single queue tap device
			netlink.TUNTAP_VNET_HDR, // parse vnet headers added by the vm's virtio_net implementation
	}

	err := netlinkProvider.LinkAdd(tapLink)
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

	err = netlinkProvider.LinkSetMTU(tapLink, mtu)
	if err != nil {
		return nil, fmt.Errorf("failed to set tap device MTU to %d: %w", mtu, err)
	}

	return tapLink, nil
}

// ensureEth0Exists checks all network interfaces in the current network namespace and returns
// nil if the eth0 interface is present, or an error if not.
//
// It returns an error if there is a problem retrieving the network interfaces or if the eth0
// interface is not found
func ensureEth0Exists(fetcher NetInterfaceFetcher) error {
	ifaces, err := fetcher.Interfaces()
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

// addIngressQdisc adds an ingress qdisc to the specified network link.
// It returns an error if the operation fails.
//
// link: The network link to which the ingress qdisc will be added.
func addIngressQdisc(link netlink.Link) error {
	netlinkProvider := new(realNetLink)
	ingress := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}
	return netlinkProvider.QdiscAdd((ingress))
}

func addRedirectFilter(source netlink.Link, target netlink.Link) error {
	netlinkProvider := new(realNetLink)
	return netlinkProvider.FilterAdd(&netlink.U32{
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

func networkSetup(tapName string, ipAdrress string, redirectLink netlink.Link, addTCRules bool) (netlink.Link, error) {
	err := ensureEth0Exists(realNetInterfaceFetcher{})
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
	netLinkProvider := new(realNetLink)
	ipn, err := netLinkProvider.ParseAddr(ipAdrress)
	if err != nil {
		return nil, err
	}
	err = netLinkProvider.AddrReplace(newTapDevice, ipn)
	if err != nil {
		return nil, err
	}

	err = netLinkProvider.LinkSetUp(newTapDevice)
	if err != nil {
		return nil, err
	}
	return newTapDevice, nil
}
