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
var ConfigName = "config.json"

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

type InitInfo struct {
	ContainerPID  int
	CommandArray  []string
	ContainerName string
	ContainerId   string
}

func RecordContainerInfo(r InitInfo) error {
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
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		return err
	}
	fileName := dirUrl + "/" + ConfigName
	if err := os.WriteFile(fileName, []byte(jsonStr), 0622); err != nil {
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
