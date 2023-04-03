package cgroups

import (
	"example/mydocker/cgroups/subsystems"

	log "github.com/sirupsen/logrus"
)

type CgroupManager struct {
	Path     string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Apply(c.Path, pid)
	}
	return nil
}

func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Set(c.Path, res)
	}
	return nil
}

func (c *CgroupManager) Remove() error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			log.Warnf("remove cgroup %s fail %v", c.Path, err)
		}
	}
	return nil
}
