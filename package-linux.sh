。/#!/bin/bash

# Linux x86_64 打包脚本
# 用于将项目打包迁移到Linux系统

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打包文件名
PACKAGE_NAME="exam-practice-linux-x86_64-$(date +%Y%m%d).tar.gz"
TEMP_DIR="linux-package-temp"

echo -e "${BLUE}开始打包项目到Linux x86_64系统...${NC}"

# 检查是否在项目根目录
if [ ! -f "package.json" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 创建临时目录
echo -e "${BLUE}创建临时目录...${NC}"
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

# 复制前端文件
echo -e "${BLUE}复制前端文件...${NC}"
mkdir -p "$TEMP_DIR/src"
cp -r src/* "$TEMP_DIR/src/"
cp package.json vite.config.js index.html "$TEMP_DIR/"
if [ -f "package-lock.json" ]; then
    cp package-lock.json "$TEMP_DIR/"
fi

# 复制后端文件 (只复制Go相关文件)
echo -e "${BLUE}复制后端Go文件...${NC}"
mkdir -p "$TEMP_DIR/go-server"
cp -r go-server/* "$TEMP_DIR/go-server/"

# 删除server目录（Node.js后台服务）
echo -e "${BLUE}排除Node.js后台服务文件...${NC}"
rm -rf "$TEMP_DIR/server"

# 复制配置文件
echo -e "${BLUE}复制配置文件...${NC}"
cp server-manager.sh "$TEMP_DIR/"
cp sample-questions.json "$TEMP_DIR/"

# 创建部署脚本
echo -e "${BLUE}创建部署脚本...${NC}"
cat > "$TEMP_DIR/deploy.sh" << 'EOF'
#!/bin/bash

# Linux部署脚本

set -e

echo "开始部署智能刷题平台..."

# 检查系统架构
ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
    echo "警告: 当前系统架构为 $ARCH，建议使用 x86_64 架构"
fi

# 检查Go
if ! command -v go &> /dev/null; then
    echo "错误: 未安装Go，请先安装Go语言环境"
    exit 1
fi

# 安装前端依赖并构建
echo "安装前端依赖..."
npm install

echo "构建前端..."
npm run build

# 编译Go后端
echo "编译Go后端..."
cd go-server
go mod tidy
go build -o exam-server main.go routes.go handlers.go wrong_questions.go exam_results.go
cd ..

# 设置执行权限
chmod +x server-manager.sh
chmod +x go-server/exam-server

echo "部署完成！"
echo ""
echo "使用方法:"
echo "1. 启动服务: ./server-manager.sh start"
echo "2. 停止服务: ./server-manager.sh stop"
echo "3. 查看状态: ./server-manager.sh status"
echo ""
echo "服务访问地址: http://localhost:3005"
echo "API接口地址: http://localhost:3005/api"
EOF

chmod +x "$TEMP_DIR/deploy.sh"

# 创建README文件
echo -e "${BLUE}创建说明文档...${NC}"
cat > "$TEMP_DIR/README-LINUX.md" << 'EOF'
# 智能刷题平台 - Linux x86_64 部署指南

## 系统要求
- Linux x86_64 系统
- Go 1.18+
- MySQL 8.0+

## 快速开始

### 1. 环境准备
```bash
# 安装Go
wget https://golang.org/dl/go1.20.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.20.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 安装MySQL
sudo apt-get install mysql-server
```

### 2. 数据库配置
```sql
CREATE DATABASE exam_practice;
CREATE USER 'exam_user'@'localhost' IDENTIFIED BY 'exam_password';
GRANT ALL PRIVILEGES ON exam_practice.* TO 'exam_user'@'localhost';
FLUSH PRIVILEGES;
```

### 3. 项目部署
```bash
# 解压包
tar -xzf exam-practice-linux-x86_64-*.tar.gz
cd exam-practice-linux-x86_64-*

# 运行部署脚本
./deploy.sh

# 配置数据库连接 (编辑 go-server/main.go 中的数据库配置)
# 启动服务
./server-manager.sh start
```

## 文件结构
```
./
├── src/                # 前端Vue.js源码
├── dist/               # 前端构建输出
├── go-server/          # 后端Go源码
├── server-manager.sh   # 服务管理脚本
├── deploy.sh          # 部署脚本
├── package.json        # 前端依赖配置
└── README-LINUX.md    # 本文档
```

## 服务管理
使用 `server-manager.sh` 脚本管理服务：

```bash
# 启动所有服务
./server-manager.sh start

# 停止所有服务  
./server-manager.sh stop

# 重启所有服务
./server-manager.sh restart

# 查看服务状态
./server-manager.sh status

# 查看日志
./server-manager.sh logs
```

## 端口配置
- 前端服务: 由Go后端提供静态文件服务
- 后端API: 3005

## 故障排除

### 端口占用
如果端口被占用，可以修改配置：
- 前端构建配置: 编辑 `vite.config.js`
- 后端端口: 编辑 `go-server/main.go`

### 数据库连接失败
检查 `go-server/main.go` 中的数据库配置：
```go
dsn := "exam_user:exam_password@tcp(localhost:3306)/exam_practice?charset=utf8mb4&parseTime=True&loc=Local"
```

### 权限问题
确保脚本有执行权限：
```bash
chmod +x *.sh
chmod +x go-server/exam-server
```

## 技术支持
如有问题请联系: 867368106@QQ.com
EOF

# 创建打包
echo -e "${BLUE}创建压缩包...${NC}"
tar -czf "$PACKAGE_NAME" -C "$TEMP_DIR" .

# 清理临时文件
echo -e "${BLUE}清理临时文件...${NC}"
rm -rf "$TEMP_DIR"

echo -e "${GREEN}打包完成！${NC}"
echo -e "${GREEN}打包文件: $PACKAGE_NAME${NC}"
echo ""
echo -e "${BLUE}部署到Linux系统的步骤:${NC}"
echo "1. 将 $PACKAGE_NAME 复制到Linux系统"
echo "2. 解压: tar -xzf $PACKAGE_NAME"
echo "3. 进入解压后的目录"
echo "4. 运行: ./deploy.sh"
echo "5. 配置数据库连接"
echo "6. 启动: ./server-manager.sh start"
echo ""
echo -e "${YELLOW}注意: 请确保Linux系统已安装Go和MySQL${NC}"