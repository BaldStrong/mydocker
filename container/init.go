package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 {
		return fmt.Errorf("run container get user command error, cmdArray is nil")
	}

	// 读到的第一个参数作为可执行文件的路径，进入容器后执行的第一个程序
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		return fmt.Errorf("exec loop path error %v", err)
	}
	log.Infof("Find path %s", path)
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		log.Errorf(err.Error())
	}
	return nil
}

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	fmt.Println("开始ReadAll")
	msg, err := ioutil.ReadAll(pipe)
	fmt.Println("结束ReadAll")
	if err != nil {
		log.Errorf("init read pipe error %v", err)
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}
