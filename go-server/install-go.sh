#!/bin/bash

# Go 环境安装脚本 (macOS)

echo "正在安装 Go 环境..."

# 检查系统架构
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    GO_ARCH="amd64"
elif [ "$ARCH" = "arm64" ]; then
    GO_ARCH="arm64"
else
    echo "不支持的系统架构: $ARCH"
    exit 1
fi

# Go 版本
GO_VERSION="1.21.5"
GO_PACKAGE="go${GO_VERSION}.darwin-${GO_ARCH}.tar.gz"
GO_URL="https://golang.org/dl/${GO_PACKAGE}"

echo "系统架构: $ARCH"
echo "Go 版本: $GO_VERSION"
echo "下载地址: $GO_URL"

# 创建临时目录
TEMP_DIR="/tmp/go-install"
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

# 下载 Go
echo "正在下载 Go..."
if command -v curl &> /dev/null; then
    curl -L -o "$GO_PACKAGE" "$GO_URL"
elif command -v wget &> /dev/null; then
    wget -O "$GO_PACKAGE" "$GO_URL"
else
    echo "错误: 需要 curl 或 wget 来下载文件"
    exit 1
fi

# 检查下载是否成功
if [ ! -f "$GO_PACKAGE" ]; then
    echo "错误: 下载失败"
    exit 1
fi

# 删除旧的 Go 安装（如果存在）
if [ -d "/usr/local/go" ]; then
    echo "正在删除旧的 Go 安装..."
    sudo rm -rf /usr/local/go
fi

# 解压安装
echo "正在安装 Go..."
sudo tar -C /usr/local -xzf "$GO_PACKAGE"

# 设置环境变量
echo "正在配置环境变量..."

# 检查并更新 shell 配置文件
SHELL_CONFIG=""
if [ -f "$HOME/.zshrc" ]; then
    SHELL_CONFIG="$HOME/.zshrc"
elif [ -f "$HOME/.bash_profile" ]; then
    SHELL_CONFIG="$HOME/.bash_profile"
elif [ -f "$HOME/.bashrc" ]; then
    SHELL_CONFIG="$HOME/.bashrc"
else
    SHELL_CONFIG="$HOME/.zshrc"
    touch "$SHELL_CONFIG"
fi

# 添加 Go 环境变量
if ! grep -q "/usr/local/go/bin" "$SHELL_CONFIG"; then
    echo "" >> "$SHELL_CONFIG"
    echo "# Go 环境变量" >> "$SHELL_CONFIG"
    echo "export PATH=\$PATH:/usr/local/go/bin" >> "$SHELL_CONFIG"
    echo "export GOPATH=\$HOME/go" >> "$SHELL_CONFIG"
    echo "export GOBIN=\$GOPATH/bin" >> "$SHELL_CONFIG"
fi

# 创建 GOPATH 目录
mkdir -p "$HOME/go/bin"
mkdir -p "$HOME/go/src"
mkdir -p "$HOME/go/pkg"

# 清理临时文件
cd /
rm -rf "$TEMP_DIR"

echo "----------------------------------------"
echo "Go 安装完成！"
echo ""
echo "请运行以下命令重新加载环境变量："
echo "source $SHELL_CONFIG"
echo ""
echo "或者重新打开终端窗口。"
echo ""
echo "然后运行以下命令验证安装："
echo "go version"
echo ""
echo "安装位置: /usr/local/go"
echo "GOPATH: $HOME/go"
echo "配置文件: $SHELL_CONFIG"
echo "----------------------------------------"