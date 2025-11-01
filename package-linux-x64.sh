#!/bin/bash

# Linux x86_64 打包脚本
# 将项目打包为可在 Linux x86_64 系统直接运行的包

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
PACKAGE_NAME="exam-practice-linux-x64-$(date +%Y%m%d-%H%M%S)"
TEMP_DIR="build-temp"
DIST_DIR="$PACKAGE_NAME"

echo -e "${BLUE}=== 智能刷题平台 Linux x86_64 打包工具 ===${NC}"
echo -e "${BLUE}开始打包项目...${NC}"

# 检查环境
echo -e "${BLUE}检查构建环境...${NC}"

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}错误: 未找到 Node.js，请先安装 Node.js${NC}"
    exit 1
fi

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

echo -e "${BLUE}构建前端...${NC}"
# 安装前端依赖
npm install

# 构建前端
npm run build

# 检查构建结果
if [ ! -d "dist" ]; then
    echo -e "${RED}错误: 前端构建失败，未找到 dist 目录${NC}"
    exit 1
fi

echo -e "${GREEN}前端构建完成${NC}"

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

# 复制前端构建文件
cp -r dist "$DIST_DIR/"

# 复制 Go 二进制文件
cp exam-server-linux-x64 "$DIST_DIR/"

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

# 创建启动脚本
cat > "$DIST_DIR/start.sh" << 'EOF'
#!/bin/bash

# 智能刷题平台启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
PORT=3005
BINARY_NAME="exam-server-linux-x64"
PID_FILE=".exam-server.pid"
LOG_FILE="exam-server.log"

# 检查二进制文件
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}错误: 未找到服务器程序 $BINARY_NAME${NC}"
    exit 1
fi

# 设置执行权限
chmod +x "$BINARY_NAME"

# 检查端口
if netstat -tuln 2>/dev/null | grep -q ":$PORT "; then
    echo -e "${YELLOW}警告: 端口 $PORT 已被占用${NC}"
    echo "请检查是否有其他服务正在运行，或修改配置文件中的端口设置"
    exit 1
fi

echo -e "${BLUE}启动智能刷题平台...${NC}"
echo -e "${BLUE}服务端口: $PORT${NC}"
echo -e "${BLUE}访问地址: http://localhost:$PORT${NC}"
echo -e "${BLUE}日志文件: $LOG_FILE${NC}"
echo ""

# 启动服务器
nohup ./"$BINARY_NAME" > "$LOG_FILE" 2>&1 &
SERVER_PID=$!

# 保存 PID
echo $SERVER_PID > "$PID_FILE"

echo -e "${GREEN}服务器已启动 (PID: $SERVER_PID)${NC}"
echo ""
echo "使用以下命令管理服务:"
echo "  查看日志: tail -f $LOG_FILE"
echo "  停止服务: kill $SERVER_PID 或 ./stop.sh"
echo "  查看状态: ps -p $SERVER_PID"
echo ""
echo -e "${GREEN}请在浏览器中访问: http://localhost:$PORT${NC}"
EOF

# 创建停止脚本
cat > "$DIST_DIR/stop.sh" << 'EOF'
#!/bin/bash

# 智能刷题平台停止脚本

PID_FILE=".exam-server.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null 2>&1; then
        echo "正在停止服务器 (PID: $PID)..."
        kill $PID
        rm -f "$PID_FILE"
        echo "服务器已停止"
    else
        echo "服务器进程不存在"
        rm -f "$PID_FILE"
    fi
else
    echo "未找到 PID 文件，服务器可能未运行"
fi
EOF

# 创建状态检查脚本
cat > "$DIST_DIR/status.sh" << 'EOF'
#!/bin/bash

# 智能刷题平台状态检查脚本

PID_FILE=".exam-server.pid"
PORT=3005

echo "=== 智能刷题平台状态 ==="

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null 2>&1; then
        echo "✓ 服务器正在运行 (PID: $PID)"
        
        # 检查端口
        if netstat -tuln 2>/dev/null | grep -q ":$PORT "; then
            echo "✓ 端口 $PORT 正在监听"
            echo "✓ 访问地址: http://localhost:$PORT"
        else
            echo "✗ 端口 $PORT 未在监听"
        fi
        
        # 显示资源使用情况
        echo ""
        echo "资源使用情况:"
        ps -p $PID -o pid,ppid,pcpu,pmem,etime,cmd --no-headers
    else
        echo "✗ 服务器进程不存在"
        rm -f "$PID_FILE"
    fi
else
    echo "✗ 服务器未运行"
fi

echo ""
echo "日志文件:"
if [ -f "exam-server.log" ]; then
    echo "  exam-server.log ($(wc -l < exam-server.log) 行)"
    echo "  最后 5 行日志:"
    tail -5 exam-server.log | sed 's/^/    /'
else
    echo "  无日志文件"
fi
EOF

# 创建部署说明
cat > "$DIST_DIR/README.md" << 'EOF'
# 智能刷题平台 - Linux x86_64

## 快速开始

### 1. 解压并进入目录
```bash
tar -xzf exam-practice-linux-x64-*.tar.gz
cd exam-practice-linux-x64-*
```

### 2. 启动服务
```bash
./start.sh
```

### 3. 访问应用
在浏览器中打开: http://localhost:3005

## 文件说明

- `exam-server-linux-x64` - 服务器程序（Go 编译的二进制文件）
- `dist/` - 前端静态文件
- `exam.db` - SQLite 数据库文件
- `sample-questions.json` - 示例题目数据
- `start.sh` - 启动脚本
- `stop.sh` - 停止脚本
- `status.sh` - 状态检查脚本

## 管理命令

```bash
# 启动服务
./start.sh

# 停止服务
./stop.sh

# 查看状态
./status.sh

# 查看日志
tail -f exam-server.log
```

## 配置

### 端口配置
默认端口: 3005
如需修改端口，请编辑 `.env` 文件或在启动前设置环境变量:
```bash
export PORT=8080
./start.sh
```

### 数据库
使用 SQLite 数据库，数据文件: `exam.db`
首次运行会自动创建必要的表结构

## 系统要求

- Linux x86_64 系统
- 可用端口 3005（或自定义端口）
- 至少 50MB 磁盘空间
- 至少 128MB 内存

## 故障排除

### 端口被占用
```bash
# 查看端口占用
netstat -tuln | grep 3005
# 或
lsof -i :3005

# 杀死占用进程
kill -9 <PID>
```

### 权限问题
```bash
# 设置执行权限
chmod +x exam-server-linux-x64 *.sh
```

### 查看详细日志
```bash
tail -f exam-server.log
```

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
echo -e "${GREEN}=== 打包完成 ===${NC}"
echo -e "${GREEN}打包文件: ${PACKAGE_NAME}.tar.gz${NC}"
echo -e "${GREEN}文件大小: $(du -h "${PACKAGE_NAME}.tar.gz" | cut -f1)${NC}"
echo ""
echo -e "${BLUE}部署到 Linux x86_64 系统的步骤:${NC}"
echo "1. 将 ${PACKAGE_NAME}.tar.gz 上传到 Linux 服务器"
echo "2. 解压: tar -xzf ${PACKAGE_NAME}.tar.gz"
echo "3. 进入目录: cd ${PACKAGE_NAME}"
echo "4. 启动服务: ./start.sh"
echo "5. 访问应用: http://localhost:3005"
echo ""
echo -e "${YELLOW}注意事项:${NC}"
echo "- 确保目标系统为 Linux x86_64 架构"
echo "- 确保端口 3005 未被占用"
echo "- 服务器需要有网络访问权限"
echo ""
echo -e "${GREEN}打包完成！${NC}"