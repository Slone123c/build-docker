#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

// 这个函数在 Go runtime 启动之前执行
// 通过环境变量接收参数，避免解析 os.Args（此时还不可用）
__attribute__((constructor)) void nsenter() {
    char *pid = getenv("MYDOCKER_EXEC_PID");
    char *cmd = getenv("MYDOCKER_EXEC_CMD");

    // 没有设置环境变量，说明是普通启动，直接返回
    if (pid == NULL || cmd == NULL) {
        return;
    }

    // 需要加入的 namespace 类型
    char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};
    int  ns_flags[]    = {CLONE_NEWIPC, CLONE_NEWUTS, CLONE_NEWNET,
                          CLONE_NEWPID, CLONE_NEWNS};
    int  ns_count      = 5;

    for (int i = 0; i < ns_count; i++) {
        char ns_path[64];
        snprintf(ns_path, sizeof(ns_path), "/proc/%s/ns/%s", pid, namespaces[i]);

        int fd = open(ns_path, O_RDONLY);
        if (fd < 0) {
            fprintf(stderr, "nsenter: open %s failed: %s\n",
                    ns_path, strerror(errno));
            exit(1);
        }

        if (setns(fd, ns_flags[i]) < 0) {
            fprintf(stderr, "nsenter: setns %s failed: %s\n",
                    ns_path, strerror(errno));
            exit(1);
        }
        close(fd);
    }

    // 成功加入所有 namespace，执行目标命令
    // cmd 格式："/bin/sh arg1 arg2"
    // 这里简化处理，实际需要 shell 解析或拆分参数
    int ret = execlp("/bin/sh", "/bin/sh", "-c", cmd, NULL);
    if (ret < 0) {
        fprintf(stderr, "nsenter: exec failed: %s\n", strerror(errno));
        exit(1);
    }
}