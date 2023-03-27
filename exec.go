package main

import (
	"encoding/json"
	"example/chap3/container"
	_ "example/chap3/nsenter"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const ENV_EXEC_PID = "mydocker_pid"
const ENV_EXEC_CMD = "mydocker_cmd"

func ExecContainer(containerName string, commandArray []string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("getContainerPidByName:%s error %v", containerName, err)
		return
	}
	oneCommand := strings.Join(commandArray, " ")
	log.Infof("pid:%s cmd:%s", pid, oneCommand)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, oneCommand)

	if err := cmd.Run(); err != nil {
		log.Errorf("Exec container %s error %v", containerName, err)
	}
}

func getContainerPidByName(containerName string) (string, error) {
	configPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configPath = configPath + container.ConfigName
	// configPath = path.Join(configPath, container.ConfigName)
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Errorf("getContainerPidByName: read configPath:%s error %v", configPath, err)
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("getContainerPidByName: Unmarshal error %v", err)
		return "", err
	}
	return containerInfo.Pid, err
}
