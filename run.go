package main

import (
	"build-docker/container"
	"os"

	log "github.com/sirupsen/logrus"
)

// Run 创建父进程并等待其完成
func Run(tty bool, cmdArray []string) {
	parent := container.NewParentProcess(tty, cmdArray)
	if err := parent.Start(); err != nil {
		log.Fatal(err)
	}
	if err := parent.Wait(); err != nil {
		log.Errorf("container process exited with error: %v", err)
		os.Exit(1)
	}
}
