package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func MountVolume(rootPath string, volume string) error {

	volumes := strings.Split(volume, ":")
	if len(volumes) != 2 {
		return fmt.Errorf("volume parameter format error, should be hostPath:containerPath")
	}

	hostPath := volumes[0]
	containerPath := volumes[1]

	// 重要： 拼凑出目标挂载点在当前视角下的绝对路径
	mntURL := filepath.Join(rootPath, "merged", containerPath)

	if err := os.MkdirAll(hostPath, 0777); err != nil {
		return fmt.Errorf("mkdir host path error: %v", err)
	}
	if err := os.MkdirAll(mntURL, 0777); err != nil {
		return fmt.Errorf("mkdir container path error: %v", err)
	}

	// 执行 bind mount
	if err := syscall.Mount(hostPath, mntURL, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount volume error: %v", err)
	}
	return nil
}
