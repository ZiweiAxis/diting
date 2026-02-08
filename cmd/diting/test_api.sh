#!/bin/bash

APP_ID="cli_a90d5a960cf89cd4"
APP_SECRET="8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"

echo "1. 获取 tenant_access_token..."
TOKEN_RESPONSE=$(curl -s -X POST "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal" \
  -H "Content-Type: application/json" \
  -d "{\"app_id\":\"$APP_ID\",\"app_secret\":\"$APP_SECRET\"}")

echo "Token 响应: $TOKEN_RESPONSE"
echo ""

TOKEN=$(echo $TOKEN_RESPONSE | grep -o '"tenant_access_token":"[^"]*"' | cut -d'"' -f4)
echo "✓ Token: $TOKEN"
echo ""

echo "2. 获取 WebSocket endpoint..."
ENDPOINT_RESPONSE=$(curl -s -X POST "https://open.feishu.cn/open-apis/im/v1/stream/get" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}')

echo "Endpoint 响应:"
echo "$ENDPOINT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$ENDPOINT_RESPONSE"
