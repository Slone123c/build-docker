//go:build !linux

package container

import (
	"fmt"
	"os/exec"
)

// NewParentProcess is a stub for non-Linux platforms.
// This tool requires Linux to run; build with GOOS=linux.
func NewParentProcess(tty bool, cmdArray []string) *exec.Cmd {
	panic("NewParentProcess is only supported on Linux")
}

// RunContainerInitProcess is a stub for non-Linux platforms.
// This tool requires Linux to run; build with GOOS=linux.
func RunContainerInitProcess() error {
	return fmt.Errorf("RunContainerInitProcess is only supported on Linux")
}
