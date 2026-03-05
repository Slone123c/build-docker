package subsystem

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type MemorySubsystem struct{}

// parseMemoryLimit 将 "10m"、"1g" 等转为字节数字符串。memory.limit_in_bytes 只接受纯数字（字节）。
func parseMemoryLimit(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	s = strings.ToLower(s)
	var mult int64 = 1
	if len(s) > 1 {
		switch s[len(s)-1] {
		case 'k':
			mult = 1024
			s = s[:len(s)-1]
		case 'm':
			mult = 1024 * 1024
			s = s[:len(s)-1]
		case 'g':
			mult = 1024 * 1024 * 1024
			s = s[:len(s)-1]
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid memory limit %q: %w", s, err)
	}
	if n < 0 {
		return "", fmt.Errorf("memory limit must be non-negative")
	}
	return strconv.FormatInt(n*mult, 10), nil
}

func (s *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	if res.MemoryLimit == "" {
		return nil
	}
	dir, isV2, err := GetMemoryCgroupPath(cgroupPath, true)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %w", cgroupPath, err)
	}
	limitBytes, err := parseMemoryLimit(res.MemoryLimit)
	if err != nil {
		return err
	}
	limitFile := "memory.limit_in_bytes"
	if isV2 {
		limitFile = "memory.max"
	}
	if err := os.WriteFile(path.Join(dir, limitFile), []byte(limitBytes), 0644); err != nil {
		return fmt.Errorf("set memory limit error: %v", err)
	}
	// cgroup v2 下 memory.max 只限制 RAM，不限制 swap。
	// 系统开启了 swap（/proc/swaps），进程超出 RAM 限制后会被 swap 到磁盘而不是被 OOM Kill。
	// 必须同时将 memory.swap.max 设为 0，禁止 swap 使用，memory.max 才真正有效。
	if isV2 {
		swapMaxPath := path.Join(dir, "memory.swap.max")
		if err := os.WriteFile(swapMaxPath, []byte("0"), 0644); err != nil {
			return fmt.Errorf("set memory swap limit error: %v", err)
		}
	}
	return nil
}

func (s *MemorySubsystem) Remove(cgroupPath string) error {
	dir, _, err := GetMemoryCgroupPath(cgroupPath, false)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
	return os.RemoveAll(dir)
}

func (s *MemorySubsystem) Apply(cgroupPath string, pid int) error {
	dir, isV2, err := GetMemoryCgroupPath(cgroupPath, true)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
	procsFile := "tasks"
	if isV2 {
		procsFile = "cgroup.procs"
	}
	pidStr := strconv.Itoa(pid) + "\n"
	if err := os.WriteFile(path.Join(dir, procsFile), []byte(pidStr), 0644); err != nil {
		return fmt.Errorf("add task to cgroup %s error: %v", cgroupPath, err)
	}
	return nil
}

func (s *MemorySubsystem) Name() string {
	return "memory"
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
