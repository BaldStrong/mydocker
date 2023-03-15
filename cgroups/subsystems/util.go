package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

// 找到对应子系统的cgroup挂载点
func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}

	defer f.Close()
	// 按行扫描f
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		// 空格分割
		fields := strings.Split(txt, " ")
		// 分割后数组后，取最后一个，查看其中是否包含子系统的名称
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			// cgroup V2不再对每个subsystem各自一个目录，而是只有一层
			if strings.Contains(opt, "memory") {
				// 找到返回路径，即挂载点
				// return path.Join(fields[4], subsystem)
				return fields[4]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return ""
}

// 获取目标cgroup挂载点，不存在就新建一个
func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsystem)
	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			// fmt.Println(cgroupPath, " 不存在，新建一个:", cgroupRoot, subsystem)
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {
			} else {
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		return path.Join(cgroupRoot, cgroupPath), nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}
