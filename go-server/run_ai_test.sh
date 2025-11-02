#!/bin/bash

# AI测试运行脚本
# 使用方法: ./run_ai_test.sh

echo "========================================"
echo "AI识别功能测试"
echo "========================================"
echo ""

cd "$(dirname "$0")"

# 检查是否设置了环境变量
if [ -z "$TENCENT_SECRET_ID" ] || [ -z "$TENCENT_SECRET_KEY" ]; then
    echo "⚠️  环境变量未设置，尝试从数据库读取配置..."
    echo ""
fi

# 构建并运行测试工具（通过debug-api入口点）
echo "构建测试工具..."
go build -o ai_test_tool main.go routes.go handlers.go wrong_questions.go exam_results.go pdf_doc_parser.go ai_service.go debug_api.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✓ 编译成功"
echo ""
echo "运行AI识别测试..."
echo ""

# 使用debug-api入口点来测试
./ai_test_tool debug-api

# 清理
rm -f ai_test_tool

