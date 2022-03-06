/*
@Time :    2022/3/1 21:50
@Author :  liuzhi
@File :    bridge
@Software: GoLand
*/

package network

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
)

// BridgeNetworkDriver 网络驱动实现，类似Bridge网桥
// 定义一个空struct，实现NetDriver的接口
type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	// 解析字符串，获取网关ip，网络ip
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	// 配置Linux Bridge
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("error init bridge: %v", err)
	}

	return n, err
}

func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	// 删除网络对应的 Linux Bridge 设备
	return netlink.LinkDel(br)
}

func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}

	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}

	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	// 创建 Bridge 设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		log.Errorf("创建 bridge 设备")
		return fmt.Errorf("error add bridge： %s, Error: %v", bridgeName, err)
	}
	// 设置 Bridge 设备的地址和路由
	// 这里的 gatewayIP 就是 *net.IPNet
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("error assigning address: %s on bridge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}
	// 启动 Bridge 设备启动
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("error set bridge up: %s, Error: %v", bridgeName, err)
	}
	// 设置 iptables SNAT 规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}
