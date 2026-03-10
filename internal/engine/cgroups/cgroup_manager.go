package cgroups

import (
	log "github.com/sirupsen/logrus"
)

type CgroupManager struct {
	Path     string
	Resource *ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

func (c *CgroupManager) Apply(pid int) error {
	for _, subIns := range SubsystemsIns {
		if err := subIns.Apply(c.Path, pid); err != nil {
			log.Warnf("apply cgroup %s error: %v", c.Path, err)
			return err
		}
	}
	return nil
}

func (c *CgroupManager) Set(res *ResourceConfig) error {
	for _, subIns := range SubsystemsIns {
		if err := subIns.Set(c.Path, res); err != nil {
			log.Warnf("set cgroup %s error: %v", c.Path, err)
			return err
		}
	}
	return nil
}

func (c *CgroupManager) Destroy() error {
	for _, subIns := range SubsystemsIns {
		if err := subIns.Remove(c.Path); err != nil {
			log.Warnf("remove cgroup %s error: %v", c.Path, err)
			return err
		}
	}
	return nil
}
