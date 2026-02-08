#!/bin/bash

APP_ID="cli_a90d5a960cf89cd4"
APP_SECRET="8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"

echo "1. 获取 tenant_access_token..."
TOKEN_RESPONSE=$(curl -s -X POST "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal" \
  -H "Content-Type: application/json" \
  -d "{\"app_id\":\"$APP_ID\",\"app_secret\":\"$APP_SECRET\"}")

TOKEN=$(echo $TOKEN_RESPONSE | grep -o '"tenant_access_token":"[^"]*"' | cut -d'"' -f4)
echo "✓ Token: $TOKEN"
echo ""

# 尝试不同的 API 路径
echo "2. 尝试不同的 WebSocket endpoint API..."
echo ""

echo "尝试 1: /open-apis/im/v1/stream/get"
curl -s -X POST "https://open.feishu.cn/open-apis/im/v1/stream/get" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
echo -e "\n"

echo "尝试 2: /open-apis/event/v1/stream/get"
curl -s -X POST "https://open.feishu.cn/open-apis/event/v1/stream/get" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
echo -e "\n"

echo "尝试 3: /open-apis/im/v2/stream/get"
curl -s -X POST "https://open.feishu.cn/open-apis/im/v2/stream/get" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
echo -e "\n"
