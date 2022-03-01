# 设置目标OS环境变量，编译代码，并移动到虚拟机共享目录
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go
mv ./main /Users/liuzhi/VagrantFile/centos