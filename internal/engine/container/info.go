package container

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var DefaultInfoLocation = "/var/run/mydocker/%s/"
var DefaultInfoBaseDir = "/var/run/mydocker/"

const CONFIG_NAME = "config.json"

func RandContainerName(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz1234567890"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type Info struct {
	Pid         string
	Id          string
	Name        string
	Command     string
	CreatedTime string
	Status      string
}

func (info *Info) IsRunning() bool {
	return info.Status == "RUNNING"
}

func (info *Info) IsStopped() bool {
	return info.Status == "STOPPED"
}

func (info *Info) HasPid() bool {
	return info.Pid != ""
}

type InitInfo struct {
	ContainerPID  int
	CommandArray  []string
	ContainerName string
	ContainerId   string
	IP            string
}

func CreateContainerInfo(r InitInfo) error {
	if r.ContainerName == "" {
		r.ContainerName = r.ContainerId
	}
	command := strings.Join(r.CommandArray, ",")
	containerInfo := &Info{
		Pid:         strconv.Itoa(r.ContainerPID),
		Id:          r.ContainerId,
		Name:        r.ContainerName,
		Command:     command,
		CreatedTime: time.Now().Format("2006-01-02 15:04:05"),
		Status:      "RUNNING",
	}
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		return err
	}
	jsonStr := string(jsonBytes)
	dirUrl := fmt.Sprintf(DefaultInfoLocation, r.ContainerId)
	if err := os.MkdirAll(dirUrl, 0755); err != nil {
		return err
	}
	fileName := dirUrl + CONFIG_NAME
	if err := os.WriteFile(fileName, []byte(jsonStr), 0644); err != nil {
		return err
	}
	return nil
}

func DeleteContainerInfo(containerID string) {
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerID)
	if err := os.RemoveAll(dirUrl); err != nil {
		log.Errorf("delete container info error: %v", err)
	}
}

func UpdateContainerStatus(containerID string, status string) error {
	configPath := fmt.Sprintf(DefaultInfoLocation, containerID) + CONFIG_NAME
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var containerInfo Info
	if err := json.Unmarshal(data, &containerInfo); err != nil {
		return err
	}

	containerInfo.Status = status
	if status == "STOPPED" {
		containerInfo.Pid = ""
	}

	newInfo, err := json.Marshal(containerInfo)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newInfo, 0644)
}

func GetContainerInfo(containerName string) (*Info, error) {
	files, err := os.ReadDir(DefaultInfoBaseDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		configPath := fmt.Sprintf(DefaultInfoLocation, file.Name()) + CONFIG_NAME
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue // 跳过无法读取的文件
		}
		var containerInfo Info
		if err := json.Unmarshal(data, &containerInfo); err != nil {
			continue // 跳过格式错误的数据
		}
		// 匹配容器名或容器ID
		if containerInfo.Name == containerName || containerInfo.Id == containerName {
			return &containerInfo, nil
		}
	}
	return nil, fmt.Errorf("container %s not found", containerName)
}
