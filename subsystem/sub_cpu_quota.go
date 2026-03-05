package subsystem

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

type CpuQuotaSubsystem struct{}

func (s *CpuQuotaSubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	if res.CpuQuota == "" {
		return nil
	}
	dir, err := GetCpuCgroupPath(cgroupPath, true)
	if err != nil {
		return err
	}
	limitFile := "cpu.max"

	if err := os.WriteFile(path.Join(dir, limitFile), []byte(res.CpuQuota), 0644); err != nil {
		return fmt.Errorf("set cpu quota error: %w", err)
	}

	return nil
}

func (s *CpuQuotaSubsystem) Apply(cgroupPath string, pid int) error {
	dir, err := GetCpuCgroupPath(cgroupPath, false)
	if err != nil {
		return fmt.Errorf("get cpu cgroup path error: %w", err)
	}
	pidStr := strconv.Itoa(pid)
	if err := os.WriteFile(path.Join(dir, "cgroup.procs"), []byte(pidStr), 0644); err != nil {
		return fmt.Errorf("write pid to cgroup error: %w", err)
	}
	return nil
}

func (s *CpuQuotaSubsystem) Remove(cgroupPath string) error {
	dir, err := GetCpuCgroupPath(cgroupPath, false)
	if err != nil {
		// 目录不存在直接忽略
		return nil
	}
	// cgroup v2：必须先写 cgroup.kill=1，内核才会把该 cgroup 下所有进程迁走，
	// 之后才能用 os.Remove 删除空目录。直接 RemoveAll 会报 "operation not permitted"。
	_ = os.WriteFile(path.Join(dir, "cgroup.kill"), []byte("1"), 0644)
	return os.Remove(dir)
}

func (s *CpuQuotaSubsystem) Name() string {
	return "cpu"
}

// 只实现 cgroup v2
func GetCpuCgroupPath(cgroupPath string, autoCreate bool) (dir string, err error) {
	root := GetCgroupV2Root()
	dir = path.Join(root, cgroupPath)
	if autoCreate {
		_ = os.WriteFile(cgroupV2SubtreeCtrl, []byte("+cpu"), 0644)
		if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
			return "", fmt.Errorf("create cgroup v2 dir %s: %w", cgroupPath, err)
		}
	}
	return dir, nil
}
