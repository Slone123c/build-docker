package container

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
)

func ListContainers() {
	files, err := os.ReadDir(DefaultInfoBaseDir)
	if err != nil {
		log.Errorf("read dir error: %v", err)
	}

	containers := make([]*Info, 0, len(files))
	for _, file := range files {
		containerInfo, err := getContainerInfo(file)
		if err != nil {
			log.Warnf("get container info error: %v", err)
			continue // 跳过错误的容器，不要添加 nil
		}
		containers = append(containers, containerInfo)
	}
	//  格式化打印容器信息 go-pretty

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, err = fmt.Fprint(w, "ID\tName\tPID\tStatus\tCommand\tCreatedTime\n")
	if err != nil {
		log.Errorf("print error: %v", err)
	}
	for _, container := range containers {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			container.Id,
			container.Name,
			container.Pid,
			container.Status,
			container.Command,
			container.CreatedTime)
		if err != nil {
			log.Errorf("print error: %v", err)
		}
	}

	if err = w.Flush(); err != nil {
		log.Errorf("flush error: %v", err)
	}
}

func getContainerInfo(file os.DirEntry) (*Info, error) {
	filePath := DefaultInfoBaseDir + file.Name() + "/" + CONFIG_NAME
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Warnf("read file error: %v", err)
		return nil, err
	}
	info := new(Info)
	if err := json.Unmarshal(content, info); err != nil {
		log.Errorf("unmarshal error: %v", err)
		return nil, err
	}
	return info, nil
}
