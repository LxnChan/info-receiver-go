# 客户端自动更新功能

## 功能说明

客户端在完成信息上报后会自动检查更新，如果发现新版本会自动下载并替换当前程序。

## 更新流程

1. 信息上报完成后，客户端会请求指定的更新检查URL
2. 服务器返回JSON格式的更新信息
3. 客户端比较版本号，如果发现新版本则下载并替换

## 更新信息格式

更新检查URL应返回以下JSON格式：

```json
{
  "version": "1.0",
  "download_url": "https://example.com/downloads/client-linux-amd64",
  "description": "更新说明（可选）"
}
```

字段说明：
- `version`: 新版本号（支持 x.y.z 格式）
- `download_url`: 新版本下载链接
- `description`: 更新说明（可选）

## 编译时设置更新URL

### 方法1：使用编译脚本

```bash
# 设置更新URL并编译
./build.sh -u https://your-server.com/update.json

# 同时设置版本号和输出目录
./build.sh -u https://your-server.com/update.json -v 1.0 -o build
```

### 方法2：手动编译

```bash
# Linux版本
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.updateCheckURL=https://your-server.com/update.json" -o client-linux-amd64 .

# Windows版本
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.updateCheckURL=https://your-server.com/update.json" -o client-windows-amd64.exe .
```

## 版本比较规则

- 支持语义化版本号（x.y.z）
- 按数字大小比较，如 1.0.1 > 1.0.0
- 版本号长度不同时，缺失部分视为0

## 更新机制

### Linux/Unix系统
- 直接替换当前程序文件
- 下次运行时使用新版本

### Windows系统
- 创建批处理脚本延迟替换
- 避免文件被占用的问题

## 安全考虑

1. **HTTPS**: 建议使用HTTPS协议传输更新信息
2. **文件校验**: 可以在更新信息中添加文件校验码
3. **权限控制**: 确保更新URL的访问权限

## 示例

### 1. 创建更新服务器

```bash
# 创建更新信息文件
cat > /var/www/html/update.json << EOF
{
  "version": "1.0",
  "download_url": "https://your-server.com/downloads/client-linux-amd64",
  "description": "新版本包含重要安全更新"
}
EOF
```

### 2. 编译客户端

```bash
./build.sh -u https://your-server.com/update.json
```

### 3. 运行客户端

```bash
./client-linux-amd64 -s http://your-server:8080
```

客户端会在上报完成后自动检查更新。

## 故障排除

1. **更新检查失败**: 检查网络连接和URL是否正确
2. **下载失败**: 检查下载URL是否可访问
3. **替换失败**: 检查程序是否有写入权限

## 注意事项

- 更新失败不会影响主功能（信息上报）
- 更新过程是异步的，不会阻塞主流程
- 建议在测试环境先验证更新机制
