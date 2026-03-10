//go:build !linux

package fs

import "fmt"

const RootPath = "/root/rootfs"

func NewWorkSpace(rootPath string) error {
	return fmt.Errorf("NewWorkSpace is only supported on Linux")
}

func DeleteWorkSpace(rootPath string) error {
	return fmt.Errorf("DeleteWorkSpace is only supported on Linux")
}
