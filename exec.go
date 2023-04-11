package main

import (
	_ "example/mydocker/nsenter"
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
	containerEnv := getEnvsByPid(pid);
	cmd.Env = append(os.Environ(), containerEnv...)

	if err := cmd.Run(); err != nil {
		log.Errorf("Exec container %s error %v", containerName, err)
	}
}


func getEnvsByPid(pid string) []string {
	environ := fmt.Sprintf("/proc/%s/environ",pid)
	context,err := ioutil.ReadFile(environ)
	if err != nil {
		log.Errorf("read /proc/%s/environ failed: %v",environ,err)
		return nil
	}
	envs := strings.Split(string(context), "\u0000")
	return envs
}