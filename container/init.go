//go:build linux

package container

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

// RunContainerInitProcess 在容器内部运行，是用户指定命令的真正 init 进程
// 使用 syscall.Exec 替换当前进程（保持 PID=1），挂载 proc 文件系统
func RunContainerInitProcess() error {
	// 从 os.Args 获取用户命令（os.Args = ["/proc/self/exe", "init", "<cmd>", ...]）
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 {
		return nil
	}

	setUpMount()

	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("exec look path error: %v", err)
		return err
	}
	logrus.Infof("find path: %s", path)

	if err := syscall.Exec(path, cmdArray, os.Environ()); err != nil {
		logrus.Errorf("exec error: %v", err)
		return err
	}
	return nil
}

// readUserCommand 从进程参数中读取用户命令
// os.Args 格式: ["/proc/self/exe", "init", "sh", ...]
// 跳过前两项（可执行路径 + "init"），返回实际命令及其参数
func readUserCommand() []string {
	if len(os.Args) < 3 {
		return nil
	}
	args := os.Args[2:]
	logrus.Infof("command: %s", strings.Join(args, " "))
	return args
}

// setUpMount 设置容器内的文件系统挂载
func setUpMount() {
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		logrus.Errorf("mount proc error: %v", err)
	}
}
