//go:build linux

package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const RootPath = "/root/rootfs"
const lowerDir = "busybox"

// NewWorkSpace 负责创建一个支持 OverlayFS 的隔离工作区
//
// rootPath: 我们工作区的根目录（比如 /root/rootfs）。
// 我们约定在这个目录下会有 4 个子目录：
//   - lower: 存放原始的 busybox 文件（只读镜像层）
//   - upper: 容器内产生的新文件和变动存放在这里（可写层）
//   - work:  OverlayFS 内部使用的工作目录（不用管内容，必须有）
//   - merged: 将 lower 和 upper 合并起来挂载到这里的目录（容器最终看到的世界）
func NewWorkSpace(rootPath string) error {
	// ── 第 1 步：准备这 4 个目录的绝对路径 ──
	lower := filepath.Join(rootPath, lowerDir)
	upper := filepath.Join(rootPath, "upper")
	work := filepath.Join(rootPath, "work")
	merged := filepath.Join(rootPath, "merged")

	// ── 第 2 步：创建 upper, work, merged 目录 ──
	// 使用 os.MkdirAll 帮它们创建出来，权限可以用 0777。
	// 注意：lower 目录应该已经放好了 busybox，不需要创建。
	// YOUR CODE HERE...
	os.MkdirAll(upper, 0777)
	os.MkdirAll(work, 0777)
	os.MkdirAll(merged, 0777)

	// ── 第 3 步：拼接 OverlayFS 专用的挂载参数 ──
	// mount -t overlay ... 中的 -o 参数部分：
	// 格式要求： "lowerdir=/...,upperdir=/...,workdir=/..."
	// 请在这里用 fmt.Sprintf 把上面拼好的三个绝对路径塞进去。
	// YOUR CODE HERE...
	data := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)

	// ── 第 4 步：调用 syscall.Mount 挂载 OverlayFS ──
	// 把上面这三层，以 "overlay" 文件系统的类型，挂载到 target（也就是 merged 目录）上。
	// 参数：
	// source: "overlay"
	// target: merged 目录的路径
	// fstype: "overlay"
	// flags: 0
	// data: 第 3 步拼接好的参数字符串
	// YOUR CODE HERE...
	if err := syscall.Mount("overlay", merged, "overlay", 0, data); err != nil {
		return fmt.Errorf("mount overlay error: %v", err)
	}

	return nil
}

func DeleteWorkSpace(rootPath string) error {
	merged := filepath.Join(rootPath, "merged")
	syscall.Unmount(merged, syscall.MNT_DETACH)

	os.RemoveAll(filepath.Join(rootPath, "upper"))
	os.RemoveAll(filepath.Join(rootPath, "work"))
	os.RemoveAll(filepath.Join(rootPath, "merged"))
	return nil
}
