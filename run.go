package main

import (
	"fmt"
	"my-container/container"
	"os"
)

func Run(tty bool, command string) {
	parent := container.NewParentProcess(tty, command)
	//err := parent.Start()
	//if err != nil {
	//	log.Error(err)
	//}
	err := parent.Run()
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	//os.Exit(-1)
}
