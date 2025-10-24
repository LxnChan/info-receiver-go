## 文件结构

```
client/
├── main.go                    # 主程序，包含更新逻辑
├── build.sh                   # 编译脚本
├── example-update.json        # 更新信息示例
├── README-AUTO-UPDATE.md      # 详细使用说明
├── sysinfo_common.go          # 系统信息收集（通用）
├── sysinfo_linux.go           # Linux系统信息收集
├── sysinfo_windows.go         # Windows系统信息收集
├── uac_common.go              # UAC处理（非Windows）
└── uac_windows.go             # UAC处理（Windows）
```

## 使用方法

### 1. 编译客户端
```bash
# 设置更新URL并编译
./build.sh -u https://your-server.com/update.json

# 同时设置版本号和输出目录
./build.sh -u https://your-server.com/update.json -v 1.0 -o build
```

### 2. 更新信息格式
服务器需要提供JSON格式的更新信息：
```json
{
  "version": "1.0",
  "download_url": "https://example.com/downloads/client-linux-amd64",
  "description": "更新说明（可选）"
}
```

### 3. 运行客户端
```bash
# Linux
./client-linux-amd64 -s http://server:8080

# Windows
client-windows-amd64.exe -s http://server:8080
```

## 更新流程

1. **信息上报**: 客户端完成系统信息上报
2. **检查更新**: 自动请求更新检查URL
3. **版本比较**: 比较服务器版本与当前版本
4. **下载更新**: 如果发现新版本，下载新程序
5. **替换程序**: 替换当前程序文件
6. **下次运行**: 下次运行时使用新版本
