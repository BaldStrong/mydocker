package main

import (
	"encoding/json"
	"example/mydocker/container"
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
)

func ListContainers() {
	configPath := fmt.Sprintf(container.DefaultInfoLocation, "")
	// 去掉“/var/run/mydocker//”最后的/，感觉这种写法不太自然
	configPath = configPath[:len(configPath)-1]
	files, err := ioutil.ReadDir(configPath)
	if err != nil {
		log.Errorf("read configPath error %v", err)
		return
	}

	var containerInfos []*container.ContainerInfo
	for _, file := range files {
		tmpInfo, err := getContainerInfo(file.Name())
		if err != nil {
			log.Errorf("getContainerInfo error %v", err)
			continue
		}
		containerInfos = append(containerInfos, tmpInfo)
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containerInfos {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreateTime)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("flush tabwriter error %v", err)
		return
	}
}

func getContainerInfo(containerName string) (*container.ContainerInfo, error) {
	configPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configPath = configPath + container.ConfigName
	// configPath = path.Join(configPath, container.ConfigName)
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Errorf("read configPath:%s error %v", configPath, err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("Unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, err
}
