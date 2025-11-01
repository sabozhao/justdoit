#!/bin/bash

# Go 后端服务器启动脚本

echo "正在启动 Go 后端服务器..."

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "错误: Go 未安装"
    echo "请先安装 Go: https://golang.org/dl/"
    exit 1
fi

# 检查 Go 版本
GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "警告: 建议使用 Go $REQUIRED_VERSION 或更高版本，当前版本: $GO_VERSION"
fi

# 进入项目目录
cd "$(dirname "$0")"

# 下载依赖
echo "正在下载依赖..."
go mod tidy

# 启动服务器
echo "正在启动服务器..."
echo "服务器地址: http://localhost:3004"
echo "按 Ctrl+C 停止服务器"
echo "----------------------------------------"

go run *.go