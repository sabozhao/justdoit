#!/bin/bash

# 分类管理功能测试脚本
# 需要先启动后端服务

API_BASE="http://localhost:3005/api"
TOKEN=""

echo "=== 分类管理功能测试 ==="
echo ""

# 1. 测试登录（获取token）
echo "1. 测试登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败，请检查用户名和密码"
  exit 1
fi

echo "✅ 登录成功，Token: ${TOKEN:0:20}..."
echo ""

# 2. 测试获取分类列表
echo "2. 测试获取分类列表..."
CATEGORIES_RESPONSE=$(curl -s -X GET "$API_BASE/categories" \
  -H "Authorization: Bearer $TOKEN")

echo "响应: $CATEGORIES_RESPONSE"
echo ""

# 3. 测试创建分类（管理员）
echo "3. 测试创建分类..."
CREATE_RESPONSE=$(curl -s -X POST "$API_BASE/admin/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"测试分类","description":"这是一个测试分类","sort_order":0}')

echo "创建响应: $CREATE_RESPONSE"
echo ""

# 4. 测试获取题库列表（带筛选）
echo "4. 测试获取题库列表（全部）..."
BANKS_ALL=$(curl -s -X GET "$API_BASE/question-banks?type=all" \
  -H "Authorization: Bearer $TOKEN")

echo "全部题库数量: $(echo $BANKS_ALL | grep -o '"id"' | wc -l)"
echo ""

echo "5. 测试获取个人题库..."
BANKS_PERSONAL=$(curl -s -X GET "$API_BASE/question-banks?type=personal" \
  -H "Authorization: Bearer $TOKEN")

echo "个人题库数量: $(echo $BANKS_PERSONAL | grep -o '"id"' | wc -l)"
echo ""

echo "6. 测试获取公共题库..."
BANKS_PUBLIC=$(curl -s -X GET "$API_BASE/question-banks?type=public" \
  -H "Authorization: Bearer $TOKEN")

echo "公共题库数量: $(echo $BANKS_PUBLIC | grep -o '"id"' | wc -l)"
echo ""

echo "=== 测试完成 ==="

