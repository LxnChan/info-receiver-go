package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
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
	Network *string `json:"Network"` // 始终返回null
}

const clientVersion = "0.9"

func main() {
	server := flag.String("s", "", "服务器地址，例如 http://host:8080 或完整接口 http://host:8080/api/client")
	comment := flag.String("c", "", "备注 comment，可为空")
	timeout := flag.Duration("t", 10*time.Second, "HTTP 超时时间")
	flag.Parse()

	if *server == "" {
		fmt.Fprintln(os.Stderr, "必须通过 -s 指定服务器地址")
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

	// 组装负载
	var nullStr *string = nil
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
		Network: nullStr,
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
}


