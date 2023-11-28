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

	"golang.org/x/sys/unix"

	"github.com/jackpal/gateway"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// FIXME: Add support for more interfaces. See: https://github.com/nubificus/urunc/issues/13
// FIXME: Discover the veth endpoint name instead of using default "eth0". See: https://github.com/nubificus/urunc/issues/14
const DefaultInterface = "eth0"
const DefaultTap = "tap0_urunc"

var ErrEth0NotFound = errors.New("eth0 device not found")
var netLog = logrus.WithField("subsystem", "network")

type UnikernelNetworkInfo struct {
	TapDevice string
	EthDevice InterfaceInfo
}

type InterfaceInfo struct {
	IP             string
	DefaultGateway string
	Mask           string
	Interface      string
	MAC	       string
}

func getInterfaceInfo(iface string) (InterfaceInfo, error) {
	ief, err := net.InterfaceByName(iface)
	if err != nil {
		return InterfaceInfo{}, err
	}

	IfMAC := ief.HardwareAddr.String()
	if IfMAC == "" {
		return InterfaceInfo{}, fmt.Errorf("failed to get MAC address of %q", iface)
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return InterfaceInfo{}, err
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
		return InterfaceInfo{}, fmt.Errorf("failed to find mask for %q", iface)
	}
	// convert to decimal notatio
	decimalParts := make([]string, len(netMask))
	for i, part := range netMask {
		decimalParts[i] = fmt.Sprintf("%d", part)
	}
	mask = strings.Join(decimalParts, ".")
	if ipAddress == "" {
		return InterfaceInfo{}, fmt.Errorf("failed to find IPv4 address for %q", iface)
	}
	gateway, err := gateway.DiscoverGateway()
	if err != nil {
		return InterfaceInfo{}, err
	}
	return InterfaceInfo{
		IP:             ipAddress,
		DefaultGateway: gateway.String(),
		Mask:           mask,
		Interface:      iface,
		MAC:		IfMAC,
	}, nil
}

// Setup creates a tap device and sets tc rules between veth interface inside the namespace to the tap device.
func Setup() (*UnikernelNetworkInfo, error) {
	vethDevName, err := GetVethDevice()
	if err == ErrEth0NotFound {
		netLog.Info("No eth0 device found")
		return nil, nil
	}
	if err != nil {
		netLog.WithError(err).Error("Couldn't find eth0 interface")
		return nil, err
	}
	redirectLink, err := GetLink(vethDevName)
	if err != nil {
		netLog.WithError(err).Error("Couldn't find eth0 link")
		return nil, err
	}
	ifaceInfo, err := getInterfaceInfo(vethDevName)
	if err != nil {
		netLog.WithError(err).Error("Couldn't find eth0 info")
		return nil, err
	}
	currentUser, err := user.Current()
	if err != nil {
		netLog.WithError(err).Error("Couldn't find current user")
		return nil, err
	}
	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		netLog.WithError(err).Error("Couldn't parse UID as int")
		return nil, err
	}
	gid, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		netLog.WithError(err).Error("Couldn't parse GID as int")
		return nil, err
	}
	tapLink, err := CreateTap(DefaultTap, redirectLink.Attrs().MTU, uid, gid)
	if err != nil {
		netLog.WithError(err).Error("Couldn't create tap device")
		return nil, err
	}
	netLog.WithField("tap", tapLink).Debug("Created TAP device")

	err = AddIngressQdisc(tapLink)
	if err != nil {
		return nil, err
	}

	err = AddIngressQdisc(redirectLink)
	if err != nil {
		return nil, err
	}

	err = AddRedirectFilter(tapLink, redirectLink)
	if err != nil {
		return nil, err
	}

	err = AddRedirectFilter(redirectLink, tapLink)
	if err != nil {
		return nil, err
	}
	return &UnikernelNetworkInfo{
		TapDevice: tapLink.Attrs().Name,
		EthDevice: ifaceInfo,
	}, nil
}

// Find all interfaces available in our netns.
// We are expecting to find an eth0 interface and a lo interface.
// The eth0 is the interface we need to use.
// If there are more interfaces, log all interfaces.
func GetVethDevice() (string, error) {
	ifaces, _ := net.Interfaces()
	if len(ifaces) > 2 {
		netLog.WithField("interfaces", ifaces).Debug("Found more than 2 interfaces")
	}
	for _, iface := range ifaces {
		netLog.Info("ifaces: ", iface)
		if iface.Name == DefaultInterface {
			return iface.Name, nil
		}
	}
	return "", ErrEth0NotFound

}

type LinkNotFoundError struct {
	device string
}

func (e LinkNotFoundError) Error() string {
	return fmt.Sprintf("did not find expected network device with name %q", e.device)
}

func GetLink(name string) (netlink.Link, error) {
	link, err := netlink.LinkByName(name)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil, &LinkNotFoundError{device: name}
	}
	return link, err
}

func CreateTap(name string, mtu int, ownerUID, ownerGID int) (netlink.Link, error) {
	tapLinkAttrs := netlink.NewLinkAttrs()
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

	err = netlink.LinkSetUp(tapLink)
	if err != nil {
		return nil, errors.New("failed to set tap up")
	}

	return tapLink, nil
}

func AddIngressQdisc(link netlink.Link) error {
	err := netlink.QdiscAdd(ingressQdisc(link))
	if err != nil {
		return fmt.Errorf("failed to add ingress qdisc to device %q: %w", link.Attrs().Name, err)
	}
	return nil
}

func ingressQdisc(link netlink.Link) netlink.Qdisc {
	return &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}
}

func RootFilterHandle() uint32 {
	return netlink.MakeHandle(0xffff, 0)
}

func AddRedirectFilter(sourceLink netlink.Link, targetLink netlink.Link) error {
	err := netlink.FilterAdd(&netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: sourceLink.Attrs().Index,
			Parent:    RootFilterHandle(),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_STOLEN,
				},
				MirredAction: netlink.TCA_EGRESS_REDIR,
				Ifindex:      targetLink.Attrs().Index,
			},
		},
	})
	if err != nil {
		err = fmt.Errorf(
			"failed to add u32 filter redirecting from device %q to device %q, does %q exist and have a qdisc attached to its ingress?",
			sourceLink.Attrs().Name, targetLink.Attrs().Name, sourceLink.Attrs().Name,
		)
	}

	return err
}

// FIXME: Remove
// func SubnetMaskToCIDR(subnetMask string) (int, error) {
// 	maskParts := strings.Split(subnetMask, ".")
// 	if len(maskParts) != 4 {
// 		return 0, fmt.Errorf("invalid subnet mask format")
// 	}

// 	var cidr int
// 	for _, part := range maskParts {
// 		val, err := strconv.Atoi(part)
// 		if err != nil || val < 0 || val > 255 {
// 			return 0, fmt.Errorf("invalid subnet mask value: %s", part)
// 		}

// 		// Convert part to binary and count the number of 1 bits
// 		binary := fmt.Sprintf("%08b", val)
// 		cidr += strings.Count(binary, "1")
// 	}

// 	return cidr, nil
// }
