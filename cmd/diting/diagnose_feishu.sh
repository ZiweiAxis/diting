#!/bin/bash

APP_ID="cli_a90d5a960cf89cd4"
APP_SECRET="8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"

echo "========================================="
echo "飞书 WebSocket 长连接诊断工具"
echo "========================================="
echo ""

# 1. 获取 tenant_access_token
echo "步骤 1: 获取 tenant_access_token"
echo "-----------------------------------------"
TOKEN_RESPONSE=$(curl -s -X POST "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal" \
  -H "Content-Type: application/json" \
  -d "{\"app_id\":\"$APP_ID\",\"app_secret\":\"$APP_SECRET\"}")

echo "响应: $TOKEN_RESPONSE"

TOKEN=$(echo $TOKEN_RESPONSE | grep -o '"tenant_access_token":"[^"]*"' | cut -d'"' -f4)
CODE=$(echo $TOKEN_RESPONSE | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$CODE" != "0" ]; then
    echo "❌ 获取 token 失败"
    exit 1
fi

if [ -z "$TOKEN" ]; then
    echo "❌ Token 为空"
    exit 1
fi

echo "✅ Token 获取成功: ${TOKEN:0:20}..."
echo ""

# 2. 测试 WebSocket endpoint API
echo "步骤 2: 测试 WebSocket endpoint API"
echo "-----------------------------------------"

# 尝试标准路径
echo "尝试: POST /open-apis/im/v1/stream/get"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "https://open.feishu.cn/open-apis/im/v1/stream/get" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE/d')

echo "HTTP 状态码: $HTTP_CODE"
echo "响应内容: $BODY"
echo ""

if [ "$HTTP_CODE" = "404" ]; then
    echo "⚠️  API 端点返回 404"
    echo ""
    echo "可能的原因："
    echo "1. 飞书开放平台未启用「事件订阅」功能"
    echo "2. 应用权限不足"
    echo "3. API 路径已变更"
    echo ""
    echo "解决方案："
    echo "1. 登录飞书开放平台: https://open.feishu.cn/app"
    echo "2. 进入你的应用: $APP_ID"
    echo "3. 点击「事件订阅」-> 「长连接」"
    echo "4. 启用长连接功能"
    echo ""
elif [ "$HTTP_CODE" = "200" ]; then
    echo "✅ API 调用成功"
    
    # 解析 WebSocket URL
    WS_URL=$(echo "$BODY" | grep -o '"url":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$WS_URL" ]; then
        echo "✅ WebSocket URL: $WS_URL"
        echo ""
        echo "步骤 3: 测试 WebSocket 连接"
        echo "-----------------------------------------"
        echo "尝试连接到: $WS_URL"
        echo "(需要安装 websocat 工具来测试)"
        
        if command -v websocat &> /dev/null; then
            timeout 5 websocat "$WS_URL" <<< '{"type":"PING"}' || echo "连接测试完成"
        else
            echo "提示: 安装 websocat 可以测试 WebSocket 连接"
            echo "      apt install websocat 或 brew install websocat"
        fi
    else
        echo "❌ 响应中未找到 WebSocket URL"
        echo "完整响应: $BODY"
    fi
else
    echo "❌ HTTP 错误: $HTTP_CODE"
    echo "响应: $BODY"
fi

echo ""
echo "========================================="
echo "诊断完成"
echo "========================================="
