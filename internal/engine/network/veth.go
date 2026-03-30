package network

import (
	"fmt"
	"net"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func SetupVeth(containerPid int, containerIP net.IP, containerID string) error {

	hostVethName := "veth-" + containerID[:5]
	containerVethName := "cv-" + containerID[:5]

	// 创建 veth 对
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: hostVethName},
		PeerName:  containerVethName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("failed to create veth pair (%s <-> %s): %v", hostVethName, containerVethName, err)
	}

	br, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return fmt.Errorf("failed to find bridge: %v", err)
	}

	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return fmt.Errorf("failed to find host veth: %v", err)
	}

	// 将 host veth 加入 bridge
	if err := netlink.LinkSetMaster(hostVeth, br); err != nil {
		return fmt.Errorf("failed to attach host veth to bridge: %v", err)
	}

	// 启动 host veth
	if err := netlink.LinkSetUp(hostVeth); err != nil {
		return fmt.Errorf("failed to bring up host veth: %v", err)
	}

	// 移动 container veth 到容器命名空间
	containerVeth, err := netlink.LinkByName(containerVethName)
	if err != nil {
		return fmt.Errorf("failed to find container veth: %v", err)
	}

	if err := netlink.LinkSetNsPid(containerVeth, containerPid); err != nil {
		return fmt.Errorf("failed to move container veth to container namespace: %v", err)
	}

	// 接下来配置容器内的网络
	if err := configureContainerNetwork(containerPid, containerIP, containerVethName); err != nil {
		return fmt.Errorf("failed to configure container network: %v", err)
	}

	return nil

}

func configureContainerNetwork(containerPid int, ip net.IP, vethName string) error {
	hostNs, err := netns.Get()
	if err != nil {
		return fmt.Errorf("get host ns: %w", err)
	}
	defer hostNs.Close()

	contNSPath := fmt.Sprintf("/proc/%d/ns/net", containerPid)
	contNS, err := netns.GetFromPath(contNSPath)
	if err != nil {
		return fmt.Errorf("get container ns: %w", err)
	}
	defer contNS.Close()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 切换到容器网络命名空间
	if err := netns.Set(contNS); err != nil {
		return fmt.Errorf("set container ns: %w", err)
	}

	defer netns.Set(hostNs)

	link, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to find container veth: %v", err)
	}

	// 将容器侧 veth 改名为 eth0
	if err := netlink.LinkSetName(link, "eth0"); err != nil {
		return fmt.Errorf("failed to rename container veth to eth0: %v", err)
	}

	// 获取更名后的 link
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return fmt.Errorf("failed to find eth0: %v", err)
	}

	_, subnet, _ := net.ParseCIDR(BridgeCIDR)

	ipNet := &net.IPNet{IP: ip, Mask: subnet.Mask}
	addr := &netlink.Addr{IPNet: ipNet}
	if err := netlink.AddrAdd(eth0, addr); err != nil {
		return fmt.Errorf("set container ip: %w", err)
	}

	if err := netlink.LinkSetUp(eth0); err != nil {
		return fmt.Errorf("set eth0 up: %w", err)
	}

	lo, _ := netlink.LinkByName("lo")
	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("set lo up: %w", err)
	}

	// 添加默认路由
	gwIP, _, _ := net.ParseCIDR(BridgeCIDR)
	defaultRoute := &netlink.Route{
		LinkIndex: eth0.Attrs().Index,
		Gw:        gwIP,
	}
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("add default route: %w", err)
	}

	return nil
}

func TeardownVeth(containerID string) error {
	hostVethName := "veth-" + containerID[:5]
	link, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return fmt.Errorf("failed to find host veth: %v", err)
	}

	return netlink.LinkDel(link)

}
