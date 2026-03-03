// ==================================================================================
// stub_darwin.go — macOS/非 Linux 平台的桩代码（Stub）
// ==================================================================================
//
// ❓ 为什么需要这个文件？
//    容器核心功能（命名空间、cgroup 等）是 Linux 内核特有的，macOS 和 Windows 并不支持。
//    但我们可能需要在 macOS 上编译这个项目（比如用来进行代码检查、IDE 跳转等），
//    所以需要提供一组"桩函数"（Stub），让编译器在非 Linux 平台也能通过编译。
//
//    这些桩函数不是真正的实现，调用它们会 panic 或返回错误。
//
// 📝 编译标签说明：
//    //go:build !linux
//    这行编译标签（Build Tag）告诉 Go 编译器：
//    只有在目标操作系统"不是 Linux"时，才编译这个文件。
//    对应地，container_process.go 和 init.go 上有 //go:build linux 标签，
//    表示它们只在 Linux 上编译。
//    这样两套文件互不冲突，编译器会根据目标平台自动选择。
//
// ==================================================================================

//go:build !linux

package container

import (
	"fmt"
	"os/exec"
)

// NewParentProcess 是 macOS/非 Linux 平台的桩函数
//
// 由于 Linux 命名空间在 macOS 上不可用，这个函数不提供实际实现。
// 如果被调用会直接 panic，提醒开发者需要在 Linux 上运行。
//
// 如需在 Linux 上运行，请使用：GOOS=linux go build
func NewParentProcess(tty bool, cmdArray []string) *exec.Cmd {
	panic("NewParentProcess is only supported on Linux")
}

// RunContainerInitProcess 是 macOS/非 Linux 平台的桩函数
//
// 同上，容器初始化需要 Linux 特有的系统调用（mount、exec 等），
// 在非 Linux 平台无法实现。
func RunContainerInitProcess() error {
	return fmt.Errorf("RunContainerInitProcess is only supported on Linux")
}
