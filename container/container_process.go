package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
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
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	mntURL := "/root/overlayFS/mnt"
	rootURL := "/root/overlayFS/"
	NewWorkSpace(rootURL, mntURL)
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

func NewWorkSpace(rootURL string, mntURL string) {
	CreatReadOnlyLayer(rootURL)
	CreatWriteLayer(rootURL)
	CreatMountPoint(rootURL, mntURL)
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
		fmt.Println(busyboxURL, " 不存在")
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

func CreatMountPoint(rootURL string, mntURL string) {
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
}

func DeleteWorkSpace(rootURL string, mntURL string) {
	DeleteWriteLayer(rootURL)
	DeleteMountPoint(rootURL, mntURL)
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
func DeleteMountPoint(rootURL string, mntURL string) {
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
