package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type ContainerInfo struct {
	Pid        string `json:"pid"`
	Id         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	CreateTime string `json:"createTime"`
	Status     string `json:"status"`
	Volume     string `json:"volume"`
	PortMapping []string `json:"portmapping"`
}

var (
	Running             string = "running"
	Stop                string = "stopped"
	Exit                string = "exited"
	DefaultInfoLocation string = "/var/run/mydocker/container/%s/"
	ConfigName          string = "config.json"
	LogName             string = "container.log"
)

var (
	RootUrl       string = "../overlayFS"
	MntUrl        string = "../overlayFS/mnt/%s"
	WriteLayerUrl string = "../overlayFS/writeLayer/%s"
	WorkLayerUrl  string = "../overlayFS/work/%s"
)

func NewParentProcess(tty bool, volume string, containerName string, imageName string, environment []string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}
	log.Info("NewParentProcess")
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		logDir := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(logDir, 0622); err != nil {
			log.Errorf("mkdir logDir:%s error %v", logDir, err)
			return nil, nil
		}
		fileName := logDir + "/" + LogName
		file, err := os.Create(fileName)
		if err != nil {
			log.Errorf("create log file:%s error %v", fileName, err)
			return nil, nil
		}
		cmd.Stdout = file
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(cmd.Env, environment...)
	NewWorkSpace(volume, containerName, imageName)
	// setUpMount()的GetWd获取
	cmd.Dir = fmt.Sprintf(MntUrl, containerName)
	// cmd.Dir = "./busybox"
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
