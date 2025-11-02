#!/bin/bash

# 智能刷题平台重启脚本
# 用法: ./restart.sh [local|prod] [--no-build]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认模式
MODE="local"
NO_BUILD=false

# 解析参数
for arg in "$@"; do
    case $arg in
        local|dev)
            MODE="local"
            shift
            ;;
        prod|production)
            MODE="prod"
            shift
            ;;
        --no-build)
            NO_BUILD=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

echo -e "${BLUE}🔄 智能刷题平台重启脚本${NC}"
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}模式: ${MODE}${NC}"

# 进入项目目录
cd "$(dirname "$0")"

# 1. 停止所有正在运行的进程
echo ""
echo -e "${BLUE}步骤 1: 停止现有服务...${NC}"
pkill -f "exam-server" 2>/dev/null && echo -e "${GREEN}✓ 后端进程已停止${NC}" || echo -e "${YELLOW}⚠ 未找到后端进程${NC}"
pkill -f "vite.*--mode" 2>/dev/null && echo -e "${GREEN}✓ 前端进程已停止${NC}" || echo -e "${YELLOW}⚠ 未找到前端进程${NC}"
sleep 2
echo -e "${GREEN}✓ 清理完成${NC}"

# 2. 重新编译后端（如果需要）
if [ "$NO_BUILD" = false ]; then
    echo ""
    echo -e "${BLUE}步骤 2: 编译后端服务...${NC}"
    cd go-server
    echo -e "${BLUE}正在运行 go mod tidy...${NC}"
    go mod tidy 2>&1 | grep -v "go: downloading" || true
    
    echo -e "${BLUE}正在编译 Go 后端...${NC}"
    if go build -o exam-server . 2>&1; then
        echo -e "${GREEN}✓ 后端编译成功${NC}"
    else
        echo -e "${RED}✗ 后端编译失败${NC}"
        exit 1
    fi
    cd ..
else
    echo ""
    echo -e "${YELLOW}⚠ 跳过编译步骤 (--no-build)${NC}"
fi

# 3. 启动后端服务
echo ""
echo -e "${BLUE}步骤 3: 启动后端服务...${NC}"
cd go-server
if [ ! -f "./exam-server" ]; then
    echo -e "${RED}错误: 找不到 exam-server 二进制文件${NC}"
    echo -e "${YELLOW}提示: 请先运行编译或移除 --no-build 参数${NC}"
    exit 1
fi

# 检查端口是否被占用
if lsof -Pi :3005 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠ 端口 3005 已被占用，尝试停止占用该端口的进程...${NC}"
    lsof -ti:3005 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

nohup ./exam-server > ../backend.log 2>&1 &
BACKEND_PID=$!
echo $BACKEND_PID > ../.backend.pid
cd ..
echo -e "${GREEN}✓ 后端服务已启动 (PID: $BACKEND_PID)${NC}"
echo -e "${BLUE}后端地址: http://localhost:3005${NC}"
echo -e "${BLUE}后端日志: tail -f backend.log${NC}"

# 等待后端启动
echo -e "${BLUE}等待后端服务就绪...${NC}"
sleep 3

# 检查后端健康状态
for i in {1..10}; do
    if curl -s http://localhost:3005/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 后端服务运行正常${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${YELLOW}⚠ 后端服务可能未正常启动，请查看日志: tail -f backend.log${NC}"
    else
        sleep 1
    fi
done

# 4. 启动前端服务
echo ""
echo -e "${BLUE}步骤 4: 启动前端服务...${NC}"
if [ "$MODE" == "local" ] || [ "$MODE" == "dev" ]; then
    echo -e "${BLUE}🏠 启动本地开发模式 (API: http://localhost:3005/api)${NC}"
    nohup npm run dev:local > frontend.log 2>&1 &
elif [ "$MODE" == "prod" ] || [ "$MODE" == "production" ]; then
    echo -e "${BLUE}🌐 启动生产环境模式 (API: https://examtest.top/api)${NC}"
    nohup npm run dev:prod > frontend.log 2>&1 &
else
    echo -e "${RED}错误: 无效的模式 '$MODE'。请使用 'local' 或 'prod'${NC}"
    exit 1
fi

FRONTEND_PID=$!
echo $FRONTEND_PID > .frontend.pid
echo -e "${GREEN}✓ 前端服务已启动 (PID: $FRONTEND_PID)${NC}"
echo -e "${BLUE}前端日志: tail -f frontend.log${NC}"

# 等待前端启动
echo -e "${BLUE}等待前端服务就绪...${NC}"
sleep 5

# 检查前端端口
FRONTEND_PORT=""
for port in 5173 5174 5175 5176 5177; do
    if lsof -ti:$port 2>/dev/null >/dev/null; then
        FRONTEND_PORT=$port
        break
    fi
done

if [ -n "$FRONTEND_PORT" ]; then
    echo -e "${GREEN}✓ 前端服务运行在端口: $FRONTEND_PORT${NC}"
    echo -e "${BLUE}前端地址: http://localhost:$FRONTEND_PORT${NC}"
else
    echo -e "${YELLOW}⚠ 未能自动检测前端端口，请查看日志: tail -f frontend.log${NC}"
    echo -e "${BLUE}通常前端运行在: http://localhost:5173${NC}"
    FRONTEND_PORT="5173"
fi

# 完成
echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}✅ 重启完成！${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo -e "${BLUE}服务信息:${NC}"
echo -e "  后端 PID: ${BACKEND_PID}"
echo -e "  前端 PID: ${FRONTEND_PID}"
echo -e "  后端地址: http://localhost:3005"
echo -e "  前端地址: http://localhost:${FRONTEND_PORT}"
echo ""
echo -e "${BLUE}查看日志:${NC}"
echo -e "  后端: tail -f backend.log"
echo -e "  前端: tail -f frontend.log"
echo ""
echo -e "${BLUE}停止服务:${NC}"
echo -e "  ./restart.sh --no-build  # 仅重启不重新编译"
echo -e "  pkill -f exam-server    # 停止后端"
echo -e "  pkill -f vite           # 停止前端"
echo ""

