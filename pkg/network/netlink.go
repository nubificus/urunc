package network

import (
	"github.com/vishvananda/netlink"
)

// NetLinkProvider defines an interface to abstract netlink
type NetLinkProvider interface {
	NewLinkAttrs() netlink.LinkAttrs
	LinkAdd(netlink.Link) error
	LinkSetMTU(netlink.Link, int) error
	QdiscAdd(netlink.Qdisc) error
	FilterAdd(netlink.Filter) error
	ParseAddr(string) (*netlink.Addr, error)
	AddrReplace(netlink.Link, *netlink.Addr) error
	LinkSetUp(netlink.Link) error
}

// realNetLink is a real implementation of NetLinkProvider.
type realNetLink struct{}

// NewLinkAttrs returns LinkAttrs structure filled with default values
func (r realNetLink) NewLinkAttrs() netlink.LinkAttrs {
	return netlink.NewLinkAttrs()
}

// LinkAdd adds a new link device. The type and features of the device are taken from the parameters in the link object. Equivalent to: `ip link add $link`
func (r realNetLink) LinkAdd(l netlink.Link) error {
	return netlink.LinkAdd(l)
}

// LinkSetMTU sets the mtu of the link device. Equivalent to: `ip link set $link mtu $mtu`
func (r realNetLink) LinkSetMTU(l netlink.Link, m int) error {
	return netlink.LinkSetMTU(l, m)
}

// QdiscAdd will add a qdisc to the system. Equivalent to: `tc qdisc add $qdisc`
func (r realNetLink) QdiscAdd(q netlink.Qdisc) error {
	return netlink.QdiscAdd(q)
}

// FilterAdd will add a filter to the system. Equivalent to: `tc filter add $filter`
func (r realNetLink) FilterAdd(f netlink.Filter) error {
	return netlink.FilterAdd(f)
}

// ParseAddr parses the string representation of an address in the form $ip/$netmask $label. The label portion is optional
func (r realNetLink) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}

// AddrReplace will replace (or, if not present, add) an IP address on a link device.
//
// Equivalent to: `ip addr replace $addr dev $link`
//
// If `addr` is an IPv4 address and the broadcast address is not given, it will be automatically computed based on the IP mask if /30 or larger.
func (r realNetLink) AddrReplace(l netlink.Link, a *netlink.Addr) error {
	return netlink.AddrReplace(l, a)
}

// LinkSetUp enables the link device. Equivalent to: `ip link set $link up`
func (r realNetLink) LinkSetUp(l netlink.Link) error {
	return netlink.LinkSetUp(l)
}
