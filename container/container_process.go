// ==================================================================================
// container_process.go — 创建容器父进程（仅 Linux 平台）
// ==================================================================================
//
// 本文件定义了 NewParentProcess() 函数，它是容器创建的核心。
// 通过 Linux 内核提供的命名空间（Namespace）机制，创建一个与宿主机隔离的子进程。
//
// ❓ 什么是命名空间（Namespace）？
//    Linux 命名空间是内核提供的资源隔离机制，可以让进程"看到"独立的：
//    - UTS:  主机名（让容器有自己的 hostname）
//    - PID:  进程 ID（容器内的进程 PID 从 1 开始）
//    - MNT:  文件系统挂载点（容器可以有独立的文件系统视图）
//    - NET:  网络栈（容器有自己的网卡、IP 地址等）
//    - IPC:  进程间通信（容器有独立的消息队列、信号量等）
//
// ❓ 什么是 /proc/self/exe？
//    这是一个特殊的文件路径，始终指向当前正在运行的可执行文件自身。
//    也就是说，exec.Command("/proc/self/exe", "init", ...) 会重新运行 my-docker 程序本身，
//    但这次传入的参数是 "init"，所以会匹配到 initCommand，进入容器初始化流程。
//
//    🔑 这是 Docker 的一个经典技巧：通过调用自身来实现「fork + exec」的效果，
//       同时利用 Go 运行时在 clone() 时设置好命名空间。
//
// ==================================================================================

//go:build linux

package container

import (
	"os"
	"os/exec"
	"syscall"
)

// NewParentProcess 创建一个配置了 Linux 命名空间的子进程命令
//
// 参数说明：
//   - tty:      是否将子进程的标准输入/输出/错误连接到当前终端
//   - cmdArray: 用户指定的命令，如 ["/bin/sh"]
//
// 返回值：
//   - *exec.Cmd: 配置好的 Cmd 对象，调用 Start() 即可启动子进程
//
// 🔄 工作原理：
//  1. 构造参数: ["init", "/bin/sh"]
//  2. 创建命令: /proc/self/exe init /bin/sh
//     → 相当于重新运行 my-docker 程序，参数为 "init /bin/sh"
//  3. 通过 SysProcAttr.Cloneflags 配置 Linux 命名空间
//     → 子进程启动时就已经在新的命名空间中了
//  4. 如果 tty=true，将子进程的 IO 连接到终端
func NewParentProcess(tty bool, cmdArray []string) *exec.Cmd {
	// 将 "init" 作为第一个参数，后面跟上用户命令
	// 这样子进程启动后，CLI 框架会解析到 init 子命令
	args := append([]string{"init"}, cmdArray...)

	// 创建命令：让程序调用自身（/proc/self/exe），传入上面的参数
	// 等效于在终端运行：/path/to/my-docker init /bin/sh
	cmd := exec.Command("/proc/self/exe", args...)

	// ── 🔑 核心：设置 Linux 命名空间 ──
	// 通过 Cloneflags 指定子进程需要创建哪些新的命名空间
	// 每个 CLONE_NEW* 标志表示创建一种命名空间：
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | // UTS 命名空间：隔离主机名和域名
			syscall.CLONE_NEWPID | //  PID 命名空间：隔离进程 ID（容器内 PID 从 1 开始）
			syscall.CLONE_NEWNS | //   MNT 命名空间：隔离文件系统挂载点
			syscall.CLONE_NEWNET | //  NET 命名空间：隔离网络栈（网卡、IP、端口等）
			syscall.CLONE_NEWIPC, //   IPC 命名空间：隔离进程间通信资源
	}

	// 如果开启了交互模式（-it），将子进程的标准 IO 连接到当前终端
	// 这样用户就可以直接在终端与容器内的进程交互（比如使用 /bin/sh）
	if tty {
		cmd.Stdin = os.Stdin   // 标准输入：从终端读取用户输入
		cmd.Stdout = os.Stdout // 标准输出：输出到终端
		cmd.Stderr = os.Stderr // 标准错误：错误信息也输出到终端
	}

	return cmd // 返回配置好的 Cmd 对象，由 run.go 中的 Run() 函数负责启动
}
