package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type ContainerInfo struct {
	Pid        string `json:"pid"`
	Id         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	CreateTime string `json:"createTime"`
	Status     string `json:"status"`
}

var (
	Running             string = "running"
	Stop                string = "stop"
	Exit                string = "exit"
	DefaultInfoLocation string = "/var/run/mydocker/%s/"
	ConfigName          string = "config.json"
	LogName             string = "container.log"
)

func NewParentProcess(tty bool, volume string, containerName string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}
	log.Info("NewParentProcess")
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		logDir := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(logDir, 0622); err != nil {
			log.Errorf("mkdir logDir:%s error %v", logDir, err)
			return nil, nil
		}
		fileName := logDir + "/" + LogName
		file, err := os.Create(fileName)
		if err != nil {
			log.Errorf("create log file:%s error %v", fileName, err)
			return nil, nil
		}
		file.Write()
		cmd.Stdout = file
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	mntURL := "/root/overlayFS/mnt"
	rootURL := "/root/overlayFS/"
	NewWorkSpace(rootURL, mntURL, volume)
	// setUpMount()的GetWd获取
	cmd.Dir = mntURL
	// cmd.Dir = "/root/busybox"
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

func NewWorkSpace(rootURL string, mntURL string, volume string) {
	CreatReadOnlyLayer(rootURL)
	CreatWriteLayer(rootURL)
	CreatMountPoint(rootURL, mntURL, volume)

}

func CreatReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	// exist, err := PathExists(busyboxURL)
	_, err := os.Stat(busyboxURL)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}

	if os.IsNotExist(err) {
		fmt.Println(busyboxURL, " isn't exist.")
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("mkdir busyboxURL %s error. %v", busyboxURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("untar busyboxTarURL %s error. %v", busyboxTarURL, err)
		}
	}
}

func CreatWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.Mkdir(writeURL, 0777); err != nil {
		log.Errorf("mkdir writeURL %s error. %v", writeURL, err)
	}
	workURL := rootURL + "work/"
	if err := os.Mkdir(workURL, 0777); err != nil {
		log.Errorf("mkdir workURL %s error. %v", workURL, err)
	}
}

func CreatMountPoint(rootURL string, mntURL string, volume string) {
	// fmt.Println("创建mnt目录:", mntURL)
	if err := os.Mkdir(mntURL, 0777); err != nil {
		log.Errorf("mkdir mntURL %s error. %v", mntURL, err)
	}

	// dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	// cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	dirs := "lowerdir=" + rootURL + "busybox" + ",upperdir=" + rootURL + "writeLayer" + ",workdir=" + rootURL + "work"
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mntURL)
	// fmt.Println("dirs:", dirs)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount %v", err)
	}

	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		if len(volumeURLs) == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(rootURL, mntURL, volumeURLs)
			log.Infof("create volume mountpoint:", strings.Join(volumeURLs, " "))
		} else {
			log.Infof("Volume parameter input is not correct")
		}
	}
}

func MountVolume(rootURL string, mntURL string, volumeURLs []string) {
	parentURL := volumeURLs[0]
	if err := os.Mkdir(parentURL, 0777); err != nil {
		log.Errorf("mkdir parentURL %s error. %v", parentURL, err)
	}
	containerURL := volumeURLs[1]
	containerVolumeURL := mntURL + containerURL
	fmt.Println("parentURL:", parentURL)
	fmt.Println("containerVolumeURL:", containerVolumeURL)
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
		log.Errorf("mkdir containerVolumeURL %s error. %v", containerVolumeURL, err)
	}
	// 这里不确定是否是正确的写法，但是可以满足要求
	// dirs := "lowerdir=" + parentURL + ",upperdir=" + rootURL + "writeLayer" + ",workdir=" + rootURL + "work"
	// dirs := "lowerdir=" + parentURL + ",upperdir=" + parentURL + ",workdir=" + rootURL + "work"
	// cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, containerVolumeURL)
	cmd := exec.Command("mount", "--bind", parentURL, containerVolumeURL)
	// fmt.Println("volume dirs:", dirs)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount volume %v", err)
	}
}

func DeleteWorkSpace(rootURL string, mntURL string, volume string) {
	DeleteMountPoint(rootURL, mntURL, volume)
	DeleteWriteLayer(rootURL)
}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("remove writeURL %s error. %v", writeURL, err)
	}
	workURL := rootURL + "work/"
	if err := os.RemoveAll(workURL); err != nil {
		log.Errorf("remove workURL %s error. %v", workURL, err)
	}
}

func DeleteMountPoint(rootURL string, mntURL string, volume string) {
	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		if len(volumeURLs) == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			cmd := exec.Command("umount", mntURL+volumeURLs[1])
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Errorf("umount volume failed %v", err)
			}
			log.Infof("Delete Volume Mount Point: ", strings.Join(volumeURLs, " "))
		}
	}

	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount %v", err)
	}

	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("remove mntURL %s error. %v", mntURL, err)
	}

}
