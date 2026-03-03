// ==================================================================================
// init.go — 容器初始化进程（仅 Linux 平台）
// ==================================================================================
//
// 本文件定义了容器的 init 进程逻辑。当子进程在新的 Linux 命名空间中启动后，
// 会执行这里的 RunContainerInitProcess() 函数来完成容器的初始化：
//   1. 挂载 /proc 文件系统（让 ps、top 等命令在容器内正常工作）
//   2. 查找用户命令的可执行文件路径
//   3. 使用 syscall.Exec 将当前进程替换为用户命令
//
// ❓ 为什么要用 syscall.Exec 而不是 exec.Command？
//    syscall.Exec 会"替换"当前进程——不是创建子进程，而是直接将当前进程变成目标程序。
//    这样做的好处是：
//    - 用户命令的 PID 就是 1（容器内的 init 进程）
//    - 不会有多余的父进程占据 PID 1
//    - 与真正的 Docker 行为一致
//
// ==================================================================================

//go:build linux

package container

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus" // 日志库
)

// RunContainerInitProcess 是容器内部的 init 进程入口
//
// 🔄 执行流程：
//  1. 从 os.Args 中读取用户命令（由 NewParentProcess 传入）
//  2. 挂载 /proc 文件系统
//  3. 用 exec.LookPath 在 $PATH 中查找命令的完整路径
//  4. 用 syscall.Exec 将当前进程替换为用户命令
//
// ⚠️ 注意：syscall.Exec 一旦执行成功，这个函数就不会返回了！
//
//	因为当前进程已经被替换成了用户命令。只有在出错时才会返回 error。
func RunContainerInitProcess() error {
	// ── 第 1 步：读取用户命令 ──
	// os.Args 此时的格式为: ["/proc/self/exe", "init", "sh", ...]
	// 我们需要跳过前两项，取出真正的用户命令
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 {
		return nil // 如果没有命令，直接返回（异常情况下的防御性处理）
	}

	// ── 第 2 步：挂载 /proc 文件系统 ──
	// 必须在 syscall.Exec 之前挂载，否则用户命令在容器内无法访问 /proc
	setUpMount()

	// ── 第 3 步：查找用户命令的完整路径 ──
	// exec.LookPath 会在 $PATH 环境变量中搜索命令
	// 例如：LookPath("sh") → "/bin/sh"
	// 这是因为 syscall.Exec 需要命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("exec look path error: %v", err)
		return err
	}
	logrus.Infof("find path: %s", path)

	// ── 第 4 步：用 syscall.Exec 替换当前进程 ──
	// 参数说明：
	//   - path:        可执行文件的完整路径（如 /bin/sh）
	//   - cmdArray:    命令及其参数（如 ["sh"]），argv[0] 是程序名
	//   - os.Environ(): 继承当前进程的所有环境变量
	//
	// ⚠️ 执行成功后，当前进程就"变成"了 /bin/sh（或其他命令），
	//    下面的 return nil 永远不会执行到！
	if err := syscall.Exec(path, cmdArray, os.Environ()); err != nil {
		logrus.Errorf("exec error: %v", err)
		return err
	}
	return nil
}

// readUserCommand 从进程参数中读取用户指定的命令
//
// os.Args 的格式：
//
//	os.Args[0] = "/proc/self/exe"  ← 当前可执行文件路径
//	os.Args[1] = "init"            ← 子命令名称
//	os.Args[2] = "sh"              ← 用户命令（从这里开始）
//	os.Args[3] = ...               ← 用户命令的参数（可选）
//
// 返回值：从 os.Args[2:] 开始的切片，即用户命令及其参数
func readUserCommand() []string {
	// 如果参数不足 3 个，说明没有传入用户命令
	if len(os.Args) < 3 {
		return nil
	}
	// 取出 os.Args[2:] 作为用户命令
	args := os.Args[2:]
	logrus.Infof("command: %s", strings.Join(args, " ")) // 打印即将执行的命令，方便调试
	return args
}

// setUpMount 在容器内部挂载 /proc 文件系统
//
// ❓ 为什么需要挂载 /proc？
//
//	/proc 是 Linux 的虚拟文件系统，里面包含了进程信息、系统信息等。
//	像 ps、top、free 等常用命令都依赖 /proc 来获取数据。
//
//	由于我们创建了新的 PID 命名空间（CLONE_NEWPID），容器内的 /proc 需要重新挂载，
//	否则它还是指向宿主机的进程信息，容器内的 ps 会看到宿主机的所有进程。
//
// 挂载标志说明：
//   - MS_NOEXEC:  不允许在此文件系统上执行程序（安全措施）
//   - MS_NOSUID:  不允许 set-user-ID 和 set-group-ID 生效（安全措施）
//   - MS_NODEV:   不允许访问设备文件（安全措施）
func setUpMount() {
	// 组合挂载标志：禁止执行程序 + 禁止 SUID + 禁止设备访问
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	// 挂载 proc 文件系统到 /proc 目录
	// 参数说明：
	//   - "proc":                  设备名称（对于虚拟文件系统，这只是一个标识符）
	//   - "/proc":                 挂载目标路径
	//   - "proc":                  文件系统类型
	//   - uintptr(defaultMountFlags): 挂载选项
	//   - "":                      附加数据（这里不需要）
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		logrus.Errorf("mount proc error: %v", err)
	}
}
