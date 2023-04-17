package main

import (
	"encoding/json"
	"example/mydocker/cgroups"
	"example/mydocker/cgroups/subsystems"
	"example/mydocker/container"
	"example/mydocker/network"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func Run(tty bool, command []string, res *subsystems.ResourceConfig, volume string, containerName string, imageName string, environment []string, nw string, portMapping []string) {
	containerID := randStringBytes(10)
	if containerName == "" {
		log.Info("name is empty, use id")
		containerName = containerID
	}
	parent, writePipe := container.NewParentProcess(tty, volume, containerName, imageName, environment)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Fatal(err)
	}

	oneCommand := strings.Join(command, " ")
	containerInfo, err := recordContainerInfo(parent.Process.Pid, containerID, oneCommand, containerName, volume, portMapping)
	if containerInfo == nil || err != nil {
		log.Errorf("record container info error %v", err)
		return
	}
	if nw != "" {
		network.Init()
		if err := network.Connect(nw,containerInfo); err != nil {
			log.Errorf("connect network failed: %v",err)
			return
		}
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
		deleteContainerInfo(containerInfo.Name)
		// run()才是程序的main函数，所以要想确保在程序执行的最后销毁东西，写在这里比较好
		container.DeleteWorkSpace(volume, containerInfo.Name)
	}else {
		log.Debug("-d模式,容器pid: ",parent.Process.Pid)
		// 判断容器进程是否存活,用于detach失败的情况
		if isProcessDefunct(parent.Process.Pid) {
			log.Debug("detach失败，容器进程挂掉，删除容器相关信息")
			StopContainer(containerName)
			RemoveContainer(containerName)
		}
	}
	os.Exit(-1)
}

func isProcessDefunct(pid int) bool {
    out, err := exec.Command("ps", "-p", strconv.Itoa(pid)).Output()
    if err != nil {
        // 执行命令出错，说明进程不存在或者没有权限
        return false
    }
	log.Debug(string(out))
	log.Debug(err)
    // 如果输出中包含defunct，则说明detach失败，进程无效
    return strings.Contains(string(out), "defunct")
}

func sendInitCommand(oneCommand string, writePipe *os.File) {
	log.Infof("command all is %s", oneCommand)
	// time.Sleep(3 * time.Second)
	writePipe.WriteString(oneCommand)
	// fmt.Println("结束writePipe")
	writePipe.Close()
}

func recordContainerInfo(containerPID int, containerID string, oneCommand string, containerName string, volume string,portMapping []string) (*container.ContainerInfo, error) {
	createTime := time.Now().Format("2006-01-02 15:04:05")
	containerInfo := &container.ContainerInfo{
		Id:         containerID,
		Pid:        strconv.Itoa(containerPID),
		Name:       containerName,
		Command:    oneCommand,
		CreateTime: createTime,
		Status:     container.Running,
		Volume:     volume,
		PortMapping: portMapping,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return nil, err
	}
	jsonStr := string(jsonBytes)

	// 数据已准备好，开始创建目录
	configPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(configPath, 0622); err != nil {
		log.Errorf("mkdir configPath:%s error %v", configPath, err)
		return nil, err
	}
	fileName := configPath + "/" + container.ConfigName
	file, err := os.Create(fileName)
	if err != nil {
		log.Errorf("create config file:%s error %v", fileName, err)
		return nil, err
	}
	defer file.Close()

	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("write config file error %v", err)
		return nil, err
	}

	return containerInfo, err
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
