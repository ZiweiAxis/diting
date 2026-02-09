#!/usr/bin/env bash
# 触发一条需审批的请求，用于验证飞书交互卡片（需先启动 All-in-One，且策略对 /admin 为 review）
set -e
GATEWAY="${GATEWAY:-http://localhost:8080}"
echo "触发审批请求: POST $GATEWAY/admin (Host: example.com)"
curl -s -X POST "$GATEWAY/admin" -H "Host: example.com" -d '{}' || true
echo ""
echo "若网关在等待审批，请到飞书查看「Diting 待确认」交互卡片并点击批准/拒绝。"
