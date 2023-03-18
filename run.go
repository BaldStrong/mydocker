package main

import (
	"example/chap3/cgroups"
	"example/chap3/cgroups/subsystems"
	"example/chap3/container"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Run(tty bool, command []string, res *subsystems.ResourceConfig) {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Fatal(err)
	}
	// 执行闪退，发现是这里的问题，后面发现是flag里面的mem参数没有传进来导致的
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Remove()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(command, writePipe)
	parent.Wait()
	// run()才是程序的main函数，所以要想确保在程序执行的最后销毁东西，写在这里比较好
	mntURL := "/root/overlayFS/mnt"
	rootURL := "/root/overlayFS/"
	container.DeleteWorkSpace(rootURL, mntURL)
	os.Exit(-1)
}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	oneCommand := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", oneCommand)
	// time.Sleep(3 * time.Second)
	writePipe.WriteString(oneCommand)
	// fmt.Println("结束writePipe")
	writePipe.Close()
}
