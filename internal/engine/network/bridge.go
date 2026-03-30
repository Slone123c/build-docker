//go:build linux

package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

const BridgeName = "mydocker0"
const BridgeCIDR = "172.18.0.1/24"

func SetupBridge() error {
	if _, err := netlink.LinkByName(BridgeName); err == nil {
		return nil
	}

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: BridgeName},
	}
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("create bridge %s error: %v", BridgeName, err)
	}

	ip, ipNet, _ := net.ParseCIDR(BridgeCIDR)
	ipNet.IP = ip
	addr := &netlink.Addr{
		IPNet: ipNet,
	}
	link, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return fmt.Errorf("lookup bridge %s: %w", BridgeName, err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("set bridge ip: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set bridge up: %w", err)
	}

	return nil
}

func TeardownBridge() error {
	link, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return nil
	}
	return netlink.LinkDel(link)
}
