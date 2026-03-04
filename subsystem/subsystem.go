package subsystem

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	cgroupV2Root        = "/sys/fs/cgroup"
	cgroupV2Controllers = "/sys/fs/cgroup/cgroup.controllers"
	cgroupV2SubtreeCtrl = "/sys/fs/cgroup/cgroup.subtree_control"
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

// mountinfo 中挂载点路径在第四列（index 4），subsystem 在最后一列且为逗号分隔，如 "rw,memory"
const mountPointIndex = 4

// IsCgroupV2 判断当前系统是否使用 cgroup v2 统一层级（如 Ubuntu 22.04）。
// 若 /sys/fs/cgroup/cgroup.controllers 存在且包含 memory，则为 v2。
func IsCgroupV2() bool {
	data, err := os.ReadFile(cgroupV2Controllers)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "memory")
}

// GetCgroupV2Root 返回 cgroup v2 根路径；若非 v2 则返回空字符串。
func GetCgroupV2Root() string {
	if IsCgroupV2() {
		return cgroupV2Root
	}
	return ""
}

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
		if len(fields) <= mountPointIndex {
			continue
		}
		// 最后一列是挂载选项，如 "rw,memory"，需按逗号拆开再匹配 subsystem
		opts := strings.Split(fields[len(fields)-1], ",")
		for _, opt := range opts {
			if opt == subsystem {
				return fields[mountPointIndex]
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

// GetMemoryCgroupPath 返回用于内存限制的 cgroup 目录路径。
// 若为 cgroup v2（如 Ubuntu 22.04），使用统一根 + cgroupPath，并确保父层级启用了 memory 控制器。
func GetMemoryCgroupPath(cgroupPath string, autoCreate bool) (dir string, isV2 bool, err error) {
	if IsCgroupV2() {
		root := GetCgroupV2Root()
		dir = path.Join(root, cgroupPath)
		if autoCreate {
			// v2 下需在根 cgroup 的 subtree_control 中启用 memory，子 cgroup 才有 memory.* 接口
			_ = os.WriteFile(cgroupV2SubtreeCtrl, []byte("+memory"), 0644)
			if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
				return "", true, fmt.Errorf("create cgroup v2 dir %s: %w", cgroupPath, err)
			}
		}
		return dir, true, nil
	}
	// v1: 按 memory subsystem 挂载点查找
	dir, err = GetCgroupPath("memory", cgroupPath, autoCreate)
	return dir, false, err
}
