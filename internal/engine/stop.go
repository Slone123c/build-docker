package engine

import (
	"build-docker/internal/engine/container"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func StopContainer(containerName string) {

	info, err := container.GetContainerInfo(containerName)
	if err != nil {
		log.Errorf("get container info error: %v", err)
		return
	}

	if info.IsStopped() {
		log.Errorf("container %s has already stopped", containerName)
		return
	}

	if !info.HasPid() {
		log.Errorf("container %s has no pid", containerName)
		return
	}

	pid, err := strconv.Atoi(info.Pid)
	if err != nil {
		log.Errorf("invalid pid %s: %v", info.Pid, err)
		return
	}

	log.Infof("sending SIGTERM to container %s (pid: %d)", containerName, pid)
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			log.Errorf("container %s has already stopped", containerName)
		} else {
			log.Errorf("failed to kill container %s: %v", containerName, err)
			return
		}
	}

	const timeout = 10 * time.Second
	const interval = 200 * time.Millisecond
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			log.Infof("container %s (pid %d) exited gracefully", containerName, pid)
			goto updateStatus
		}
		time.Sleep(interval)
	}

	log.Warnf("container %s (pid %d) did not exit gracefully in %v, sending SIGKILL", containerName, pid, timeout)
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		log.Errorf("SIGKILL failed for container %s: %v", containerName, err)
		return
	}
	time.Sleep(300 * time.Millisecond)

updateStatus:
	if err := container.UpdateContainerStatus(info.Id, "STOPPED"); err != nil {
		log.Errorf("failed to update container %s status: %v", containerName, err)
		return
	}
	log.Infof("container %s (pid %d) stopped", containerName, pid)
}

func processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}
