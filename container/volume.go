package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// 为每个容器创建一个workspace
func NewWorkSpace(volume string, containerName string, imageName string) {
	CreatReadOnlyLayer(imageName)
	CreatWriteLayer(containerName)
	CreatMountPoint(containerName, imageName, volume)
}

func CreatReadOnlyLayer(imageName string) {
	imageURL := RootUrl + "/" + imageName + "/"
	imageTarURL := RootUrl + "/" + imageName + ".tar"
	// exist, err := PathExists(busyboxURL)
	_, err := os.Stat(imageURL)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", imageURL, err)
	}

	if os.IsNotExist(err) {
		fmt.Println(imageURL, " isn't exist.")
		if err := os.MkdirAll(imageURL, 0777); err != nil {
			log.Errorf("mkdir imageURL %s error. %v", imageURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", imageTarURL, "-C", imageURL).CombinedOutput(); err != nil {
			log.Errorf("untar imageTarURL %s error. %v", imageTarURL, err)
		}
	}
}

func CreatWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl,containerName)
	if err := os.MkdirAll(writeURL, 0777); err != nil {
		log.Errorf("mkdir writeURL %s error. %v", writeURL, err)
	}
	workURL := fmt.Sprintf(WorkLayerUrl,containerName)
	if err := os.MkdirAll(workURL, 0777); err != nil {
		log.Errorf("mkdir workURL %s error. %v", workURL, err)
	}
}

func CreatMountPoint(containerName string, imageName string, volume string) {
	mntURL := fmt.Sprintf(MntUrl,containerName)
	// fmt.Println("创建mnt目录:", mntURL)
	if err := os.MkdirAll(mntURL, 0777); err != nil {
		log.Errorf("mkdir mntURL %s error. %v", mntURL, err)
	}
	imageURL := RootUrl + "/" + imageName + "/"
	writeURL := fmt.Sprintf(WriteLayerUrl,containerName)
	workURL := fmt.Sprintf(WorkLayerUrl,containerName)
	dirs := "lowerdir=" + imageURL + ",upperdir=" + writeURL + ",workdir=" + workURL
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
			MountVolume(volumeURLs,containerName)
			log.Infof("create volume mountpoint:", strings.Join(volumeURLs, " "))
		} else {
			log.Infof("Volume parameter input is not correct")
		}
	}
}

func MountVolume(volumeURLs []string,containerName string) {
	parentURL := volumeURLs[0]
	if err := os.MkdirAll(parentURL, 0777); err != nil {
		log.Errorf("mkdir parentURL %s error. %v", parentURL, err)
	}
	containerURL := volumeURLs[1]
	mntURL := fmt.Sprintf(MntUrl,containerName)
	containerVolumeURL := mntURL+ "/" + containerURL
	fmt.Println("parentURL:", parentURL)
	fmt.Println("containerVolumeURL:", containerVolumeURL)
	if err := os.MkdirAll(containerVolumeURL, 0777); err != nil {
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

func DeleteWorkSpace(volume string,containerName string) {
	DeleteMountPoint(containerName, volume)
	DeleteWriteLayer(containerName)
}

func DeleteWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl,containerName)
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("remove writeURL %s error. %v", writeURL, err)
	}
	workURL := fmt.Sprintf(WorkLayerUrl,containerName)
	if err := os.RemoveAll(workURL); err != nil {
		log.Errorf("remove workURL %s error. %v", workURL, err)
	}
}

func DeleteMountPoint(containerName string, volume string) {
	mntURL := fmt.Sprintf(MntUrl,containerName)
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
