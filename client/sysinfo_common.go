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
	Network string // WIFI 或 ETHERNET，无法判定可为空字符串
}



