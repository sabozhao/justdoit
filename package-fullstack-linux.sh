#!/bin/bash

# Linux x86_64 全栈打包脚本
# 打包前端源码 + 后端二进制，支持前后端分离部署

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
PACKAGE_NAME="exam-fullstack-linux-x64-$(date +%Y%m%d-%H%M%S)"
TEMP_DIR="build-temp"
DIST_DIR="$PACKAGE_NAME"

# 服务器配置
AUTO_UPLOAD=true
SERVER_HOST="119.91.68.147"
SERVER_USER="root"
SERVER_PASSWORD="Sabo2018！"
SERVER_PORT="22"
SERVER_PATH="/tmp"

echo -e "${BLUE}=== 智能刷题平台全栈 Linux x86_64 打包工具 ===${NC}"
echo -e "${BLUE}开始打包项目（前后端分离模式）...${NC}"

# 检查环境
echo -e "${BLUE}检查构建环境...${NC}"

# 检查 Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go，请先安装 Go 语言环境${NC}"
    exit 1
fi

# 检查项目文件
if [ ! -f "package.json" ] || [ ! -d "go-server" ] || [ ! -d "src" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 清理旧的构建文件
echo -e "${BLUE}清理旧的构建文件...${NC}"
rm -rf "$TEMP_DIR" "$DIST_DIR" "${PACKAGE_NAME}.tar.gz"

# 创建构建目录
mkdir -p "$TEMP_DIR"
mkdir -p "$DIST_DIR"

echo -e "${BLUE}复制前端源码...${NC}"
# 复制前端源码和配置
cp -r src "$DIST_DIR/"
cp package.json package-lock.json vite.config.js index.html "$DIST_DIR/"

echo -e "${BLUE}交叉编译 Go 后端到 Linux x86_64...${NC}"
cd go-server

# 设置 Go 交叉编译环境变量
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# 编译 Go 程序
go mod tidy
go build -ldflags="-s -w" -o ../exam-server-linux-x64 .

cd ..

# 检查编译结果
if [ ! -f "exam-server-linux-x64" ]; then
    echo -e "${RED}错误: Go 后端编译失败${NC}"
    exit 1
fi

echo -e "${GREEN}Go 后端编译完成${NC}"

echo -e "${BLUE}打包文件...${NC}"

# 复制 Go 二进制文件
cp exam-server-linux-x64 "$DIST_DIR/"

# 复制 Go 源码（用于调试）
mkdir -p "$DIST_DIR/go-server"
cp go-server/*.go "$DIST_DIR/go-server/"
cp go-server/go.mod go-server/go.sum "$DIST_DIR/go-server/"

# 复制数据库文件
if [ -f "go-server/exam.db" ]; then
    cp go-server/exam.db "$DIST_DIR/"
fi

# 复制配置文件
if [ -f "go-server/.env" ]; then
    cp go-server/.env "$DIST_DIR/"
fi

# 复制示例数据
if [ -f "sample-questions.json" ]; then
    cp sample-questions.json "$DIST_DIR/"
fi

# 创建服务管理脚本（基于 server-manager.sh）
cat > "$DIST_DIR/service-manager.sh" << 'EOF'
#!/bin/bash

# 智能刷题平台服务管理脚本 - Linux 版本
# 用于启动、停止、重启前端和后端服务

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 端口配置
FRONTEND_PORT=5173
BACKEND_PORT=3005

# 进程ID文件
FRONTEND_PID_FILE=".frontend.pid"
BACKEND_PID_FILE=".backend.pid"

# 日志文件
FRONTEND_LOG="frontend.log"
BACKEND_LOG="backend.log"

# 二进制文件名
BACKEND_BINARY="exam-server-linux-x64"

# 显示帮助信息
show_help() {
    echo -e "${BLUE}智能刷题平台服务管理脚本${NC}"
    echo "用法: $0 {start|stop|restart|status|logs|help}"
    echo ""
    echo "命令:"
    echo "  start    - 启动前端和后端服务"
    echo "  stop     - 停止前端和后端服务"
    echo "  restart  - 重启前端和后端服务"
    echo "  status   - 查看服务状态"
    echo "  logs     - 查看服务日志"
    echo "  help     - 显示此帮助信息"
    echo ""
    echo "配置信息:"
    echo "  前端端口: $FRONTEND_PORT"
    echo "  后端端口: $BACKEND_PORT"
    echo "  后端程序: $BACKEND_BINARY"
}

# 检查依赖
check_dependencies() {
    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        echo -e "${RED}错误: 未找到 Node.js，请先安装 Node.js${NC}"
        echo "安装方法："
        echo "  Ubuntu/Debian: sudo apt-get install nodejs npm"
        echo "  CentOS/RHEL: sudo yum install nodejs npm"
        echo "  或访问: https://nodejs.org/"
        return 1
    fi
    
    # 检查 npm
    if ! command -v npm &> /dev/null; then
        echo -e "${RED}错误: 未找到 npm${NC}"
        return 1
    fi
    
    # 检查后端二进制文件
    if [ ! -f "$BACKEND_BINARY" ]; then
        echo -e "${RED}错误: 未找到后端程序 $BACKEND_BINARY${NC}"
        return 1
    fi
    
    return 0
}

# 检查端口是否被占用
check_port() {
    local port=$1
    local service=$2
    if netstat -tuln 2>/dev/null | grep -q ":$port " || ss -tuln 2>/dev/null | grep -q ":$port "; then
        echo -e "${YELLOW}警告: $service 端口 $port 已被占用${NC}"
        return 1
    fi
    return 0
}

# 安装前端依赖
install_frontend_deps() {
    if [ ! -d "node_modules" ]; then
        echo -e "${BLUE}安装前端依赖...${NC}"
        npm install
        if [ $? -ne 0 ]; then
            echo -e "${RED}前端依赖安装失败${NC}"
            return 1
        fi
    fi
    return 0
}

# 启动前端服务
start_frontend() {
    echo -e "${BLUE}启动前端服务...${NC}"
    
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${YELLOW}前端服务已在运行 (PID: $pid)${NC}"
            return 0
        fi
    fi
    
    check_port $FRONTEND_PORT "前端" || return 1
    
    # 安装依赖
    install_frontend_deps || return 1
    
    # 启动前端服务并记录PID
    nohup npm run dev > "$FRONTEND_LOG" 2>&1 &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > "$FRONTEND_PID_FILE"
    
    echo -e "${GREEN}前端服务启动成功 (PID: $FRONTEND_PID)${NC}"
    echo -e "${GREEN}前端地址: http://localhost:$FRONTEND_PORT${NC}"
    
    # 等待前端服务完全启动
    sleep 3
}

# 启动后端服务
start_backend() {
    echo -e "${BLUE}启动后端服务...${NC}"
    
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${YELLOW}后端服务已在运行 (PID: $pid)${NC}"
            return 0
        fi
    fi
    
    check_port $BACKEND_PORT "后端" || return 1
    
    # 设置执行权限
    chmod +x "$BACKEND_BINARY"
    
    # 启动后端服务并记录PID
    nohup ./"$BACKEND_BINARY" > "$BACKEND_LOG" 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > "$BACKEND_PID_FILE"
    
    echo -e "${GREEN}后端服务启动成功 (PID: $BACKEND_PID)${NC}"
    echo -e "${GREEN}后端地址: http://localhost:$BACKEND_PORT${NC}"
    
    # 等待后端服务完全启动
    sleep 2
}

# 停止前端服务
stop_frontend() {
    echo -e "${BLUE}停止前端服务...${NC}"
    
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p $pid > /dev/null 2>&1; then
            kill $pid
            echo -e "${GREEN}前端服务已停止 (PID: $pid)${NC}"
        else
            echo -e "${YELLOW}前端服务未运行${NC}"
        fi
        rm -f "$FRONTEND_PID_FILE"
    else
        echo -e "${YELLOW}前端服务未运行${NC}"
    fi
    
    # 强制杀死占用端口的进程
    local pids=$(netstat -tuln 2>/dev/null | grep ":$FRONTEND_PORT " | awk '{print $7}' | cut -d'/' -f1 2>/dev/null)
    if [ -n "$pids" ]; then
        echo $pids | xargs kill -9 2>/dev/null
        echo -e "${YELLOW}已强制清理前端端口占用${NC}"
    fi
}

# 停止后端服务
stop_backend() {
    echo -e "${BLUE}停止后端服务...${NC}"
    
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p $pid > /dev/null 2>&1; then
            kill $pid
            echo -e "${GREEN}后端服务已停止 (PID: $pid)${NC}"
        else
            echo -e "${YELLOW}后端服务未运行${NC}"
        fi
        rm -f "$BACKEND_PID_FILE"
    else
        echo -e "${YELLOW}后端服务未运行${NC}"
    fi
    
    # 强制杀死占用端口的进程
    local pids=$(netstat -tuln 2>/dev/null | grep ":$BACKEND_PORT " | awk '{print $7}' | cut -d'/' -f1 2>/dev/null)
    if [ -n "$pids" ]; then
        echo $pids | xargs kill -9 2>/dev/null
        echo -e "${YELLOW}已强制清理后端端口占用${NC}"
    fi
}

# 查看服务状态
check_status() {
    echo -e "${BLUE}服务状态检查:${NC}"
    
    # 检查前端服务
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${GREEN}✓ 前端服务运行中 (PID: $pid, 端口: $FRONTEND_PORT)${NC}"
        else
            echo -e "${RED}✗ 前端服务已停止 (PID文件存在但进程不存在)${NC}"
        fi
    else
        echo -e "${YELLOW}○ 前端服务未运行${NC}"
    fi
    
    # 检查后端服务
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${GREEN}✓ 后端服务运行中 (PID: $pid, 端口: $BACKEND_PORT)${NC}"
        else
            echo -e "${RED}✗ 后端服务已停止 (PID文件存在但进程不存在)${NC}"
        fi
    else
        echo -e "${YELLOW}○ 后端服务未运行${NC}"
    fi
    
    # 检查端口占用情况
    echo -e "${BLUE}端口占用情况:${NC}"
    if netstat -tuln 2>/dev/null | grep -q ":$FRONTEND_PORT " || ss -tuln 2>/dev/null | grep -q ":$FRONTEND_PORT "; then
        echo -e "${GREEN}✓ 前端端口 $FRONTEND_PORT 已被占用${NC}"
    else
        echo -e "${YELLOW}○ 前端端口 $FRONTEND_PORT 未被占用${NC}"
    fi
    
    if netstat -tuln 2>/dev/null | grep -q ":$BACKEND_PORT " || ss -tuln 2>/dev/null | grep -q ":$BACKEND_PORT "; then
        echo -e "${GREEN}✓ 后端端口 $BACKEND_PORT 已被占用${NC}"
    else
        echo -e "${YELLOW}○ 后端端口 $BACKEND_PORT 未被占用${NC}"
    fi
}

# 查看日志
show_logs() {
    echo -e "${BLUE}选择要查看的日志:${NC}"
    echo "1) 前端日志"
    echo "2) 后端日志"
    echo "3) 全部日志"
    echo "4) 实时前端日志"
    echo "5) 实时后端日志"
    echo -n "请选择 [1-5]: "
    read choice
    
    case $choice in
        1)
            if [ -f "$FRONTEND_LOG" ]; then
                echo -e "${BLUE}=== 前端日志 ===${NC}"
                tail -50 "$FRONTEND_LOG"
            else
                echo -e "${YELLOW}前端日志文件不存在${NC}"
            fi
            ;;
        2)
            if [ -f "$BACKEND_LOG" ]; then
                echo -e "${BLUE}=== 后端日志 ===${NC}"
                tail -50 "$BACKEND_LOG"
            else
                echo -e "${YELLOW}后端日志文件不存在${NC}"
            fi
            ;;
        3)
            if [ -f "$FRONTEND_LOG" ]; then
                echo -e "${BLUE}=== 前端日志 ===${NC}"
                tail -20 "$FRONTEND_LOG"
            fi
            echo ""
            if [ -f "$BACKEND_LOG" ]; then
                echo -e "${BLUE}=== 后端日志 ===${NC}"
                tail -20 "$BACKEND_LOG"
            fi
            ;;
        4)
            if [ -f "$FRONTEND_LOG" ]; then
                echo -e "${BLUE}实时前端日志 (Ctrl+C 退出)${NC}"
                tail -f "$FRONTEND_LOG"
            else
                echo -e "${YELLOW}前端日志文件不存在${NC}"
            fi
            ;;
        5)
            if [ -f "$BACKEND_LOG" ]; then
                echo -e "${BLUE}实时后端日志 (Ctrl+C 退出)${NC}"
                tail -f "$BACKEND_LOG"
            else
                echo -e "${YELLOW}后端日志文件不存在${NC}"
            fi
            ;;
        *)
            echo -e "${RED}无效选择${NC}"
            ;;
    esac
}

# 主函数
main() {
    case "$1" in
        start)
            check_dependencies || exit 1
            start_backend
            start_frontend
            echo ""
            check_status
            echo ""
            echo -e "${GREEN}服务启动完成！${NC}"
            echo -e "${GREEN}前端访问: http://localhost:$FRONTEND_PORT${NC}"
            echo -e "${GREEN}后端API: http://localhost:$BACKEND_PORT${NC}"
            ;;
        stop)
            stop_frontend
            stop_backend
            check_status
            ;;
        restart)
            stop_frontend
            stop_backend
            sleep 2
            check_dependencies || exit 1
            start_backend
            start_frontend
            echo ""
            check_status
            ;;
        status)
            check_status
            ;;
        logs)
            show_logs
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}未知命令: $1${NC}"
            echo "使用 '$0 help' 查看帮助信息"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$1"
EOF

# 创建快速启动脚本
cat > "$DIST_DIR/start.sh" << 'EOF'
#!/bin/bash
echo "启动智能刷题平台..."
./service-manager.sh start
EOF

# 创建快速停止脚本
cat > "$DIST_DIR/stop.sh" << 'EOF'
#!/bin/bash
echo "停止智能刷题平台..."
./service-manager.sh stop
EOF

# 创建部署说明
cat > "$DIST_DIR/README.md" << 'EOF'
# 智能刷题平台 - Linux x86_64 全栈部署

## 系统要求

- Linux x86_64 系统
- Node.js 16+ 和 npm
- 可用端口 5173（前端）和 3005（后端）
- 至少 200MB 磁盘空间
- 至少 512MB 内存

## 快速开始

### 1. 解压并进入目录
```bash
tar -xzf exam-fullstack-linux-x64-*.tar.gz
cd exam-fullstack-linux-x64-*
```

### 2. 启动服务
```bash
# 方式1: 使用快速启动脚本
./start.sh

# 方式2: 使用服务管理脚本
./service-manager.sh start
```

### 3. 访问应用
- 前端页面: http://localhost:5173
- 后端API: http://localhost:3005

## 文件说明

- `exam-server-linux-x64` - 后端服务程序（Go 编译的二进制文件）
- `src/` - 前端 Vue.js 源码
- `go-server/` - 后端 Go 源码（用于调试）
- `package.json` - 前端依赖配置
- `exam.db` - SQLite 数据库文件
- `sample-questions.json` - 示例题目数据
- `service-manager.sh` - 服务管理脚本
- `start.sh` - 快速启动脚本
- `stop.sh` - 快速停止脚本

## 服务管理

### 使用服务管理脚本
```bash
# 启动所有服务
./service-manager.sh start

# 停止所有服务
./service-manager.sh stop

# 重启所有服务
./service-manager.sh restart

# 查看服务状态
./service-manager.sh status

# 查看日志
./service-manager.sh logs
```

### 手动管理
```bash
# 启动后端
./exam-server-linux-x64 &

# 启动前端
npm run dev &
```

## 环境安装

### Ubuntu/Debian
```bash
# 安装 Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# 验证安装
node --version
npm --version
```

### CentOS/RHEL
```bash
# 安装 Node.js
curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
sudo yum install -y nodejs

# 验证安装
node --version
npm --version
```

## 配置

### 端口配置
如需修改端口，编辑以下文件：
- 前端端口: `vite.config.js` 中的 `server.port`
- 后端端口: `go-server/.env` 中的 `PORT` 或修改源码重新编译

### 数据库
使用 SQLite 数据库，数据文件: `exam.db`
首次运行会自动创建必要的表结构

## 故障排除

### 端口被占用
```bash
# 查看端口占用
netstat -tuln | grep 5173
netstat -tuln | grep 3005

# 杀死占用进程
sudo kill -9 <PID>
```

### Node.js 依赖问题
```bash
# 清理并重新安装依赖
rm -rf node_modules package-lock.json
npm install
```

### 权限问题
```bash
# 设置执行权限
chmod +x exam-server-linux-x64 *.sh
```

### 查看详细日志
```bash
# 查看前端日志
tail -f frontend.log

# 查看后端日志
tail -f backend.log
```

## 架构说明

这是一个前后端分离的部署方案：
- **前端**: Vue.js + Vite 开发服务器 (端口 5173)
- **后端**: Go 服务 (端口 3005)
- **数据库**: SQLite (本地文件)

前端通过 API 调用与后端通信，支持完整的开发和调试功能。

## 技术支持

如有问题请检查日志文件或联系技术支持。
EOF

# 设置脚本执行权限
chmod +x "$DIST_DIR"/*.sh

echo -e "${BLUE}创建压缩包...${NC}"
tar -czf "${PACKAGE_NAME}.tar.gz" "$DIST_DIR"

# 清理临时文件
rm -rf "$TEMP_DIR" "$DIST_DIR" exam-server-linux-x64

# 显示结果
echo ""
echo -e "${GREEN}=== 全栈打包完成 ===${NC}"
echo -e "${GREEN}打包文件: ${PACKAGE_NAME}.tar.gz${NC}"
echo -e "${GREEN}文件大小: $(du -h "${PACKAGE_NAME}.tar.gz" | cut -f1)${NC}"
echo ""
echo -e "${BLUE}部署到 Linux x86_64 系统的步骤:${NC}"
echo "1. 将 ${PACKAGE_NAME}.tar.gz 上传到 Linux 服务器"
echo "2. 解压: tar -xzf ${PACKAGE_NAME}.tar.gz"
echo "3. 进入目录: cd ${PACKAGE_NAME}"
echo "4. 启动服务: ./start.sh"
echo "5. 访问应用:"
echo "   - 前端: http://localhost:5173"
echo "   - 后端: http://localhost:3005"
echo ""
echo -e "${YELLOW}注意事项:${NC}"
echo "- 确保目标系统为 Linux x86_64 架构"
echo "- 确保已安装 Node.js 16+ 和 npm"
echo "- 确保端口 5173 和 3005 未被占用"
echo "- 首次启动会自动安装前端依赖"
echo ""
echo -e "${GREEN}打包完成！${NC}"

# 自动上传到服务器
if [ "$AUTO_UPLOAD" = true ]; then
    echo ""
    echo -e "${BLUE}=== 开始上传到服务器 ===${NC}"
    echo -e "${BLUE}服务器: ${SERVER_USER}@${SERVER_HOST}:${SERVER_PORT}${NC}"
    echo -e "${BLUE}目标路径: ${SERVER_PATH}${NC}"
    
    # 检查是否有 sshpass
    if ! command -v sshpass &> /dev/null; then
        echo -e "${YELLOW}警告: 未找到 sshpass 工具${NC}"
        echo -e "${YELLOW}正在安装 sshpass...${NC}"
        
        # 检测操作系统并安装 sshpass
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if command -v brew &> /dev/null; then
                brew install hudochenkov/sshpass/sshpass 2>/dev/null || {
                    echo -e "${RED}安装 sshpass 失败，请手动安装: brew install hudochenkov/sshpass/sshpass${NC}"
                    echo -e "${YELLOW}跳过自动上传，请手动上传文件${NC}"
                    exit 0
                }
            else
                echo -e "${RED}未找到 Homebrew，无法自动安装 sshpass${NC}"
                echo -e "${YELLOW}请手动安装: brew install hudochenkov/sshpass/sshpass${NC}"
                echo -e "${YELLOW}或使用手动上传: scp -P ${SERVER_PORT} ${PACKAGE_NAME}.tar.gz ${SERVER_USER}@${SERVER_HOST}:${SERVER_PATH}/${NC}"
                exit 0
            fi
        elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Linux
            if command -v apt-get &> /dev/null; then
                sudo apt-get update && sudo apt-get install -y sshpass 2>/dev/null || {
                    echo -e "${RED}安装 sshpass 失败，请手动安装: sudo apt-get install sshpass${NC}"
                    echo -e "${YELLOW}跳过自动上传，请手动上传文件${NC}"
                    exit 0
                }
            elif command -v yum &> /dev/null; then
                sudo yum install -y sshpass 2>/dev/null || {
                    echo -e "${RED}安装 sshpass 失败，请手动安装: sudo yum install sshpass${NC}"
                    echo -e "${YELLOW}跳过自动上传，请手动上传文件${NC}"
                    exit 0
                }
            else
                echo -e "${RED}未找到包管理器，无法自动安装 sshpass${NC}"
                echo -e "${YELLOW}请手动安装 sshpass 后重新运行脚本${NC}"
                exit 0
            fi
        else
            echo -e "${RED}不支持的操作系统类型${NC}"
            echo -e "${YELLOW}跳过自动上传，请手动上传文件${NC}"
            exit 0
        fi
    fi
    
    # 测试服务器连接
    echo -e "${BLUE}测试服务器连接...${NC}"
    if sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -p "$SERVER_PORT" "$SERVER_USER@$SERVER_HOST" "echo 'Connection successful'" &>/dev/null; then
        echo -e "${GREEN}服务器连接成功${NC}"
    else
        echo -e "${YELLOW}警告: 无法连接到服务器，跳过上传${NC}"
        echo -e "${YELLOW}请检查网络连接和服务器配置${NC}"
        exit 0
    fi
    
    # 上传文件
    echo -e "${BLUE}正在上传 ${PACKAGE_NAME}.tar.gz ...${NC}"
    if sshpass -p "$SERVER_PASSWORD" scp -o StrictHostKeyChecking=no -P "$SERVER_PORT" "${PACKAGE_NAME}.tar.gz" "${SERVER_USER}@${SERVER_HOST}:${SERVER_PATH}/"; then
        echo -e "${GREEN}文件上传成功！${NC}"
        echo ""
        echo -e "${GREEN}文件已上传到服务器:${NC}"
        echo -e "${GREEN}  ${SERVER_PATH}/${PACKAGE_NAME}.tar.gz${NC}"
        echo ""
        echo -e "${BLUE}在服务器上执行以下命令来部署:${NC}"
        echo "  ssh -p ${SERVER_PORT} ${SERVER_USER}@${SERVER_HOST}"
        echo "  cd ${SERVER_PATH}"
        echo "  tar -xzf ${PACKAGE_NAME}.tar.gz"
        echo "  cd ${PACKAGE_NAME}"
        echo "  ./start.sh"
    else
        echo -e "${RED}文件上传失败${NC}"
        echo -e "${YELLOW}请检查网络连接和服务器配置${NC}"
        exit 1
    fi
else
    echo ""
    echo -e "${YELLOW}自动上传已禁用${NC}"
    echo -e "${YELLOW}要启用自动上传，请在脚本中设置 AUTO_UPLOAD=true${NC}"
fi