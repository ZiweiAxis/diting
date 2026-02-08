#!/usr/bin/env bash
# 飞书审批最小验证：编译 + 启动 + 低风险测试（步骤 1～2）
# 步骤 3～5 需你在飞书中回复审批，见下方说明。

set -e
cd "$(dirname "$0")"

echo "=== 步骤 1: 编译 diting ==="
go build -o diting main.go
echo "✓ 编译成功"
echo ""

echo "=== 步骤 2: 启动 diting（前台，请保留此终端）==="
echo "  启动后请在【另一终端】执行："
echo "    curl -x http://127.0.0.1:8081 https://httpbin.org/get          # 低风险，应直接返回"
echo "    curl -x http://127.0.0.1:8081 -X DELETE https://httpbin.org/delete  # 高风险，飞书收审批后回复 approve <请求ID> 或 deny <请求ID>"
echo "  按 Ctrl+C 可停止服务。"
echo ""
./diting
