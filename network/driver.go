/*
@Time :    2022/3/1 21:52
@Author :  liuzhi
@File :    NetDriver
@Software: GoLand
*/

package network

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
