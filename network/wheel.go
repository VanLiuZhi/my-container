/*
@Time :    2022/3/1 22:21
@Author :  liuzhi
@File :    wheel
@Software: GoLand
*/

package network

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"my-container/container"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func ConfigEndpointIpAddressAndRoute(ep *Endpoint, containerInfo *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	defer enterContainerNetNameSpace(&peerLink, containerInfo)()

	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}

	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

func ConfigPortMapping(ep *Endpoint, containerInfo *container.ContainerInfo) error {
	log.Info("端口映射为：", containerInfo.PortMapping)
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error, %v", pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		// err := cmd.Run()
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}

func enterContainerNetNameSpace(enLink *netlink.Link, containerInfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", containerInfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	netNamespace, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns, %v", err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns, %v", err)
	}
	return func() {
		_ = netns.Set(netNamespace)
		_ = netNamespace.Close()
		runtime.UnlockOSThread()
		_ = f.Close()
	}
}

// 创建 link bridge
func createBridgeInterface(bridgeName string) error {
	// 先检查是否已经创建这个设备，通过 net.InterfaceByName 查询
	_, err := net.InterfaceByName(bridgeName)
	// 已存在则返回 nil，报错返回 err
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// new 一个 netlink 的 link 对象
	la := netlink.NewLinkAttrs()
	// 设置其 name 为 bridgeName
	la.Name = bridgeName

	// 创建一个 netlink的Bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
	// 通过 LinkAdd 创建 虚拟网络设备Bridge。相当于 ip link add xxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

// Set the IP addr of a netlink interface
// 设置一个网络接口的IP地址
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var ipLinkDriver netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		// 查找网络接口
		ipLinkDriver, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}
	// 将字符串转换成 net.ipNet 对象，并设置ip(参考源码)
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	// netlink.AddrAdd，给网络接口配置地址，相当于 ip link add xxx
	// 同时如果配置了地址所在网段的信息，例如 192.168.0.0/24
	// 还会配置路由表 192.168.0.0/24 转发到这个 bridge 的网络接口上
	addr := &netlink.Addr{IPNet: ipNet, Peer: ipNet, Label: "", Flags: 0, Scope: 0, Broadcast: nil}
	return netlink.AddrAdd(ipLinkDriver, addr)
}

// 设置网络接口为UP状态
func setInterfaceUP(interfaceName string) error {
	// 查询
	ipLinkDriver, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("error retrieving a link named [ %s ]: %v", ipLinkDriver.Attrs().Name, err)
	}
	// 启动，相当于 ip link set xxx up
	if err := netlink.LinkSetUp(ipLinkDriver); err != nil {
		return fmt.Errorf("error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// 设置 iptables 对应 bridge 的 MASQUERADE 规则 (地址伪装，也就是SNAT)
/*
iptables -t nat -A POSTROUTING
只要是从这个网桥上出来的包，都会对其做源IP的转换
保证了容器经过宿主机访问到宿主机外部网络请求的包转换成机器的IP，从而能正确的送达和接收
*/
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	// err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}
