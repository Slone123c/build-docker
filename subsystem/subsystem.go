package subsystem

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

type ResourceConfig struct {
	MemoryLimit string
	CpuShare    string
	CpuSet      string
}

type Subsystem interface {
	// 返回 subsystem 的名称
	Name() string
	// 设置某个cgroup 的资源限制
	Set(path string, res *ResourceConfig) error
	// 将进程添加到cgroup中
	Apply(path string, pid int) error
	// 删除某个cgroup
	Remove(path string) error
}

var (
	SubsystemsIns = []Subsystem{
		// &CpusetSubsystem{},
		&MemorySubsystem{},
		// &CpuSubsystem{},
	}
)

func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		fields := strings.Split(text, " ")
		for _, opt := range fields {
			if opt == subsystem {
				return fields[4]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return ""
}

func GetCgroupPath(subsytem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsytem)

	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err != nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err != nil {
				return "", fmt.Errorf("create cgroup %s error: %v", cgroupPath, err)
			}
		} else {
			return "", fmt.Errorf("cgroup %s already exists", cgroupPath)
		}
	}
	return path.Join(cgroupRoot, cgroupPath), nil

}
