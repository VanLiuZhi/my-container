package main

import (
	log "github.com/sirupsen/logrus"
	"my-container/container"
	"os"
)

func Run(tty bool, command string) {
	parent := container.NewParentProcess(tty, command)
	if err := parent.Start(); err != nil {
		log.Error("返回配置好的command对象发生异常")
		log.Error(err)
	}
	err := parent.Wait()
	if err != nil {
		log.Error("执行Run命令失败, ERROR:", err)
	}
	os.Exit(1)
}
