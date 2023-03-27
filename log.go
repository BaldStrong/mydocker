package main

import (
	"example/chap3/container"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

func LogContainer(containerName string) {
	logDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logPath := logDir + container.LogName
	file, err := os.Open(logPath)
	if err != nil {
		log.Errorf("read logPath:%s error %v", logPath, err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("read logfile error %v", err)
	}
	fmt.Fprint(os.Stdout, string(content))
}
