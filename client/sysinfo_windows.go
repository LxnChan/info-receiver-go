//go:build windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func CollectSystemInfo() (SysInfo, error) {
	var info SysInfo

	// 计算机名
	info.Name = os.Getenv("COMPUTERNAME")
	if info.Name == "" { if h, _ := os.Hostname(); h != "" { info.Name = h } }

	// CPU 型号
	if out, err := runPwsh("Get-CimInstance Win32_Processor | Select-Object -ExpandProperty Name"); err == nil {
		info.CPU = firstLine(out)
	}

	// 内存容量（总物理内存）
	if out, err := runPwsh("(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory"); err == nil {
		info.RAM = humanSize(parseInt64(firstLine(out)))
	}

	// 系统盘大小（系统卷）
	if out, err := runPwsh(`$sys=(Get-CimInstance Win32_OperatingSystem).SystemDrive; (Get-CimInstance Win32_LogicalDisk | Where-Object {$_.DeviceID -eq $sys}).Size`); err == nil {
		info.Disk = humanSize(parseInt64(firstLine(out)))
	}

	// 序列号（避免 wmic，使用 CIM）
	if out, err := runPwsh("(Get-CimInstance Win32_BIOS).SerialNumber"); err == nil {
		info.SN = strings.TrimSpace(firstLine(out))
	}

	// MAC 与 IP：选非回环、已启用适配器
	if out, err := runPwsh(`Get-NetIPConfiguration | Where-Object {$_.NetAdapter.Status -eq 'Up'} | Select-Object -First 1 -ExpandProperty NetAdapter | Select-Object -ExpandProperty MacAddress`); err == nil {
		info.MAC = strings.TrimSpace(firstLine(out))
	}
	if out, err := runPwsh(`(Get-NetIPConfiguration | Where-Object {$_.IPv4Address -ne $null})[0].IPv4Address.IPAddress`); err == nil {
		info.IP = strings.TrimSpace(firstLine(out))
	}

	if info.Name == "" && info.CPU == "" {
		return info, errors.New("未能成功采集关键字段")
	}
	return info, nil
}

func runPwsh(script string) (string, error) {
	// 优先使用 pwsh，其次 powershell
	cmd := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("powershell执行失败: %v, output=%s", err, string(out))
		}
	}
	return string(out), nil
}

func firstLine(s string) string {
	for _, ln := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" { return ln }
	}
	return ""
}

func humanSize(bytesCount int64) string {
	if bytesCount <= 0 { return "0B" }
	units := []string{"B","KB","MB","GB","TB"}
	var i int
	val := float64(bytesCount)
	for i < len(units)-1 && val >= 1024 {
		val /= 1024
		i++
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", val), "0"), ".") + units[i]
}

func parseInt64(s string) int64 {
	// 去除非数字字符
	re := regexp.MustCompile(`[^0-9]`)
	s = re.ReplaceAllString(s, "")
	var n int64
	for _, r := range s { n = n*10 + int64(r-'0') }
	return n
}


