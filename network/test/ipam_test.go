/*
@Time :    2022/2/20 11:16
@Author :  liuzhi
@File :    ipam_test
@Software: GoLand
*/

package network

import (
	"fmt"
	"my-container/network"
	"net"
	"strings"
	"testing"
)

func TestAllocate(t *testing.T) {
	cidr, ipNet, _ := net.ParseCIDR("192.168.1.100/24")
	// 192.168.1.100 192.168.1.0/24
	fmt.Println(cidr.String(), ipNet.String())

	size, bits := ipNet.Mask.Size()
	fmt.Println("ipNet信息: ", size, bits)
	fmt.Println(strings.Repeat("0", 1<<uint8(size-bits)))
}

func TestAllocateSubnetJsonLoad(t *testing.T) {
	ipam := &network.IPAM{
		SubnetAllocatorPath: "./subnet.json",
		Subnets:             &map[string]string{},
	}
	load, _ := ipam.ExternalLoadByTest()
	fmt.Println(load.Subnets)
	fmt.Println((*load.Subnets)["ip"])
}

func TestAllocateSetDefaultSubnet(t *testing.T) {
	ipam := &network.IPAM{
		SubnetAllocatorPath: "./subnet.json",
		Subnets:             &map[string]string{},
	}
	_, _ = ipam.ExternalLoadByTest()
	_, subnet, _ := net.ParseCIDR("192.168.1.100/24")
	size, bits := subnet.Mask.Size()
	(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(bits-size))

	for c := range (*ipam.Subnets)[subnet.String()] {
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
			ipAlloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipAlloc)
			ip := subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}

	fmt.Println(ipam.Subnets)
}
