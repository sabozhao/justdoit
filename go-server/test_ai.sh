#!/bin/bash

# AI测试脚本 - 测试并优化提示词直到生成正确的JSON

echo "========================================"
echo "AI识别功能测试脚本"
echo "========================================"
echo ""

# 检查环境变量
if [ -z "$TENCENT_SECRET_ID" ] || [ -z "$TENCENT_SECRET_KEY" ]; then
    echo "❌ 错误: 请先设置环境变量 TENCENT_SECRET_ID 和 TENCENT_SECRET_KEY"
    echo ""
    echo "示例:"
    echo "export TENCENT_SECRET_ID=your_secret_id"
    echo "export TENCENT_SECRET_KEY=your_secret_key"
    exit 1
fi

echo "✓ 环境变量已设置"
echo ""

# 运行测试
echo "运行AI识别测试..."
echo ""

cd "$(dirname "$0")"
go test -v -run TestAIRecognition 2>&1 | tee test_result.log

echo ""
echo "========================================"
echo "测试完成，结果已保存到 test_result.log"
echo "========================================"

