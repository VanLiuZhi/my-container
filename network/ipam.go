/*
@Time :    2022/2/20 10:05
@Author :  liuzhi
@File :    ipam
@Software: GoLand
*/

package network

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/run/go-micro-demo-demo/network/ipam/subnet.json"

// IPAM 定义IPAM
type IPAM struct {
	// 子网分配文件存储路径
	SubnetAllocatorPath string
	// 子网信息，通过读取文件并转换成map，通过位图标识已分配的地址（key 是网段，value 是位图数组内容，也就是网络地址的字符串）
	// 比如 key -> value : 192.168.1.1/24 -> 0000000000000000000000001000...
	Subnets *map[string]string
}

// 实例化结构体 IPAM，默认实例
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) ExternalLoadByTest() (*IPAM, error) {
	_ = ipam.load()
	return ipam, nil
}

// 加载配置
func (ipam *IPAM) load() error {
	// 通过 os.Stat 返回一个描述文件信息的 FileInfo 对象
	// 类似通过 ls -al 这样能查看文件的信息将其包装在FileInfo，比如判断这个文件是是不是一个 dir
	if stat, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		// 判断一下文件是否存在（第一次文件肯定是不存在的，无需加载历史数据，直接返回）
		log.Debug("os.Stat info -> ", stat)
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	// 打开文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer func(subnetConfigFile *os.File) {
		err := subnetConfigFile.Close()
		if err != nil {
			log.Debug("file close error")
		}
	}(subnetConfigFile)

	if err != nil {
		return err
	}
	// 定义读取文件的数据变量
	subnetConfigJson := make([]byte, 3000)
	// Read 方法把文件对象装在到传递的[]byte中，返回文件长度(reads up to len(b) bytes from the File.)
	read, err := subnetConfigFile.Read(subnetConfigJson)
	if err != nil {
		return err
	}
	// 将json数据转换成map赋值给结构体成员 Subnets(先切片，只保留原始数据，因为我们分配的数组比较长，只切原始数据)
	err = json.Unmarshal(subnetConfigJson[:read], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump Allocator info, %v", err)
		return err
	}
	return nil
}

// 转储配置
func (ipam *IPAM) dump() error {
	// 解析配置文件路径，目录不存在就创建一个新的
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}
	// 打开文件，不存在则创建 (标记位：O_TRUNC 如果存在则清空)
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer func(subnetConfigFile *os.File) {
		err := subnetConfigFile.Close()
		if err != nil {
			log.Debug("file close error")
		}
	}(subnetConfigFile)
	if err != nil {
		return err
	}
	// 读取对象信息转换成json
	imapConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err

	}
	// 序列化成json写入文件
	_, err = subnetConfigFile.Write(imapConfigJson)
	if err != nil {
		return err
	}
	return nil
}

// Allocator 分配ip
func (ipam *IPAM) Allocator(subnet *net.IPNet) (ip net.IP, err error) {
	// 定义存放网段信息的map(在实例化结构体的时候没初始化对象，这里处理)
	ipam.Subnets = &map[string]string{}

	// 从文件中加载已分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("ipam load 网络分配信息失败")
	}

	_, ipNet, _ := net.ParseCIDR(subnet.String())

	// ip：192.168.1.100/24 网络地址：192.168.1.0/24 子网掩码：255.255.255.0
	// 255.255.255.0 调用此方法返回 24 32，网络前缀占24位，主机位 32 - 8
	ones, bits := ipNet.Mask.Size()

	// 如果之前没有分配过这个网段，则初始化网段配置（看有多少主机地址位数，初始化多少个0）
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		// (32 - 8) ^ 2, 等效 1 << 24
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(bits-ones))
	}

	// 遍历位图字符串（可以考虑用byte数组）
	for c := range (*ipam.Subnets)[subnet.String()] {
		// 找到为 "0" 项，也就是还未配置的
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 字符串不能修改，把字符串转换成 byte 数组，然后根据当前的 index，也就是c，修改为 "1"
			ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
			ipAlloc[c] = '1'
			// 重新转回字符串
			(*ipam.Subnets)[subnet.String()] = string(ipAlloc)
			// 赋值返回值ip，先从 subnet.IP 取出网络地址
			// ip：192.168.1.100/24 则网络地址：192.168.1.0/24，那么从0开始计算偏移量即可
			ip = subnet.IP
			/*
				ipv4, IP 的 byte 数组是 4位长度
				比如网段是 172.16.0.0/12，数组序号是 65555. 那么在172.16.0.0
				上依次加[uint8(65555 >> 24)、uint8(65555 >> 16)、uint8(65555 >> 8)、uint8(65555 >> 0)，即[0, 1, 0, 19)
				那么获得的 IP 就是 172.17.0.19
			*/
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			// 由于网络地址不能分配，是从1开始分配的，所以加1
			ip[3] += 1
			break
		}
	}
	_ = ipam.dump()
	return
}

// Release 释放分配的IP，参数是网段地址和需要释放的IP
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())
	// 加载配置
	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}
	// 初始化索引，待会计算后赋值，表示ip地址在网段中的索引位置
	c := 0
	// 转换成4字节表示
	releaseIP := ipaddr.To4()
	// 由于 IP 是从1开始分配的，所以转换成索引应减 1
	releaseIP[3] -= 1
	/*
		与分配 IP 相反，释放 IP 获得索引的方式是 IP 地址的每一位相减之后分别左移将对应的数值加到索引上
	*/
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}
	// 将分配的位图数组中索引位置的值置为0
	ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
	ipAlloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipAlloc)

	_ = ipam.dump()
	return nil
}
