package main

// SysInfo 采集到的系统信息
type SysInfo struct {
	Name string
	CPU  string
	RAM  string
	Disk string
	SN   string
	MAC  string
	IP   string
}

// CollectSystemInfo 由平台特定文件实现
func CollectSystemInfo() (SysInfo, error) { return SysInfo{}, nil }


