//go:build windows

package main

import (
    "fmt"
    "os"
    "os/exec"
    "syscall"
)

// ensureAdmin 检测是否管理员；若不是，则请求UAC提升并以管理员重启自身
func ensureAdmin() {
    // 通过 whoami /groups 检测 Administrators 组
    cmd := exec.Command("whoami", "/groups")
    cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    out, err := cmd.CombinedOutput()
    if err == nil && containsAdmin(string(out)) {
        return
    }

    // 不是管理员，触发UAC
    exe, err := os.Executable()
    if err != nil { return }

    // 使用 powershell Start-Process -Verb RunAs 重新启动自身并传递参数
    args := append([]string{"-NoProfile", "-NonInteractive", "-Command"}, fmt.Sprintf("Start-Process -FilePath '%s' -ArgumentList '%s' -Verb RunAs", exe, escapeArgs(os.Args[1:])))
    _ = exec.Command("powershell", args...).Start()
    os.Exit(0)
}

func containsAdmin(s string) bool {
    // 文本包含 "S-1-5-32-544" 或 "BUILTIN\\Administrators" 认为有管理员组
    return contains(s, "S-1-5-32-544") || contains(s, "BUILTIN\\Administrators") || contains(s, "Administrators")
}

func contains(s, sub string) bool {
    return len(s) >= len(sub) && (func() bool { return stringIndex(s, sub) >= 0 })()
}

func stringIndex(s, sub string) int {
    // 简单包装避免引入 strings 额外依赖
    for i := 0; i+len(sub) <= len(s); i++ {
        if s[i:i+len(sub)] == sub {
            return i
        }
    }
    return -1
}

func escapeArgs(args []string) string {
    // 粗略拼接参数，按空格分隔并用双引号包裹
    if len(args) == 0 { return "" }
    out := ""
    for i, a := range args {
        if i > 0 { out += "," }
        out += fmt.Sprintf("\"%s\"", a)
    }
    return out
}


