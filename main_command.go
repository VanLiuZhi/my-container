package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"my-container/container"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
	},
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missage container command")
		}
		cmd := ctx.Args().Get(0)
		tty := ctx.Bool("ti")
		// 准备启动容器
		log.Info("参数校验ok，准备运行container")
		Run(tty, cmd)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container",
	Action: func(ctx *cli.Context) error {
		log.Info("init come on")
		cmd := ctx.Args().Get(0)
		log.Infof("command %s", cmd)
		err := container.RunContainerInitProcess(cmd, nil)
		return err
	},
}
