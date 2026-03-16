// ==================================================================================
// init.go — 容器初始化进程（仅 Linux 平台）
// ==================================================================================
//
// ⚠️ 命名说明：本文件中的 "init" 指的是容器的 init 进程（PID 1），
//    与 Go 语言的 init() 函数（包初始化）完全无关。
//    "容器 init 进程"是 Linux 中每个 PID 命名空间中的第一个进程，类似于系统启动时的 /sbin/init。
//
// 本文件定义了容器的 init 进程逻辑。当子进程在新的 Linux 命名空间中启动后，
// 会执行这里的 RunContainerInitProcess() 函数来完成容器的初始化：
//   1. 挂载 /proc 文件系统（让 ps、top 等命令在容器内正常工作）
//   2. 查找用户命令的可执行文件路径
//   3. 使用 syscall.Exec 将当前进程替换为用户命令
//
// 📁 文件职责分层（从上到下）：
//
//   cli_commands.go         → CLI 参数解析，调用 RunContainer()
//   container_runner.go     → 调度：创建子进程、等待结束
//   namespace_process.go    → 创建带命名空间隔离的子进程
//   init.go                 → 容器内 init 进程：挂载 + exec 替换  ← 你在这里
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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"build-docker/internal/engine/fs"

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
		return errors.New("user command is empty")
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

const FD_INDEX = 3

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(FD_INDEX), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("read user command error: %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
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
	// ── 关键步骤：将当前挂载树设为私有传播（private）──
	//
	// ❓ 为什么需要这一步？
	//    新的 MNT 命名空间（CLONE_NEWNS）默认会「继承」宿主机挂载点的传播属性。
	//    如果宿主机挂载点是 shared 模式（默认），子进程对 /proc 的 mount 操作会
	//    「传播」回宿主机的 MNT 命名空间，污染宿主机的 /proc。
	//    容器退出后，宿主机的 ps 命令就会报：
	//      "Error, do this: mount -t proc proc /proc"
	//
	//    解决方法：在子进程内，先把整个挂载树改为 private（私有）传播，
	//    这样后续的 mount/umount 操作就只对本命名空间可见，不会泄漏到宿主机。
	//
	// 参数说明：
	//   - "":        设备名称（这里是递归操作整个命名空间，不针对某个设备）
	//   - "/":       作用目标：根挂载点，配合 MS_REC 递归影响所有挂载点
	//   - "":        文件系统类型（改传播属性时不需要）
	//   - MS_PRIVATE | MS_REC: 将所有挂载点改为「私有」模式，且递归作用
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		logrus.Errorf("mount private error: %v", err)
	}

	volume := os.Getenv("volume")
	if volume != "" {
		if err := fs.MountVolume(fs.RootPath, volume); err != nil {
			logrus.Errorf("mount volume error: %v", err)
		}
	}

	// ── 关键步骤：执行 pivot_root 切换根目录 ──
	//
	// ❓ 为什么需要这一步？
	//    我们希望容器内部看到的是我们准备好的 rootfs 目录（/root/rootfs/rootfs），
	//    而不是宿主机的真实根目录（/）。
	//    pivot_root 系统调用可以将当前进程的根目录切换到指定目录。
	//
	// 参数说明：
	//   - newRoot:    新的根目录（我们准备好的 rootfs）
	//   - putOld:     旧的根目录（宿主机根目录）的临时存放位置
	//
	// ⚠️ 注意：pivot_root 只能在已挂载了新根目录（newRoot）的前提下使用。
	//    所以这一步必须在 setUpMount() 之后执行。
	if err := setUpPivotRoot(fs.RootPath + "/merged"); err != nil {
		logrus.Errorf("set up pivot root error: %v", err)
	}

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

// setUpPivotRoot 是实现真正文件系统隔离的核心函数！
// 它的目标是把当前的根文件系统切换到传入的 root 目录（比如 busybox 所在目录）。
func setUpPivotRoot(root string) error {
	// ── 第 1 步：Bind Mount 自身 ──
	// 为了让 root 目录成为一个独立的挂载点，满足 pivot_root 的条件。
	// 请在这里使用 syscall.Mount 进行 bind mount 自己（标志使用 syscall.MS_BIND | syscall.MS_REC）
	// YOUR CODE HERE...
	flags := syscall.MS_BIND | syscall.MS_REC
	if err := syscall.Mount(root, root, "bind", uintptr(flags), ""); err != nil {
		return fmt.Errorf("mount bind error: %v", err)
	}
	// ── 第 2 步：创建 put_old 目录 ──
	// 在新的 root 目录下建立一个名为 .put_old 的目录，用来临时存放原来的宿主机根目录。
	// 注意需要拼好路径（root + "/.put_old"）。
	// YOUR CODE HERE...
	oldPath := filepath.Join(root, ".put_old")
	if err := os.Mkdir(oldPath, 0777); err != nil {
		return fmt.Errorf("mkdir error: %v", err)
	}
	// ── 第 3 步：调用 pivot_root ──
	// 调用 syscall.PivotRoot 系统调用，把当前的根（宿主机 /）移到 .put_old 目录，
	// 并把新的 root 目录提升为真正的容器根目录。
	// YOUR CODE HERE...
	if err := syscall.PivotRoot(root, oldPath); err != nil {
		return fmt.Errorf("pivot root error: %v", err)
	}
	// ── 第 4 步：切换工作目录 ──
	// 上一步执行完后，当前进程的工作目录（CWD）可能还在老的地方或者没有明确。
	// 所以需要调用 syscall.Chdir("/")，确保当前进程的工作目录切换到容器里全新的 "/"。
	// YOUR CODE HERE...
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir error: %v", err)
	}
	// ── 第 5 步：卸载并删除旧根目录 ──
	// 旧根目录已经被挂载到了 /.put_old。我们需要：
	// 1. 取消挂载（umount2），使用 MNT_DETACH 标志（懒卸载，即使被占用也会在释放后自动卸载）。
	// 2. 删除（rmdir）空出的 /.put_old 目录。
	// YOUR CODE HERE...
	putOld := filepath.Join("/", ".put_old")
	if err := syscall.Unmount(putOld, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("umount error: %v", err)
	}

	if err := os.Remove(putOld); err != nil {
		return fmt.Errorf("os remove error: %v", err)
	}

	return nil
}
