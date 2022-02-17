/*
@Time :    2022/2/17 23:17
@Author :  liuzhi
@File :    network
@Software: GoLand
*/

package network

import (
	"github.com/vishvananda/netlink"
	"net"
)

// Network 定义网络基本模型
type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 网络地址端，比如 10.10.1.0/24
	Driver  string     // 网络驱动名
}

type Endpoint struct {
	ID     string `json:"id"`
	Device netlink.Veth
}
