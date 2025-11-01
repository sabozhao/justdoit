#!/bin/bash

# 智能刷题平台启动脚本
# 支持不同环境的API配置

echo "🚀 智能刷题平台启动脚本"
echo "================================"

# 检查参数
if [ "$1" = "prod" ] || [ "$1" = "production" ]; then
    echo "🌐 启动生产环境模式 (API: https://examtest.top/api)"
    npm run dev:prod
elif [ "$1" = "local" ] || [ "$1" = "dev" ]; then
    echo "🏠 启动本地开发模式 (API: http://localhost:3005/api)"
    npm run dev:local
else
    echo "❓ 使用方法:"
    echo "  ./start.sh local   - 本地开发模式"
    echo "  ./start.sh prod    - 生产环境模式"
    echo ""
    echo "🏠 默认启动本地开发模式..."
    npm run dev:local
fi
