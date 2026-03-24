package engine

import (
	"build-docker/internal/engine/container"
	_ "build-docker/internal/engine/nsenter"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	ExecPidEnv = "MYDOCKER_EXEC_PID"
	ExecCmdEnv = "MYDOCKER_EXEC_CMD"
)

func ExecContainer(containerName string, cmdArray []string) {
	if os.Getenv(ExecPidEnv) != "" {
		log.Infof("container %s already exec, pid %s", containerName, os.Getenv(ExecPidEnv))
		return
	}

	pid, err := getContainerPid(containerName)
	if err != nil {
		log.Errorf("get container %s pid err: %v", containerName, err)
		return
	}
	log.Infof("container %s pid %s", containerName, pid)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%s", ExecPidEnv, pid),
		fmt.Sprintf("%s=%s", ExecCmdEnv, strings.Join(cmdArray, " ")),
	)
	if err := cmd.Run(); err != nil {
		log.Errorf("exec container %s err: %v", containerName, err)
		return
	}
}

func getContainerPid(name string) (string, error) {
	files, err := os.ReadDir(container.DefaultInfoBaseDir)
	if err != nil {
		return "", err
	}

	for _, f := range files {
		path := container.DefaultInfoBaseDir + f.Name() + "/" + container.CONFIG_NAME
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		var info container.Info
		if err := json.Unmarshal(data, &info); err != nil {
			return "", err
		}
		if info.Name == name || info.Id == name {
			return info.Pid, nil
		}
	}
	return "", fmt.Errorf("container %s not found", name)
}
