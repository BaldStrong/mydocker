package main

import (
	_ "example/chap3/nsenter"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const ENV_EXEC_PID = "mydocker_pid"
const ENV_EXEC_CMD = "mydocker_cmd"

func ExecContainer(containerName string, commandArray []string) {
	containerInfo, err := getContainerInfo(containerName)
	pid := containerInfo.Pid
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
