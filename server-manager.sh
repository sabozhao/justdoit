#!/bin/bash

# 服务器管理脚本
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

# 显示帮助信息
show_help() {
    echo -e "${BLUE}服务器管理脚本${NC}"
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
}

# 检查端口是否被占用
check_port() {
    local port=$1
    local service=$2
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${YELLOW}警告: $service 端口 $port 已被占用${NC}"
        return 1
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
    
    # 启动前端服务并记录PID
    npm run dev > "$FRONTEND_LOG" 2>&1 &
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
    
    # 启动后端服务并记录PID
    cd go-server
    go run main.go routes.go handlers.go wrong_questions.go exam_results.go > "../$BACKEND_LOG" 2>&1 &
    BACKEND_PID=$!
    cd ..
    echo $BACKEND_PID > "$BACKEND_PID_FILE"
    
    echo -e "${GREEN}后端服务启动成功 (PID: $BACKEND_PID)${NC}"
    echo -e "${GREEN}后端地址: http://localhost:$BACKEND_PORT${NC}"
    
    # 等待后端服务完全启动
    sleep 5
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
    if lsof -ti:$FRONTEND_PORT >/dev/null 2>&1; then
        lsof -ti:$FRONTEND_PORT | xargs kill -9
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
    if lsof -ti:$BACKEND_PORT >/dev/null 2>&1; then
        lsof -ti:$BACKEND_PORT | xargs kill -9
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
    if lsof -Pi :$FRONTEND_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${GREEN}✓ 前端端口 $FRONTEND_PORT 已被占用${NC}"
    else
        echo -e "${YELLOW}○ 前端端口 $FRONTEND_PORT 未被占用${NC}"
    fi
    
    if lsof -Pi :$BACKEND_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
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
    echo "5) 实时候端日志"
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
                echo -e "${BLUE}实时候端日志 (Ctrl+C 退出)${NC}"
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
            start_backend
            start_frontend
            check_status
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
            start_backend
            start_frontend
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

# 检查是否在项目根目录
if [ ! -f "package.json" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 执行主函数
main "$1"