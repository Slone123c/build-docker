//go:build linux

package network

import (
	"fmt"
	"os/exec"
)

func SetupIPTables() error {
	// 先检查规则是否已存在（避免重复添加）
	checkCmd := exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING",
		"-s", "172.18.0.0/24",
		"!", "-o", BridgeName,
		"-j", "MASQUERADE",
	)
	if checkCmd.Run() == nil {
		return nil // 规则已存在
	}

	// 添加NAT转发规则
	addCmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", "172.18.0.0/24",
		"!", "-o", BridgeName,
		"-j", "MASQUERADE",
	)
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables MASQUERADE: %s %w", out, err)
	}

	return exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
}
