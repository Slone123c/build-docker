// ==================================================================================
// cli_commands.go — CLI 子命令定义文件
// ==================================================================================
//
// 本文件定义了两个 CLI 子命令：
//   - runCommand:  用户直接调用，用于创建并运行一个新容器
//   - initCommand: 内部调用，用于在容器内部完成初始化
//
// ❓ 为什么需要两个命令？
//    Docker 的容器创建分为两个阶段：
//    1. 父进程阶段（run）：在宿主机上创建子进程，为子进程设置 Linux 命名空间
//    2. 子进程阶段（init）：在隔离的命名空间中，挂载 proc 等文件系统，
//       然后用 syscall.Exec 替换为用户指定的命令（如 /bin/sh）
//
//    这是因为 Linux 的命名空间隔离必须在进程创建时通过 clone() 系统调用设置，
//    所以需要先 fork 出一个子进程，然后在子进程内部再做初始化工作。
//
// 📁 本文件（cli_commands.go）是"命令层"，只负责解析 CLI 参数、调用下层函数：
//
//    runCommand.Action()  → RunContainer()  （在 container_runner.go 中定义）
//    initCommand.Action() → container.RunContainerInitProcess()  （在 container/init.go 中定义）
//
// ==================================================================================

package main

import (
	"fmt"

	"build-docker/container" // 引入容器相关逻辑的子包
	"build-docker/subsystem"

	log "github.com/sirupsen/logrus" // 日志库
	"github.com/urfave/cli"          // CLI 框架
)

// ── runCommand：创建容器的主命令 ──
//
// 用法示例：
//
//	./my-docker run -it /bin/sh
//	./my-docker run -it /bin/ls
//
// 参数说明：
//   - -it: 开启交互模式（将标准输入/输出连接到容器内）
//   - 最后的参数: 要在容器内运行的命令
var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a new container with namespace: mydocker run -it [command]",

	// Flags 定义了该命令支持的参数/选项
	Flags: []cli.Flag{
		// BoolFlag 表示一个布尔类型的标志（不需要值，出现即为 true）
		cli.BoolFlag{
			Name:  "it",                                       // 参数名称，用户通过 -it 来使用
			Usage: "enable stdin/stdout and interactive mode", // 参数说明
		},
		cli.StringFlag{
			Name:  "mem", // 限制进程内存使用量，为了避免和 stress 命令的 -m 参数冲突 这里使用 -mem,到时候可以看下解决冲突的方法
			Usage: "memory limit,e.g.: -mem 100m",
		},
		cli.StringFlag{
			Name:  "cpu",
			Usage: "cpu quota,e.g.: -cpu 100", // 限制进程 cpu 使用率
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit,e.g.: -cpuset 2,4", // 限制进程 cpu 使用率
		},
	},

	// Action 是当用户执行 run 命令时实际执行的函数
	Action: func(context *cli.Context) error {
		// 检查用户是否提供了要在容器内运行的命令
		// context.NArg() 返回非 flag 参数的数量
		// 例如 "./my-docker run -it /bin/sh" 中，NArg() = 1（/bin/sh）
		if context.NArg() < 1 {
			return fmt.Errorf("please provide a command to run")
		}

		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		tty := context.Bool("it")
		resConf := &subsystem.ResourceConfig{
			MemoryLimit: context.String("mem"),
			CpuShare:    context.String("cpu"),
			CpuSet:      context.String("cpuset"),
		}

		// 调用 RunContainer() 函数（定义在 container_runner.go 中）
		// 它会创建一个带有 Linux 命名空间隔离的子进程来运行用户命令
		RunContainer(tty, cmdArray, resConf)
		return nil
	},
}

// ── initCommand：容器初始化命令（内部使用，用户不应直接调用） ──
//
// ❓ 为什么用户不应该调用这个命令？
//
//	因为 initCommand 只有在已经创建了新的 Linux 命名空间的子进程中运行才有意义。
//	它需要在隔离的命名空间中挂载 /proc 文件系统，并替换为用户命令。
//	如果在宿主机上直接调用，会破坏宿主机的 /proc 挂载并带来安全风险。
//
// 🔄 调用流程：
//  1. runCommand → RunContainer() → container.NewParentProcess() 创建子进程
//  2. 子进程通过 /proc/self/exe init <cmd> 重新调用自身
//  3. CLI 框架匹配到 initCommand，执行下面的 Action
//  4. container.RunContainerInitProcess() 完成容器内部初始化
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",

	// Action 是 init 命令实际执行的函数
	// 当子进程以 "./my-docker init <cmd>" 启动时会执行这里
	Action: func(context *cli.Context) error {
		log.Infof("init come on") // 打印日志表示 init 命令已开始执行

		// 调用容器初始化函数（定义在 container/init.go 中）
		// 该函数会：1) 挂载 /proc  2) 查找用户命令的路径  3) 用 syscall.Exec 替换当前进程
		err := container.RunContainerInitProcess()
		return err
	},
}
