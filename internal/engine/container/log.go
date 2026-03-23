package container

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

var logPath = DefaultInfoLocation + "log.log"

func LogContainer(containerName string) {
	// 根据容器名称查找容器 ID
	containerId, err := findContainerIdByName(containerName)
	if err != nil {
		log.Errorf("find container error: %v", err)
		return
	}

	logFilePath := fmt.Sprintf(DefaultInfoLocation, containerId) + "log.log"
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Errorf("open log file error: %v", err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("read log file error: %v", err)
		return
	}
	_, err = fmt.Fprint(os.Stdout, string(content))
	if err != nil {
		log.Errorf("print log file error: %v", err)
		return
	}
}

// findContainerIdByName 根据容器名称或 ID 查找容器 ID
func findContainerIdByName(name string) (string, error) {
	files, err := os.ReadDir(DefaultInfoBaseDir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		configPath := DefaultInfoBaseDir + file.Name() + "/" + ConfigName
		content, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		var info Info
		if err := json.Unmarshal(content, &info); err != nil {
			continue
		}

		// 支持按名称或 ID 查找
		if info.Name == name || info.Id == name {
			return info.Id, nil
		}
	}

	return "", fmt.Errorf("container %s not found", name)
}
