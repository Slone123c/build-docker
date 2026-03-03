// ==================================================================================
// main.go — 程序入口文件
// ==================================================================================
//
// 本文件是整个项目的入口点。我们基于 urfave/cli 库构建了一个命令行工具 "my-docker"，
// 它可以通过子命令（run / init）来创建和初始化容器。
//
// ❓ 为什么需要 CLI 框架？
//    Docker 这样的工具有很多子命令（run、pull、build 等），
//    使用 CLI 框架可以方便地定义和管理这些子命令及其参数。
//
// 🔄 程序的整体执行流程：
//    1. 用户运行: ./my-docker run -it /bin/sh
//    2. main() 解析命令行参数，匹配到 runCommand
//    3. runCommand 调用 Run()，创建一个新的子进程
//    4. 子进程通过 /proc/self/exe 重新运行自己，并传入 init 命令
//    5. init 命令匹配到 initCommand，调用 container.RunContainerInitProcess()
//    6. init 进程设置好容器的文件系统、命名空间，然后用 syscall.Exec 替换为用户命令
//
// ==================================================================================

package main

import (
	"os"

	log "github.com/sirupsen/logrus" // logrus: 一个流行的 Go 日志库，提供结构化日志功能
	"github.com/urfave/cli"          // urfave/cli: Go 语言的命令行应用框架，用于解析命令行参数
)

// usage 定义了程序的使用说明，会显示在 --help 的输出中
const usage = "my-docker is a simple container runtime implementation. The purpose of this project is to learn how docker works and how to write a docker by ourselves."

func main() {
	// ── 第 1 步：创建 CLI 应用实例 ──
	// cli.NewApp() 会创建一个命令行应用的实例，我们可以往里面添加命令、参数等
	app := cli.NewApp()
	app.Name = "my-docker" // 应用名称，会显示在帮助信息中
	app.Usage = usage      // 应用的使用说明

	// ── 第 2 步：注册子命令 ──
	// 我们把 initCommand 和 runCommand 注册到 app 上
	// 这样用户就可以通过 ./my-docker run ... 和 ./my-docker init ... 来触发对应的逻辑
	//
	// ⚠️ 注意：initCommand 是内部使用的，用户不应该直接调用它
	//    它由 runCommand 在创建子进程时自动调用
	app.Commands = []cli.Command{
		initCommand, // init 命令：在容器内部初始化环境（用户不直接调用）
		runCommand,  // run  命令：创建并运行一个新容器
	}

	// ── 第 3 步：设置全局初始化钩子 ──
	// app.Before 会在任何子命令执行之前运行
	// 这里我们用它来配置日志格式和输出位置
	app.Before = func(context *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{}) // 设置日志输出格式为 JSON（结构化日志，方便分析）
		log.SetOutput(os.Stdout)               // 将日志输出到标准输出（控制台）
		return nil
	}

	// ── 第 4 步：运行应用 ──
	// app.Run(os.Args) 会解析命令行参数并执行对应的子命令
	// os.Args 就是用户在终端输入的参数列表，如 ["./my-docker", "run", "-it", "/bin/sh"]
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err) // 如果出错，打印错误并退出
	}
}
