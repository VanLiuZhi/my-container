/*
@Time :    2022/2/17 23:17
@Author :  liuzhi
@File :    network
@Software: GoLand
*/

package network

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"my-container/container"
	"my-container/network/wheel"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/my-container/network/"
	drivers            = map[string]NetDriver{}
	networks           = map[string]*Network{}
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

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// 通过 ParseCIDR 转换网段字符串
	// For example, ParseCIDR("192.0.2.1/24") returns the IP address
	// 192.0.2.1 and the network 192.0.2.0/24.
	ip, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		panic("subnet 参数非法")
	}
	log.Debug("网络转换：ip -> ", ip)
	log.Debug("网络转换：cidr -> ", cidr)
	// 通过IPAM组件分配IP，获取网段中的第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocator(cidr)
	if err != nil {
		return err
	}
	// cidr 由 { IP，Mask } 两个成员变量组成
	cidr.IP = gatewayIp
	network, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	// 保存网络配置，方便后续查询网络端点信息，数据保存在文件中
	return network.dump(defaultNetworkPath)
}

// Connect 连接网络，相当于把veth设备挂到Linux Bridge网桥上
func Connect(networkName string, containerInfo *container.ContainerInfo) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("network %s not found", networkName)
	}
	// 分配容器IP地址
	ip, err := ipAllocator.Allocator(network.IpRange)
	if err != nil {
		log.Error("调用 Connect, ip 分配失败")
		return err
	}
	// 创建网络端点
	endpoint := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", containerInfo.Id, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: containerInfo.PortMapping,
	}
	// 调用网络驱动 挂载和配置网络端点
	if err = drivers[network.Driver].Connect(network, endpoint); err != nil {
		return err
	}
	// 在容器的 namespace 中配置网络设备的IP地址
	if err = wheel.ConfigEndpointIpAddressAndRoute(endpoint, containerInfo); err != nil {
		return err
	}
	// 配置容器到端口的主机端口的映射
	return wheel.ConfigPortMapping(endpoint, containerInfo)

}

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no Such Network: %s", networkName)
	}

	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("error Remove Network gateway ip: %s", err)
	}

	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("error Remove Network DriverError: %s", err)
	}

	return nw.remove(defaultNetworkPath)
}

func Init() error {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = nil

	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	_ = filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(nwPath, "/") {
			return nil
		}
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		if err := nw.load(nwPath); err != nil {
			log.Errorf("error load network: %s", err)
		}

		networks[nwName] = nw
		return nil
	})

	log.Infof("networks: %v", networks)

	return nil
}

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}
