/*
@Time :    2022/2/17 23:17
@Author :  liuzhi
@File :    network
@Software: GoLand
*/

package network

import (
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
)

// Network 定义网络基本模型
type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 网络地址端，比如 10.10.1.0/24
	Driver  string     // 网络驱动名
}

// Endpoint 网络端点
// tips Go json对象在定义的时候，要声明成员变量是大写才能导出，也就是才会被序列化，可以指定序列化后的名称，一般就是转回驼峰或重写名称
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portMapping"`
	Network     *Network
}

// NetDriver 定义网络驱动的接口
type NetDriver interface {
	// Name 驱动名
	Name() string

	// Create 创建网络
	Create(subnet string, name string) (*Network, error)

	// Delete 删除网络
	Delete(network Network) error

	// Connect 连接容器端点到网络
	Connect(network *Network, endpoint *Endpoint) error

	// Disconnect 移除连接端点
	Disconnect(network Network, endpoint *Endpoint) error
}

func CreateNetwork(driver, subnet, name string) error {
	// 通过 ParseCIDR 转换网段字符串
	// For example, ParseCIDR("192.0.2.1/24") returns the IP address
	// 192.0.2.1 and the network 192.0.2.0/24.
	cidr, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		panic("subnet 参数非法")
	}
	log.Debug("网络转换：cidr -> ", cidr)
	log.Debug("网络转换：ipNet -> ", ipNet)
	// 通过IPAM组件分配IP，获取网段中的第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocator(ipNet)
	if err != nil {
		return err
	}
	ipNet.IP = gatewayIp

	return nil
}
