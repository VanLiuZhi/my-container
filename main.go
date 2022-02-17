/*
@Time :    2022/2/17 23:04
@Author :  liuzhi
@File :    main
@Software: GoLand
*/

package main

import (
	"github.com/urfave/cli"
	"my-container/wheel"
	"os"
)
import log "github.com/sirupsen/logrus"

const usage = `my docker(一个类docker的demo)`

func main() {
	app := cli.NewApp()
	app.Name = "my-docker"
	app.Usage = usage

	app.Commands = []cli.Command{
		wheel.RunCommand,
		wheel.InitCommand,
	}

	app.Before = func(context *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("运行失败，即将打印错误栈日志")
		log.Fatal(err)
	}
}
