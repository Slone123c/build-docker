// ==================================================================================
// run.go — 容器运行入口
// ==================================================================================
//
// 本文件定义了 Run() 函数，它是创建和运行容器的核心入口。
// 由 command.go 中的 runCommand 调用。
//
// ==================================================================================

package main

import (
	"build-docker/container" // 容器子包，负责创建进程和初始化容器
	"build-docker/subsystem"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus" // 日志库
)

// Run 是创建和运行容器的入口函数
//
// 参数说明：
//   - tty:      是否开启交互式终端（当 -it 参数被传入时为 true）
//   - cmdArray: 用户指定的命令及其参数，如 ["/bin/sh"] 或 ["/bin/ls", "-l"]
//
// 🔄 执行流程：
//  1. 调用 container.NewParentProcess() 创建一个配置了 Linux 命名空间的子进程命令
//  2. parent.Start() 启动该子进程
//     → 子进程会执行 /proc/self/exe init <cmd>，最终触发 initCommand
//  3. parent.Wait() 等待子进程结束
//     → 类似于在终端执行一条命令后等它跑完
func Run(tty bool, cmdArray []string, res *subsystem.ResourceConfig) {
	// 创建"父进程"（实际是一个 exec.Cmd 对象）
	// 这个进程一旦启动，就已经在新的 Linux 命名空间中了
	parent, writePipe, err := container.NewParentProcess(tty)
	if err != nil {
		logrus.Errorf("New parent process error")
		return
	}

	// 启动子进程
	// Start() 不会等待进程结束，只是将子进程启动起来
	if err := parent.Start(); err != nil {
		log.Fatal(err) // 启动失败则打印错误并退出
	}

	cgroupManager := NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(cmdArray, writePipe)

	// 等待子进程运行结束
	// Wait() 会阻塞，直到子进程退出
	// 子进程内部会用 syscall.Exec 替换为用户命令（如 /bin/sh），
	// 所以这里等待的实际上是用户命令的执行
	if err := parent.Wait(); err != nil {
		log.Errorf("container process exited with error: %v", err)
		os.Exit(1) // 如果子进程退出时有错误，以非零状态码退出
	}
}

func sendInitCommand(cmArray []string, writePipe *os.File) {
	command := strings.Join(cmArray, " ")
	logrus.Infof("send command to init process: %s", command)
	_, err := writePipe.WriteString(command)
	if err != nil {
		logrus.Errorf("send command to init process error: %v", err)
	}
	writePipe.Close()
}
