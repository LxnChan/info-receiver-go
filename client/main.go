package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Payload 与服务端期望的字段一致
type Payload struct {
	Name    string  `json:"Name"`
	CPU     string  `json:"CPU"`
	RAM     string  `json:"RAM"`
	Disk    string  `json:"Disk"`
	SN      string  `json:"SN"`
	MAC     string  `json:"MAC"`
	IP      string  `json:"IP"`
	UpVer   string  `json:"up_ver"`
	Comment string  `json:"comment"`
    Network *string `json:"Network"`
}

const clientVersion = "1.3"

// UpdateInfo 更新信息结构体
type UpdateInfo struct {
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Downloads   map[string]DownloadInfo `json:"downloads"`
}

// DownloadInfo 下载信息结构体
type DownloadInfo struct {
	URL string `json:"url"`
}

// 编译时指定的更新检查URL，可以通过 -ldflags 参数动态设置
var updateCheckURL = "https://example.com/update.json"

func main() {
    // Windows: 自动请求管理员（UAC）后再继续；其他平台无操作
	// 暂时先不实装
    // ensureAdmin()
	server := flag.String("s", "", "服务器地址，例如 http://host:8080 或完整接口 http://host:8080/api/client")
	comment := flag.String("c", "", "备注 comment，可为空")
	timeout := flag.Duration("t", 10*time.Second, "HTTP 超时时间")
	flag.Parse()

	if *server == "" {
		fmt.Fprintln(os.Stderr, "无服务器地址，请通过-s参数指定")
		os.Exit(2)
	}

	endpoint := strings.TrimSpace(*server)
	if !strings.HasSuffix(strings.ToLower(endpoint), "/api/client") {
		endpoint = strings.TrimRight(endpoint, "/") + "/api/client"
	}

	info, err := CollectSystemInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "采集信息失败: %v\n", err)
		os.Exit(1)
	}

	// 统一规范 MAC 格式为 xxxx.xxxx.xxxx
	info.MAC = formatMacXXXX(info.MAC)

	// 组装负载
    // Network：根据采集结果设置，若为空字符串则仍上报为 null
    var networkPtr *string
    if strings.TrimSpace(info.Network) != "" {
        v := info.Network
        networkPtr = &v
    } else {
        networkPtr = nil
    }
	p := Payload{
		Name:    info.Name,
		CPU:     info.CPU,
		RAM:     info.RAM,
		Disk:    info.Disk,
		SN:      info.SN,
		MAC:     info.MAC,
		IP:      info.IP,
		UpVer:   clientVersion,
		Comment: *comment,
        Network: networkPtr,
	}

	body, err := json.Marshal(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "编码JSON失败: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: *timeout}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建请求失败: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		fmt.Fprintf(os.Stderr, "服务器返回错误状态: %s\n", resp.Status)
		os.Exit(1)
	}

	fmt.Println("上报完成")
	
	// 信息上报完成后，检查更新
	if err := checkAndUpdate(); err != nil {
		fmt.Fprintf(os.Stderr, "更新检查失败: %v\n", err)
		// 更新失败不影响主流程，只记录错误
	}
}

// formatMacXXXX 将任意常见 MAC 字符串（aa:bb:cc:dd:ee:ff / aa-bb-... / aabb.ccdd.eeff）
// 规范化为小写 "xxxx.xxxx.xxxx" 形式；若无法解析则返回原值。
func formatMacXXXX(mac string) string {
	if mac == "" { return mac }
	s := strings.ToLower(mac)
	// 移除分隔符
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")
	// 保留前12个十六进制字符
	var hexOnly strings.Builder
	for _, r := range s {
        if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
            hexOnly.WriteRune(r)
        }
        if hexOnly.Len() == 12 { break }
    }
    cleaned := hexOnly.String()
    if len(cleaned) != 12 { return mac }
    return cleaned[0:4] + "." + cleaned[4:8] + "." + cleaned[8:12]
}

// checkAndUpdate 检查并执行自动更新
func checkAndUpdate() error {
	// 如果更新URL为空或默认值，跳过更新检查
	if updateCheckURL == "" || updateCheckURL == "https://raw.githubusercontent.com/LxnChan/info-receiver-go/refs/heads/main/update.json" {
		return nil
	}
	
	fmt.Println("正在检查更新...")
	
	// 获取更新信息
	updateInfo, err := fetchUpdateInfo(updateCheckURL)
	if err != nil {
		return fmt.Errorf("获取更新信息失败: %v", err)
	}
	
	// 比较版本
	if !isNewerVersion(updateInfo.Version, clientVersion) {
		fmt.Println("当前已是最新版本")
		return nil
	}
	
	fmt.Printf("发现新版本 %s，正在下载更新...\n", updateInfo.Version)
	
	// 获取当前平台对应的下载链接
	downloadURL, err := getDownloadURL(updateInfo)
	if err != nil {
		return fmt.Errorf("获取下载链接失败: %v", err)
	}
	
	// 下载并替换
	if err := downloadAndReplace(downloadURL); err != nil {
		return fmt.Errorf("下载更新失败: %v", err)
	}
	
	fmt.Println("更新完成，程序将在下次运行时使用新版本")
	return nil
}

// fetchUpdateInfo 从URL获取更新信息
func fetchUpdateInfo(url string) (*UpdateInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var updateInfo UpdateInfo
	if err := json.Unmarshal(body, &updateInfo); err != nil {
		return nil, err
	}
	
	if updateInfo.Version == "" || updateInfo.Downloads == nil {
		return nil, fmt.Errorf("更新信息格式错误")
	}
	
	return &updateInfo, nil
}

// isNewerVersion 比较版本号，返回true表示newVersion比currentVersion更新
func isNewerVersion(newVersion, currentVersion string) bool {
	// 简单的版本比较，支持 x.y.z 格式
	newParts := strings.Split(newVersion, ".")
	currentParts := strings.Split(currentVersion, ".")
	
	maxLen := len(newParts)
	if len(currentParts) > maxLen {
		maxLen = len(currentParts)
	}
	
	for i := 0; i < maxLen; i++ {
		var newNum, currentNum int
		
		if i < len(newParts) {
			fmt.Sscanf(newParts[i], "%d", &newNum)
		}
		if i < len(currentParts) {
			fmt.Sscanf(currentParts[i], "%d", &currentNum)
		}
		
		if newNum > currentNum {
			return true
		} else if newNum < currentNum {
			return false
		}
	}
	
	return false // 版本相同
}

// getDownloadURL 根据当前平台获取对应的下载链接
func getDownloadURL(updateInfo *UpdateInfo) (string, error) {
	// 构建平台标识符
	platform := runtime.GOOS + "-" + runtime.GOARCH
	
	// 查找对应的下载链接
	if downloadInfo, exists := updateInfo.Downloads[platform]; exists {
		return downloadInfo.URL, nil
	}
	
	// 如果没找到精确匹配，尝试查找通用版本
	if downloadInfo, exists := updateInfo.Downloads[runtime.GOOS]; exists {
		return downloadInfo.URL, nil
	}
	
	return "", fmt.Errorf("未找到平台 %s 对应的下载链接", platform)
}

// downloadAndReplace 下载新版本并替换当前程序
func downloadAndReplace(downloadURL string) error {
	// 获取当前程序路径
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %v", err)
	}
	
	// 创建临时文件
	tempFile := currentPath + ".tmp"
	
	// 下载文件
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，HTTP状态码: %d", resp.StatusCode)
	}
	
	// 创建临时文件
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer file.Close()
	
	// 复制数据
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("保存文件失败: %v", err)
	}
	file.Close()
	
	// 设置执行权限
	if err := os.Chmod(tempFile, 0755); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("设置执行权限失败: %v", err)
	}
	
	// 在Windows上，需要先删除原文件再重命名
	if runtime.GOOS == "windows" {
		// 创建批处理文件来执行替换
		batchContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
del "%s"
move "%s" "%s"
del "%~f0"
`, currentPath, tempFile, currentPath)
		
		batchFile := filepath.Join(filepath.Dir(currentPath), "update.bat")
		if err := os.WriteFile(batchFile, []byte(batchContent), 0644); err != nil {
			os.Remove(tempFile)
			return fmt.Errorf("创建更新脚本失败: %v", err)
		}
		
		// 执行批处理文件
		cmd := exec.Command("cmd", "/C", batchFile)
		cmd.Start() // 异步执行，不等待完成
		
	} else {
		// Unix系统直接替换
		if err := os.Rename(tempFile, currentPath); err != nil {
			os.Remove(tempFile)
			return fmt.Errorf("替换文件失败: %v", err)
		}
	}
	
	return nil
}


