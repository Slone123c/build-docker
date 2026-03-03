# 📖 build-docker 代码阅读路线图

> 本项目通过手写一个简易 Docker，深入理解容器的底层原理。
> 建议按照下面的路线图顺序阅读代码，循序渐进。

---

## 🗺️ 项目结构速览

```
build-docker/
├── main.go                        # 入口：CLI 应用初始化
├── command.go                     # 命令定义：run / init 子命令
├── run.go                         # 核心：创建父进程并等待
├── go.mod                         # Go 模块依赖
└── container/
    ├── container_process.go       # Linux：创建带命名空间的子进程
    ├── init.go                    # Linux：容器 init 进程逻辑
    └── stub_darwin.go             # macOS：编译桩，不含实际逻辑
```

---

## 🚀 第一阶段：理解整体流程（入口层）

### 第 1 步 — `main.go`
**目标**：了解程序如何启动，CLI 框架如何工作

- 了解 `urfave/cli` 框架的基本用法
- 看懂 `app.Commands` 如何注册子命令
- 了解 `app.Before` 钩子的作用（配置日志）

**关键问题**：程序启动后，是如何找到 `run` 或 `init` 子命令的？

---

### 第 2 步 — `command.go`
**目标**：搞清楚两个子命令的职责分工

- `runCommand`：用户直接调用，创建容器
- `initCommand`：内部调用，初始化容器内部环境

**关键问题**：为什么需要两个命令？用户调用 `run`，`init` 是什么时候被调用的？

---

### 第 3 步 — `run.go`
**目标**：理解父进程的创建和等待

- `NewParentProcess()` 返回什么？
- `Start()` 和 `Wait()` 分别做什么？

**关键问题**：`parent.Start()` 启动的子进程，会做什么事情？

---

## 🔧 第二阶段：深入容器核心（实现层）

### 第 4 步 — `container/container_process.go`（仅 Linux）
**目标**：理解 Linux 命名空间隔离机制

重点关注以下内容：

| 知识点 | 说明 |
|--------|------|
| `/proc/self/exe` | 为何让程序调用自身？ |
| `SysProcAttr.Cloneflags` | 如何通过 clone() 创建新命名空间？ |
| `CLONE_NEWUTS` | 隔离主机名 |
| `CLONE_NEWPID` | 隔离进程 ID，容器内 PID 从 1 开始 |
| `CLONE_NEWNS` | 隔离文件系统挂载点 |
| `CLONE_NEWNET` | 隔离网络栈 |
| `CLONE_NEWIPC` | 隔离进程间通信 |
| `tty` 参数 | 如何将容器 IO 连接到终端？ |

**关键问题**：命名空间是在哪个时刻生效的——`exec.Command()` 时还是 `Start()` 时？

---

### 第 5 步 — `container/init.go`（仅 Linux）
**目标**：理解容器 init 进程的工作原理

阅读顺序：

1. `readUserCommand()` — 如何从 `os.Args` 解析出用户命令？
2. `setUpMount()` — 为什么要在容器内重新挂载 `/proc`？
3. `RunContainerInitProcess()` — `syscall.Exec` 如何"替换"当前进程？

| 知识点 | 说明 |
|--------|------|
| `os.Args` 结构 | `["/proc/self/exe", "init", "sh", ...]` |
| `exec.LookPath` | 在 `$PATH` 中查找命令的完整路径 |
| `syscall.Exec` | 替换进程而非新建子进程，保持 PID=1 |
| `/proc` 挂载 | 让 `ps`、`top` 等命令在容器内正常工作 |
| 挂载标志 | `MS_NOEXEC`、`MS_NOSUID`、`MS_NODEV` 的安全意义 |

**关键问题**：`syscall.Exec` 成功后，`RunContainerInitProcess()` 函数还会返回吗？为什么？

---

### 第 6 步 — `container/stub_darwin.go`（macOS）
**目标**：了解 Go 跨平台编译的处理方式

- 理解编译标签 `//go:build linux` 和 `//go:build !linux` 的含义
- 了解为什么需要桩（Stub）函数

---

## 🔄 完整执行流程图

```
用户执行: ./my-docker run -it /bin/sh
         │
         ▼
      main.go
    app.Run(os.Args)
         │
         ▼
     command.go
    runCommand.Action()
      ├─ 解析 -it 参数 → tty = true
      └─ 解析命令参数 → cmdArray = ["/bin/sh"]
         │
         ▼
       run.go
      Run(tty=true, ["/bin/sh"])
         │
         ▼
  container/container_process.go
  NewParentProcess(tty, cmdArray)
    ├─ args = ["init", "/bin/sh"]
    ├─ cmd = exec.Command("/proc/self/exe", "init", "/bin/sh")
    ├─ 设置 Cloneflags（5 种命名空间）
    └─ 连接 stdin/stdout/stderr
         │
    parent.Start()  ← 启动子进程，命名空间在此刻生效
         │
         │（子进程开始，已在新命名空间中）
         ▼
      main.go（子进程中重新执行）
    app.Run(["init", "/bin/sh"])
         │
         ▼
     command.go
    initCommand.Action()
         │
         ▼
  container/init.go
  RunContainerInitProcess()
    ├─ readUserCommand() → ["/bin/sh"]
    ├─ setUpMount()      → 挂载 /proc
    ├─ LookPath("/bin/sh") → "/bin/sh"
    └─ syscall.Exec("/bin/sh", ["/bin/sh"], env)
         │
         ▼
   🎉 容器内运行 /bin/sh（PID=1）
```

---

## 📚 延伸阅读

在读完代码后，可以进一步了解以下概念来加深理解：

- **Linux Namespace**：[man7.org - namespaces(7)](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- **Linux proc 文件系统**：[man7.org - proc(5)](https://man7.org/linux/man-pages/man5/proc.5.html)
- **Go 编译标签**：[Go 官方文档 - Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- **syscall.Exec vs exec.Command**：前者替换进程，后者创建子进程

---

## 🧪 动手实验建议

> ⚠️ 以下实验需要在 **Linux 环境**中执行（可使用虚拟机或 Docker 容器）

```bash
# 1. 编译项目
GOOS=linux GOARCH=amd64 go build -o my-docker .

# 2. 运行容器，进入交互式 shell
sudo ./my-docker run -it /bin/sh

# 3. 在容器内验证隔离效果
ps aux          # 只会看到容器内的进程（PID 从 1 开始）
hostname        # 尝试修改主机名，不影响宿主机
cat /proc/1/status  # 查看 PID 1 的进程信息
```

---

*最后更新：2026-03-03*
