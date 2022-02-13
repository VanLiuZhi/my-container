package container

import (
	log "github.com/sirupsen/logrus"
	"os"
	"syscall"
)

func RunContainerInitProcess(cmd string, args []string) error {
	log.Infof("进入RunContainerInitProcess, command %s", cmd)
	// Systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示
	// 声明你要这个新的mount namespace独立。
	// 具体细节参考namespace关于mount的描述
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		return err
	}
	argv := []string{cmd}
	if err := syscall.Exec(cmd, argv, os.Environ()); err != nil {
		log.Error(err.Error())
	}
	return nil
}
