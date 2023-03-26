package main

import (
	"encoding/json"
	"example/chap3/cgroups"
	"example/chap3/cgroups/subsystems"
	"example/chap3/container"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func Run(tty bool, command []string, res *subsystems.ResourceConfig, volume string, containerName string) {
	parent, writePipe := container.NewParentProcess(tty, volume, containerName)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Fatal(err)
	}

	oneCommand := strings.Join(command, " ")
	containerName, err := recordContainerInfo(parent.Process.Pid, oneCommand, containerName)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return
	}
	// 执行闪退，发现是这里的问题，后面发现是flag里面的mem参数没有传进来导致的
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Remove()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(oneCommand, writePipe)
	// 只有当交互式时父进程会等待子进程结束
	if tty {
		parent.Wait()
		deleteContainerInfo(containerName)
	}
	// run()才是程序的main函数，所以要想确保在程序执行的最后销毁东西，写在这里比较好
	mntURL := "/root/overlayFS/mnt"
	rootURL := "/root/overlayFS/"
	container.DeleteWorkSpace(rootURL, mntURL, volume)
	os.Exit(-1)
}

func sendInitCommand(oneCommand string, writePipe *os.File) {
	log.Infof("command all is %s", oneCommand)
	// time.Sleep(3 * time.Second)
	writePipe.WriteString(oneCommand)
	// fmt.Println("结束writePipe")
	writePipe.Close()
}

func recordContainerInfo(containerPID int, oneCommand string, containerName string) (string, error) {
	id := randStringBytes(10)
	createTime := time.Now().Format("2006-01-02 15:04:05")
	if containerName == "" {
		log.Info("name is empty, use id")
		containerName = id
	}
	containerInfo := &container.ContainerInfo{
		Id:         id,
		Pid:        strconv.Itoa(containerPID),
		Name:       containerName,
		Command:    oneCommand,
		CreateTime: createTime,
		Status:     container.Running,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	// 数据已准备好，开始创建目录
	configPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(configPath, 0622); err != nil {
		log.Errorf("mkdir configPath:%s error %v", configPath, err)
		return "", err
	}
	fileName := configPath + "/" + container.ConfigName
	file, err := os.Create(fileName)
	if err != nil {
		log.Errorf("create config file:%s error %v", fileName, err)
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("write config file error %v", err)
		return "", err
	}

	return containerName, err
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	// rand.Seed(time.Now().UnixNano()) 在Go1.21中已废弃
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[random.Intn(len(letterBytes))]
	}
	return string(b)
}

func deleteContainerInfo(containerName string) {
	configPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(configPath); err != nil {
		log.Errorf("remove configPath:%s error %v", configPath, err)
	}
}
