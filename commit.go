package main

import (
	"example/mydocker/container"
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func CommitContainer(containerName string, imageName string) {
	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	imageTarURL := container.RootUrl + "/" + imageName + ".tar"
	// 此处必须要使用-C将tar目录切换到mntURL，如果直接指定mntURL，会将mntURL也带入，
	// 导致压缩文件包含/root/overlayFS/mnt才到镜像文件所在目录
	if a, err := exec.Command("tar", "-czf", imageTarURL, "-C", mntURL, ".").CombinedOutput(); err != nil {
		fmt.Println(a)
		log.Errorf("tar folder %s error %v", imageTarURL, err)
	}
}
