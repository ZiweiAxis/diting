#!/usr/bin/env bash
# 飞书审批验证：覆盖「原有审理」（HTTP 代理）与「新逻辑」（执行层 Exec）两条路径。
# 两条路径共用同一 CHEQ、同一飞书投递与长连接，仅在请求来源不同（/admin vs /auth/exec）。
#
# 用法（在 cmd/diting 目录下执行，或通过本脚本自动 cd）：
#   ./scripts/verify_feishu_approval.sh preflight   # 仅预检
#   ./scripts/verify_feishu_approval.sh start       # 启动服务并提示
#   ./scripts/verify_feishu_approval.sh trigger     # 触发原有审理（POST /admin）
#   ./scripts/verify_feishu_approval.sh trigger_exec # 触发新逻辑（POST /auth/exec exec:sudo）
#   ./scripts/verify_feishu_approval.sh full        # 预检+启动+Phase1 原有+Phase2 新逻辑（两次飞书点击）
set -e
cd "$(dirname "$0")/.."
RUN="${RUN:-./run_acceptance.sh}"
case "${1:-full}" in
  preflight)    "$RUN" preflight ;;
  start)        "$RUN" start ;;
  trigger)      "$RUN" trigger ;;
  trigger_exec) "$RUN" trigger_exec ;;
  full)         "$RUN" full_feishu ;;
  *)
    echo "用法: $0 { preflight | start | trigger | trigger_exec | full }"
    echo "  preflight     - 检查 .env 与飞书配置"
    echo "  start        - 启动服务，提示后续触发命令"
    echo "  trigger      - 原有审理：POST /admin，等待飞书点击"
    echo "  trigger_exec - 新逻辑：POST /auth/exec (exec:sudo)，等待飞书点击"
    echo "  full         - 先触发原有审理，再触发新逻辑（共两次飞书点击）"
    exit 1
    ;;
esac
