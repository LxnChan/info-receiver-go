//go:build linux

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func CollectSystemInfo() (SysInfo, error) {
	var info SysInfo

	// 主机名
	if h, err := os.Hostname(); err == nil {
		info.Name = h
	}

	// CPU 型号
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.ToLower(line), "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 { info.CPU = strings.TrimSpace(parts[1]) }
				break
			}
		}
	}

	// 内存容量（总量，四舍五入到MB/GB字符串）
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var kB int64
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if v, err := strconv.ParseInt(fields[1], 10, 64); err == nil { kB = v }
				}
				break
			}
		}
		if kB > 0 { info.RAM = humanSize(int64(kB) * 1024) }
	}

	// 磁盘（根分区大小）
	if out, err := exec.Command("df", "-k", "/").Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		// 跳过表头
		if scanner.Scan() {}
		if scanner.Scan() {
			line := scanner.Text()
			fs := strings.Fields(line)
			if len(fs) >= 2 {
				if blocks, err := strconv.ParseInt(fs[1], 10, 64); err == nil {
					info.Disk = humanSize(blocks * 1024) // 1K-blocks
				}
			}
		}
	}

	// 序列号（尝试从 /sys；需要root可能更稳定）
	if data, err := os.ReadFile("/sys/class/dmi/id/product_serial"); err == nil {
		info.SN = strings.TrimSpace(string(data))
	} else if out, err := exec.Command("dmidecode", "-s", "system-serial-number").Output(); err == nil {
		info.SN = strings.TrimSpace(string(out))
	}

	// MAC 与 IP：取一个非回环、非虚拟接口
	ifaces, _ := net.Interfaces()
	for _, nic := range ifaces {
		name := strings.ToLower(nic.Name)
		// 排除回环与常见虚拟接口
		if nic.Flags&net.FlagLoopback != 0 { continue }
		if isVirtualIface(name) { continue }
		// 要求内核视为 up（链路up）
		if nic.Flags&net.FlagUp == 0 { continue }
		// 要求为物理接口：存在 /sys/class/net/<iface>/device
		if !exists("/sys/class/net/" + name + "/device") { continue }
		// 要求 operstate 为 up
		if state, err := os.ReadFile("/sys/class/net/" + name + "/operstate"); err == nil {
			if strings.TrimSpace(string(state)) != "up" { continue }
		}

		mac := nic.HardwareAddr.String()
		if mac != "" && mac != "00:00:00:00:00:00" { info.MAC = mac }
		addrs, _ := nic.Addrs()
		for _, a := range addrs {
			ip, _, _ := net.ParseCIDR(a.String())
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() { continue }
			if ip.To4() != nil { info.IP = ip.String(); break }
		}
		if info.IP != "" && info.MAC != "" { break }
	}

	if info.Name == "" && info.CPU == "" {
		return info, errors.New("未能成功采集关键字段")
	}
	return info, nil
}

func isVirtualIface(name string) bool {
    // 常见虚拟/隧道接口前缀
    patterns := []string{
        "docker", "veth", "br-", "vmnet", "vboxnet", "vbox", "vmware", "virbr",
        "zt", "zerotier", "tailscale", "ts", "wg", "tun", "tap", "lo", "vcan",
    }
    for _, p := range patterns { if strings.HasPrefix(name, p) { return true } }
    return false
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
	// 去除多余小数
	re := regexp.MustCompile(`\.?0+$`)
	return re.ReplaceAllString(fmt.Sprintf("%.1f%s", val, units[i]), "")
}

func exists(path string) bool {
    if _, err := os.Stat(path); err == nil { return true }
    return false
}


