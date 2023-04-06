package main

import (
	"example/mydocker/container"

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
	deleteContainerInfo(containerName)
}