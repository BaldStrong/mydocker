package main

import (
	"example/mydocker/container"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)


func RemoveContainer(containerName string) {
	containerInfo, err := getContainerInfo(containerName)
	if err != nil {
		log.Errorf("getContainerInfo:%s error %v", containerName, err)
		return
	}
	if containerInfo.Status != container.Stop {
		log.Errorf("cann't remove running container")
		return
	}
	configDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(configDir); err != nil {
		log.Errorf("remove configDir %s fail %v",configDir, err)
		return
	}
}