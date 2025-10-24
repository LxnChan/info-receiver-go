#!/bin/bash

# 客户端编译脚本
# 支持设置更新检查URL

# 默认参数
UPDATE_URL="https://raw.githubusercontent.com/LxnChan/info-receiver-go/refs/heads/main/update.json"
OUTPUT_DIR="dist"
VERSION="1.3"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--update-url)
            UPDATE_URL="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  -u, --update-url URL    设置更新检查URL"
            echo "  -o, --output DIR        设置输出目录 (默认: dist)"
            echo "  -v, --version VERSION   设置版本号 (默认: 0.9)"
            echo "  -h, --help              显示此帮助信息"
            echo ""
            echo "示例:"
            echo "  $0 -u https://raw.githubusercontent.com/LxnChan/info-receiver-go/refs/heads/main/update.json"
            echo "  $0 -u https://raw.githubusercontent.com/LxnChan/info-receiver-go/refs/heads/main/update.json -o build -v 1.0"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 -h 或 --help 查看帮助"
            exit 1
            ;;
    esac
done

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

echo "开始编译客户端..."
echo "版本: $VERSION"
sed -i "s/const clientVersion = \"0\.9\"/const clientVersion = \"${VERSION}\"/g" main.go
echo "输出目录: $OUTPUT_DIR"

# 编译函数
compile_version() {
    local os=$1
    local arch=$2
    local ext=$3
    
    echo "编译 $os-$arch 版本..."
    if [ -n "$UPDATE_URL" ]; then
        GOOS=$os GOARCH=$arch go build -ldflags "-X main.updateCheckURL=$UPDATE_URL" -o "$OUTPUT_DIR/client-$os-$arch$ext" .
    else
        GOOS=$os GOARCH=$arch go build -o "$OUTPUT_DIR/client-$os-$arch$ext" .
    fi
    
    if [ $? -eq 0 ]; then
        echo "✓ $os-$arch 版本编译成功: $OUTPUT_DIR/client-$os-$arch$ext"
    else
        echo "✗ $os-$arch 版本编译失败"
        return 1
    fi
}

# 编译所有版本
if [ -n "$UPDATE_URL" ]; then
    echo "更新URL: $UPDATE_URL"
else
    echo "更新URL: 未设置 (将跳过更新检查)"
fi

# Linux版本
compile_version "linux" "amd64" "" || exit 1
compile_version "linux" "arm64" "" || exit 1

# Windows版本
compile_version "windows" "amd64" ".exe" || exit 1
compile_version "windows" "arm64" ".exe" || exit 1

echo ""
echo "编译完成！"
echo "文件位置:"
echo "  Linux AMD64:   $OUTPUT_DIR/client-linux-amd64"
echo "  Linux ARM64:   $OUTPUT_DIR/client-linux-arm64"
echo "  Windows AMD64: $OUTPUT_DIR/client-windows-amd64.exe"
echo "  Windows ARM64: $OUTPUT_DIR/client-windows-arm64.exe"
echo ""
echo "使用方法:"
echo "  Linux:   ./client-linux-amd64 -s http://server:8080"
echo "  Windows: client-windows-amd64.exe -s http://server:8080"
