//go:build linux

package container

import (
	"os"
	"os/exec"
	"syscall"
)

// NewParentProcess 创建父进程（宿主进程），通过 /proc/self/exe 调用自身的 init 子命令
// 将容器内要运行的命令作为参数传递给子进程
func NewParentProcess(tty bool, cmdArray []string) *exec.Cmd {
	args := append([]string{"init"}, cmdArray...)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}
