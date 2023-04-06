package main

import (
	"encoding/json"
	"example/mydocker/container"
	"fmt"
	"io/ioutil"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
)


func StopContainer(containerName string) {
	containerInfo, err := getContainerInfo(containerName)
	if err != nil {
		log.Errorf("getContainerInfo:%s error %v", containerName, err)
		return
	}
	if containerInfo.Status != container.Running {
		log.Errorf("container is no longer running")
		return
	}
	pid := containerInfo.Pid
	pidInt,err := strconv.Atoi(pid)
	if err != nil {
		log.Errorf("conver pid from string to int error %v", err)
		return
	}
	if err := syscall.Kill(pidInt,syscall.SIGTERM); err != nil {
		log.Errorf("kill container: %s error %v", containerName, err)
		return
	}
	containerInfo.Status = container.Stop
	containerInfo.Pid = ""
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("record container info error %v", err) 
		return
	}
	
	configPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configPath = configPath + container.ConfigName
	if err := ioutil.WriteFile(configPath, jsonBytes, 0622); err != nil {
		log.Errorf("update config fail %v", err)
		return
	}
}