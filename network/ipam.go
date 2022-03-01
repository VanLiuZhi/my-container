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
	// 子网信息，通过读取文件并转换成map
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
		// 判断一下文件是否存在
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
	// 打开文件，不存在则创建
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
	_ = ipam.load()

	_, ipNet, _ := net.ParseCIDR(subnet.String())

	size, bits := ipNet.Mask.Size()

	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(bits-size))
	}

	for c := range (*ipam.Subnets)[subnet.String()] {
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
			ipAlloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipAlloc)
			// 赋值返回值ip
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}
	_ = ipam.dump()
	return
}

// Release 释放分配的IP
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}

	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1

	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
	ipAlloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipAlloc)

	_ = ipam.dump()
	return nil
}
