package subsystems

type ResourceConfig struct {
	MemoryLimit string
	// cpu时间片权重
	CpuShare string
	// CPU核心数
	CpuSet string
}

type Subsystem interface {
	Name() string
	Set(path string, res *ResourceConfig) error
	Apply(path string, pid int) error
	Remove(path string) error
}

var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)
