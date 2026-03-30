//go:build linux

package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

const ipamFile = "/var/run/mydocker/ipam.json"

type IPAM struct {
	Subnet    string
	Allocated map[string]string
}

func loadIPAM() (*IPAM, error) {
	ipam := &IPAM{
		Subnet:    "172.18.0.0/24",
		Allocated: make(map[string]string),
	}
	data, err := os.ReadFile(ipamFile)
	if os.IsNotExist(err) {
		return ipam, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, ipam); err != nil {
		return nil, err
	}
	return ipam, nil
}

func saveIPAM(ipam *IPAM) error {
	if err := os.MkdirAll("/var/run/mydocker", 0755); err != nil {
		return err
	}
	data, err := json.Marshal(ipam)
	if err != nil {
		return err
	}
	return os.WriteFile(ipamFile, data, 0644)
}

func AllocateIP(containerID string) (net.IP, error) {
	ipam, err := loadIPAM()
	if err != nil {
		return nil, err
	}

	_, subnet, _ := net.ParseCIDR(ipam.Subnet)
	for ip := cloneIP(subnet.IP); subnet.Contains(ip); incrementIP(ip) {
		last := ip[len(ip)-1]
		if last < 2 || last > 254 {
			continue
		}
		ipStr := ip.String()
		if _, used := ipam.Allocated[ipStr]; !used {
			ipam.Allocated[ipStr] = containerID
			if err := saveIPAM(ipam); err != nil {
				return nil, err
			}
			return cloneIP(ip), nil
		}
	}
	return nil, fmt.Errorf("no available IP address")
}

func ReleaseIP(containerID string) error {
	ipam, err := loadIPAM()
	if err != nil {
		return err
	}

	for ip, id := range ipam.Allocated {
		if id == containerID {
			delete(ipam.Allocated, ip)
		}
	}
	return saveIPAM(ipam)
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}
