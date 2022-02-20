/*
@Time :    2022/2/20 10:05
@Author :  liuzhi
@File :    ipam
@Software: GoLand
*/

package network

import (
	log "github.com/sirupsen/logrus"
	"net"
	"os"
)

const ipamDefaultAllocatorPath = "/var/run/go-test-demo/network/ipam/subnet.json"

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

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
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	// 读取文件的数据
	subnetConfigJson := make([]byte, 2000)
	read, err := subnetConfigFile.Read(subnetConfigJson)
	if err != nil {
		return err
	}

}

func (ipam *IPAM) Allocator(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}

}
