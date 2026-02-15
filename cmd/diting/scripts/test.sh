#!/bin/bash
# Diting 代理与审批测试脚本（需先启动 diting，默认监听 8080）
# 可重复执行：在 cmd/diting 下先 make run 或 ./bin/diting，再本脚本。
# 代理端口可通过环境变量覆盖：PROXY_PORT=8080 ./scripts/test.sh

PROXY_PORT="${PROXY_PORT:-8080}"
PROXY="http://127.0.0.1:${PROXY_PORT}"

echo "╔════════════════════════════════════════════════════════╗"
echo "║         Diting 测试脚本                                ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""
echo "代理: $PROXY（可设置 PROXY_PORT 覆盖）"
echo ""

# 测试 1: 低风险请求（GET，应该自动放行）
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 1: 低风险 GET 请求（应该自动放行）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -x "$PROXY" -s -o /dev/null -w "状态码: %{http_code}\n" https://httpbin.org/get
echo ""

# 测试 2: 中风险请求（POST 到安全域名）
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 2: 中风险 POST 请求（需要审批）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "请在飞书中回复 '批准' 或 '拒绝'"
curl -x "$PROXY" -X POST -s -o /dev/null -w "状态码: %{http_code}\n" https://httpbin.org/post
echo ""

# 测试 3: 高风险请求（DELETE 到危险路径）
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 3: 高风险 DELETE 请求（需要审批）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "请在飞书中回复 '批准' 或 '拒绝'"
curl -x "$PROXY" -X DELETE -s -o /dev/null -w "状态码: %{http_code}\n" https://httpbin.org/delete
echo ""

# 测试 4: HTTPS CONNECT
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 4: HTTPS CONNECT 隧道"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -x "$PROXY" -s -o /dev/null -w "状态码: %{http_code}\n" https://www.google.com
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试完成！请检查："
echo "1. Diting 终端输出"
echo "2. 飞书群聊中的审批消息"
echo "3. logs/audit.jsonl 审计日志"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
