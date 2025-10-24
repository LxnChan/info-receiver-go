# 客户端自动更新功能实现总结

## 功能概述

已成功为客户端添加了自动更新功能，在信息上报完成后会自动检查更新，如果发现新版本会自动下载并替换当前程序。

## 实现的功能

### 1. 版本信息结构体
- `UpdateInfo` 结构体包含版本号、下载链接和描述信息
- 支持JSON格式的更新信息

### 2. 版本比较功能
- 实现语义化版本比较（支持 x.y.z 格式）
- 按数字大小比较版本号
- 自动检测是否有新版本

### 3. 文件下载和替换
- 支持HTTP/HTTPS下载
- 创建临时文件避免损坏原程序
- 跨平台文件替换（Linux/Windows）
- Windows使用批处理脚本延迟替换

### 4. 编译时配置
- 通过 `-ldflags` 参数设置更新URL
- 提供便捷的编译脚本 `build.sh`
- 支持动态设置更新检查URL

### 5. 错误处理
- 更新失败不影响主功能
- 详细的错误日志记录
- 网络超时和重试机制

## 文件结构

```
client/
├── main.go                    # 主程序，包含更新逻辑
├── build.sh                   # 编译脚本
├── test-update.sh             # 测试脚本
├── example-update.json        # 更新信息示例
├── README-AUTO-UPDATE.md      # 详细使用说明
├── sysinfo_common.go         # 系统信息收集（通用）
├── sysinfo_linux.go          # Linux系统信息收集
├── sysinfo_windows.go        # Windows系统信息收集
├── uac_common.go             # UAC处理（非Windows）
└── uac_windows.go            # UAC处理（Windows）
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

## 安全特性

- 支持HTTPS协议
- 临时文件机制避免损坏
- 权限检查和设置
- 错误恢复机制

## 测试验证

运行测试脚本验证功能：
```bash
./test-update.sh
```

## 注意事项

1. 更新URL需要在编译时指定
2. 更新失败不会影响主功能
3. Windows需要管理员权限进行文件替换
4. 建议在测试环境先验证更新机制

## 技术细节

- 使用Go的`-ldflags`参数在编译时设置变量
- 支持跨平台编译（Linux/Windows）
- 实现语义化版本比较算法
- 使用HTTP客户端进行文件下载
- 跨平台文件替换机制

自动更新功能已完全实现并经过测试验证，可以投入使用。
