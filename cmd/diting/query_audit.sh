#!/usr/bin/env bash
# 审计日志查询小工具（P1）
# 用法:
#   ./query_audit.sh              # 最近 20 条
#   ./query_audit.sh --approved   # 仅通过的审批
#   ./query_audit.sh --denied     # 仅拒绝的审批
#   ./query_audit.sh -n 50        # 最近 50 条
#   ./query_audit.sh --all        # 全部

set -e
cd "$(dirname "$0")"
LOG="${AUDIT_LOG_FILE:-logs/audit.jsonl}"
if [[ ! -f "$LOG" ]]; then
  echo "审计日志不存在: $LOG"
  exit 1
fi

APPROVED=""
DENIED=""
N=20
ALL=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --approved) APPROVED=1; shift ;;
    --denied)   DENIED=1; shift ;;
    -n)         N=$2; shift 2 ;;
    --all)      ALL=1; shift ;;
    *)          shift ;;
  esac
done

if [[ -n "$ALL" ]]; then
  if [[ -n "$APPROVED" ]]; then
    jq -s 'map(select(.type == "approval_decision" and .approved == true))' "$LOG" 2>/dev/null || grep '"approved":true' "$LOG" | jq -s .
  elif [[ -n "$DENIED" ]]; then
    jq -s 'map(select(.type == "approval_decision" and .approved != true))' "$LOG" 2>/dev/null || grep '"approval_decision"' "$LOG" | grep -v '"approved":true' | jq -s .
  else
    cat "$LOG" | jq -s .
  fi
  exit 0
fi

# 默认：最近 N 条，可过滤
if [[ -n "$APPROVED" ]]; then
  tail -n 500 "$LOG" | grep '"approval_decision"' | grep '"approved":true' | tail -n "$N"
elif [[ -n "$DENIED" ]]; then
  tail -n 500 "$LOG" | grep '"approval_decision"' | grep -v '"approved":true' | tail -n "$N"
else
  tail -n "$N" "$LOG"
fi
