## 网络流程总览

1. create，先创建网桥
2. connect，把容器的网络栈通过veth-pair设备连接上网桥

## 网络SDK

- net

  golang 提供的网络操作模块

- github.com/vishvananda/netlink

  操作网络接口，路由表配置的库。提供了Linux下ip命令

## 创建网络

mydocker network create --subnet 192.168.0.0/24 --driver bridge testbridgenet

## 指定容器网络

mydocker run -ti -p 80 : 80 --net testbridgenet xxxx