/*
@Time :    2022/2/17 23:17
@Author :  liuzhi
@File :    network
@Software: GoLand
*/

package network

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"path"
)

var (
	defaultNetworkPath = "/var/run/my-container/network/"
	drivers            = map[string]NetDriver{}
)

// Network 定义网络基本模型
type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 网络地址端，比如 10.10.1.0/24
	Driver  string     // 网络驱动名
}

// Endpoint 网络端点
// Tips Golang json对象在定义的时候，要声明成员变量是大写才能导出，也就是才会被序列化，可以指定序列化后的名称，一般就是转回驼峰或重写名称
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

func (net *Network) dump(dumpPath string) error {
	// 判断目录是否存在，没有则创建
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}
	// 打开文件
	newPath := path.Join(dumpPath, net.Name)
	nwFile, err := os.OpenFile(newPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("error：%v", err)
		return err
	}
	defer func(nwFile *os.File) {
		err := nwFile.Close()
		if err != nil {
			log.Error("file close error")
		}
	}(nwFile)
	// 序列化
	nwJson, err := json.Marshal(net)
	if err != nil {
		log.Errorf("error：%v", err)
		return err
	}
	// 写入文件
	_, err = nwFile.Write(nwJson)
	if err != nil {
		log.Errorf("error：%v", err)
		return err
	}
	return nil
}

func (net *Network) remove(dumpPath string) error {
	// 调用 Stat 方法，查看文件信息(linux下，目录，文件，设备 都算 "文件")
	// err 不为 nil，方法异常
	if _, err := os.Stat(path.Join(dumpPath, net.Name)); err != nil {
		// 传递异常给 IsNotExist，不存在，则返回 nil，不用执行后续逻辑(说明文件没有，不用remove)
		if os.IsNotExist(err) {
			return nil
		} else {
			// 不是 IsNotExist 能处理的异常(ErrNotExist)，返回 err
			return err
		}
	} else {
		// 文件存在，删除
		return os.Remove(path.Join(dumpPath, net.Name))
	}
}

func (net *Network) load(dumpPath string) error {
	// 打开，读，反序列化
	netConfigFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer func(netConfigFile *os.File) {
		err := netConfigFile.Close()
		if err != nil {
			log.Error("file close error")
		}
	}(netConfigFile)
	netJson := make([]byte, 3000)
	read, err := netConfigFile.Read(netJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(netJson[:read], net)
	if err != nil {
		log.Errorf("error load netConfigFile info：%v", err)
		return err
	}
	return nil
}

func CreateNetwork(driver, subnet, name string) error {
	// 通过 ParseCIDR 转换网段字符串
	// For example, ParseCIDR("192.0.2.1/24") returns the IP address
	// 192.0.2.1 and the network 192.0.2.0/24.
	cidr, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		panic("subnet 参数非法")
	}
	log.Debug("网络转换：ip -> ", cidr)
	log.Debug("网络转换：ipNet -> ", ipNet)
	// 通过IPAM组件分配IP，获取网段中的第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocator(ipNet)
	if err != nil {
		return err
	}
	ipNet.IP = gatewayIp
	network, err := drivers[driver].Create(ipNet.String(), name)
	if err != nil {
		return err
	}
	return network.dump(defaultNetworkPath)
}
