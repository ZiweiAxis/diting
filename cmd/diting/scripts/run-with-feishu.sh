#!/bin/bash
# 使用飞书投递启动 diting，用于验收「飞书审批流程」
# 使用前请设置：DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET、DITING_FEISHU_APPROVAL_USER_ID

cd "$(dirname "$0")"
if [ -z "$DITING_FEISHU_APP_ID" ] || [ -z "$DITING_FEISHU_APP_SECRET" ]; then
  echo "[diting] 未设置 DITING_FEISHU_APP_ID 或 DITING_FEISHU_APP_SECRET，将使用占位投递（不发飞书）"
  echo "[diting] 要看到飞书审批流程，请先执行："
  echo "  export DITING_FEISHU_APP_ID=xxxx"
  echo "  export DITING_FEISHU_APP_SECRET=***"
  echo "  export DITING_FEISHU_APPROVAL_USER_ID=xxxx"
  echo ""
fi
if [ -z "$DITING_FEISHU_APPROVAL_USER_ID" ] && [ -z "$DITING_FEISHU_CHAT_ID" ]; then
  [ -n "$DITING_FEISHU_APP_ID" ] && echo "[diting] 未设置 DITING_FEISHU_APPROVAL_USER_ID，请设置否则飞书无法发送接收人"
fi
exec ./bin/diting
