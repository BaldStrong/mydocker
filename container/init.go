package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 {
		return fmt.Errorf("run container get user command error, cmdArray is nil")
	}

	//需要手动将proc挂载到该进程下
	// syscall.Mount("", "/", "", syscall.MS_PRIVATE | syscall.MS_REC, "")
	// defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	// syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	// setUpMount()

	// 读到的第一个参数作为可执行文件的路径，进入容器后执行的第一个程序
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		return fmt.Errorf("exec loop path error %v", err)
	}
	log.Infof("Find path %s", path)
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		log.Errorf(err.Error())
	}
	return nil
}

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	// fmt.Println("开始ReadAll")
	msg, err := ioutil.ReadAll(pipe)
	// fmt.Println("结束ReadAll")
	if err != nil {
		log.Errorf("init read pipe error %v", err)
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Get current location error %v", err)
		return
	}
	log.Infof("Current location is %s", pwd)
	pivotRoot(pwd)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")

}

func pivotRoot(rootPath string) error {
	// 为了使当前rootPath所在的rootfs和接下来要切换到的rootfs不在同一个rootfs里面，
	// 这里需要把rootPath重新挂载一次，利用bind mount的方法：把相同内容换一个挂载点(rootfs)的挂载方法
	if err := syscall.Mount(rootPath, rootPath, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itself error: %v", err)
	}

	// 创建/rootPath/.pivot_root目录
	pivotDir := filepath.Join(rootPath, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	//切换根目录挂载点到rootPath，旧的old_root挂载到pivotDir上
	if err := syscall.PivotRoot(rootPath, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}
	// 切换完成后，切换当前目录为根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir %v", err)
	}
	// 卸载掉原来的rootfs
	pivotDir = filepath.Join("/", ".pivot_root")
	// MNT_DETACH是umount的一个参数，表示延迟卸载，立即断开文件系统与挂载点的连接，在挂载点空闲时才真正卸载
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("umount pivot_root dir %v", err)
	}
	// 删除pivotDir目录
	return os.Remove(pivotDir)
}
