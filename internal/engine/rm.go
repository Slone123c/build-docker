//go:build linux

package engine

import (
	"build-docker/internal/engine/container"
	"build-docker/internal/engine/fs"

	log "github.com/sirupsen/logrus"
)

func RemoveContainer(containerName string, force bool) {

	info, err := container.GetContainerInfo(containerName)
	if err != nil {
		log.Errorf("get container info error: %v", err)
		return
	}

	if !info.IsStopped() {
		if !force {
			log.Errorf("cannot remove running container: %s, use -f to force remove", containerName)
			return
		}

		log.Infof("force remove running container: %s", containerName)
		StopContainer(containerName)

		info, err = container.GetContainerInfo(containerName)
		if err != nil {
			log.Errorf("get container info error: %v", err)
			return
		}
		if info.IsRunning() {
			log.Errorf("failed to stop container: %s", containerName)
			return
		}
	}

	log.Infof("cleaning up container: %s", containerName)
	if err := fs.DeleteWorkSpace(fs.RootPath, info.Id); err != nil {
		log.Errorf("failed to delete workspace: %v", err)
	}

	container.DeleteContainerInfo(info.Id)
	log.Infof("container %s removed", containerName)
}
