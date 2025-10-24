# GoUP 服务端程序

这是一个基于Go语言的服务端程序，用于接收客户端POST的JSON数据并存储到MySQL数据库中。

## 功能特性

- 持续运行的HTTP服务器
- 接收客户端POST的JSON数据
- 自动解析JSON并存储到MySQL数据库
- **智能重复数据处理**：基于MAC地址或SN自动更新现有记录
- 命令行参数配置
- 可选日志输出
- 健康检查端点
- 完整的错误处理和日志记录

## 环境要求

- Go 1.21 或更高版本
- MySQL 5.7 或更高版本

## 安装和运行

### 1. 克隆项目并安装依赖

```bash
go mod tidy
```

### 2. 配置数据库

确保MySQL数据库正在运行，并创建数据库：

```sql
CREATE DATABASE goup;
```

### 3. 运行程序

程序现在使用命令行参数进行配置：

```bash
# 基本运行（必需指定DSN）
go run main.go -dsn "root:password@tcp(localhost:3306)/goup"

# 指定端口
go run main.go -dsn "root:password@tcp(localhost:3306)/goup" -port 8080

# 启用日志输出到指定目录
go run main.go -dsn "root:password@tcp(localhost:3306)/goup" -log-dir ./logs

# 完整参数示例
go run main.go -dsn "root:password@tcp(localhost:3306)/goup" -port 8080 -log-dir ./logs
```

### 4. 命令行参数说明

- `-dsn`: 数据库连接字符串（必需）
  - 格式：`user:password@tcp(host:port)/dbname`
  - 示例：`root:password@tcp(localhost:3306)/goup`
- `-port`: 服务器端口（可选，默认8080）
- `-log-dir`: 日志目录（可选，不指定则不输出日志文件）

程序启动后，您将看到类似以下的输出：

```
数据库连接成功，数据表已创建
服务器启动在端口 8080
健康检查: http://localhost:8080/health
客户端数据接口: http://localhost:8080/api/client
```

如果指定了 `-log-dir` 参数，还会显示：
```
日志将输出到: ./logs/goup-server.log
```

## API 接口

### 健康检查

**GET** `/health`

返回服务器状态信息。

**响应示例：**
```json
{
  "status": "ok",
  "message": "服务运行正常"
}
```

### 客户端数据接口

**POST** `/api/client`

接收客户端发送的JSON数据。

**请求体示例：**
```json
{
  "Name": "DESKTOP-4JKIOMP",
  "CPU": "Intel Core i9-9900K",
  "RAM": "16GB",
  "Disk": "936GB",
  "SN": "J7K9NOLK",
  "MAC": "a5e9.e487.71f2",
  "IP": "192.168.233.233",
  "up_ver": "0.9",
  "comment": "Lily's Notebook",
  "Network": "WIFI"
}
```

**注意：** 所有字段都是可选的，客户端可以只发送部分字段。

**成功响应（新记录）：**
```json
{
  "status": "success",
  "message": "数据已成功保存"
}
```

**成功响应（更新记录）：**
```json
{
  "status": "success",
  "message": "数据已成功更新"
}
```

## 重复数据处理

程序具有智能的重复数据处理功能，并记录时间与变更历史：

- **检查条件**：如果接收到的数据中MAC地址或SN与数据库中现有记录相同，则更新该记录
- **更新策略**：更新所有字段（Name、CPU、RAM、Disk、IP、up_ver、comment、Network等），自动刷新 `updated_at`
- **时间字段**：`created_at` 为创建时间，`updated_at` 为最后修改时间
- **日志记录**：会明确记录是"保存新数据"还是"更新现有数据"
- **响应消息**：API响应会明确告知是保存还是更新操作
- **变更表**：每次插入或更新都会在 `client_changes` 中记录一条变更，包含操作类型与快照

这样可以避免同一设备产生多条记录，保持数据的唯一性和最新性。

## 数据库表结构

程序会自动创建 `client_info` 表与 `client_changes` 变更记录表，结构如下：

```sql
CREATE TABLE client_info (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255),
    cpu VARCHAR(255),
    ram VARCHAR(255),
    disk VARCHAR(255),
    sn VARCHAR(255),
    mac VARCHAR(255),
    ip VARCHAR(255),
    up_ver VARCHAR(255),
    comment TEXT,
    network VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_mac (mac)
);
-- 变更记录表
CREATE TABLE client_changes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    client_id INT NOT NULL,
    change_type VARCHAR(16) NOT NULL, -- insert/update
    name VARCHAR(255),
    cpu VARCHAR(255),
    ram VARCHAR(255),
    disk VARCHAR(255),
    sn VARCHAR(255),
    mac VARCHAR(255),
    ip VARCHAR(255),
    up_ver VARCHAR(255),
    comment TEXT,
    network VARCHAR(255),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_client_id (client_id),
    INDEX idx_change_mac (mac)
);
```

**注意：** 程序会自动在MAC地址字段上创建索引以提高查询性能。

## 测试

### 使用curl测试

```bash
# 健康检查
curl http://localhost:8080/health

# 发送完整客户端数据
curl -X POST http://localhost:8080/api/client \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "DESKTOP-4JKIOMP",
    "CPU": "Intel Core i9-9900K",
    "RAM": "16GB",
    "Disk": "936GB",
    "SN": "J7K9NOLK",
    "MAC": "a5e9.e487.71f2",
    "IP": "192.168.233.233",
    "up_ver": "0.9",
    "comment": "Lily'\''s Notebook",
    "Network": "WIFI"
  }'

# 发送部分数据（所有字段都是可选的）
curl -X POST http://localhost:8080/api/client \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "My Computer",
    "IP": "192.168.1.100",
    "Network": "WIFI"
  }'
```

## 日志

程序会输出详细的日志信息，包括：
- 数据库连接状态
- 接收到的客户端数据
- 操作类型（保存新数据或更新现有数据）
- 错误信息

**日志示例：**
```
成功保存新客户端数据: DESKTOP-4JKIOMP (192.168.233.233) - MAC: A5:E9:E4:87:71:F2, SN: J7K9NOLK
成功更新客户端数据: DESKTOP-4JKIOMP (192.168.233.234) - MAC: A5:E9:E4:87:71:F2, SN: J7K9NOLK
```

## 客户端

客户端用于在主机上采集信息并上报到本服务的 `/api/client` 接口，支持 Linux 与 Windows（Windows 仅支持Windows 10及以上版本）。

### 采集内容

- Name: 计算机名
- CPU: 处理器型号
- RAM: 物理内存总量（人类可读）
- Disk: 系统盘大小（Linux 根分区，Windows 系统卷）
- SN: 系统序列号
- MAC: 所选网卡的 MAC，统一规范为小写 `xxxx.xxxx.xxxx`
- IP: 所选网卡的 IPv4 地址
- up_ver: 客户端版本
- comment: 命令行传入的备注
- Network: 根据所选网卡判断，`WIFI` 或 `ETHERNET`，无法判定为 `null`

选择网卡规则（去除虚拟网卡影响）：
- 仅选择“物理网卡”且“连接状态为 Up”的第一块网卡
- Linux：排除 docker/veth/br-/vmnet/vboxnet/vmware/virbr/zerotier/tailscale/wg/tun/tap/lo 等；要求 `operstate=up` 且存在 `/sys/class/net/<iface>/device`。无线判断基于 `/sys/class/net/<iface>/wireless`。
- Windows：`Get-NetAdapter` 过滤 `Status='Up'`、`Virtual=$false`、`HardwareInterface=$true`；网络类型基于 `NdisPhysicalMedium` 或描述兜底判断。

### 使用方法

```bash
# Linux 运行
./client-linux -s http://server:8080 -c "This is my host"

# Windows 运行（可在 Linux 交叉编译）
./client-windows.exe -s http://server:8080 -c "Office PC"

# 也可直接指定完整接口
./client-linux -s http://server:8080/api/client -c "lab"
```

参数说明：
- `-s` 服务器地址（必填）。可为 `http://host:port` 或完整接口 `http://host:port/api/client`。
- `-c` 备注 comment（可选）。
- `-t` HTTP 超时时间（可选，默认 10s）。

请求示例（客户端上报实际 JSON）：
```json
{
  "Name": "DESKTOP-4JKIOMP",
  "CPU": "Intel Core i9-9900K",
  "RAM": "16GB",
  "Disk": "936GB",
  "SN": "J7K9NOLK",
  "MAC": "a5e9.e487.71f2",
  "IP": "192.168.233.233",
  "up_ver": "0.9",
  "comment": "This is my host",
  "Network": "ETHERNET"
}
```

## 服务端部署

### 编译为可执行文件

```bash
go build -o goup-server main.go
```

运行编译后的程序：

```bash
# 基本运行
./goup-server -dsn "root:password@tcp(localhost:3306)/goup"

# 启用日志
./goup-server -dsn "root:password@tcp(localhost:3306)/goup" -log-dir ./logs
```

## 故障排除

1. **数据库连接失败**：检查MySQL服务是否运行，DSN参数是否正确
2. **端口被占用**：使用 `-port` 参数指定其他端口
3. **JSON解析失败**：确保发送的JSON格式正确，字段名称匹配
4. **日志目录创建失败**：确保有权限在指定目录创建日志文件
5. **缺少DSN参数**：必须使用 `-dsn` 参数指定数据库连接字符串
