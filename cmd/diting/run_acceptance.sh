#!/usr/bin/env bash
# 闭环验收脚本：启动服务、检查日志、触发审批请求（人工在飞书点击）
set -e
cd "$(dirname "$0")"
LOG_FILE="${LOG_FILE:-/tmp/diting_acceptance.log}"
BINARY="${BINARY:-./bin/diting}"
CONFIG="${CONFIG:-}"
LISTEN="${LISTEN:-:8080}"

# 查找占用 $port 的 PID（兼容 8080 或 :8080）
get_pid_on_port() {
  local port="${1:-8080}"
  port="${port#:}"
  (ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null) | awk -v p=":${port}" '$4 ~ p { gsub(/.*pid=/,""); sub(/,.*/,""); if ($0 != "") print; exit }'
}

start_server() {
  echo "[验收] 释放端口 8080..."
  local pid
  pid=$(get_pid_on_port 8080)
  if [[ -n "$pid" ]]; then
    kill "$pid" 2>/dev/null || true
    sleep 2
  fi
  if [[ ! -f "$BINARY" ]]; then
    echo "[验收] 未找到 $BINARY，正在编译..."
    go build -o bin/diting ./cmd/diting_allinone
  fi
  if [[ -n "$CONFIG" ]]; then
    echo "[验收] 启动服务：$BINARY -config $CONFIG（日志: $LOG_FILE）"
    nohup "$BINARY" -config "$CONFIG" >> "$LOG_FILE" 2>&1 &
  else
    echo "[验收] 启动服务：$BINARY（使用默认 config.yaml / config.example.yaml，日志: $LOG_FILE）"
    nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
  fi
  echo $! > /tmp/diting_acceptance.pid
  sleep 5
  if ! grep -q "飞书投递已启用" "$LOG_FILE" 2>/dev/null; then
    echo "[验收] 警告：日志中未看到「飞书投递已启用」，请检查 .env 与飞书配置。"
  else
    echo "[验收] 已看到「飞书投递已启用」。"
  fi
  if grep -q "长连接已建立\|长连接已启动" "$LOG_FILE" 2>/dev/null; then
    echo "[验收] 飞书长连接已就绪。"
  else
    echo "[验收] 若启用长连接，稍等几秒后日志应出现「飞书长连接已建立」。"
  fi
  echo ""
  echo "请在同一机器另一终端执行触发请求（在 120 秒内到飞书点击批准/拒绝）："
  echo "  curl -s -X POST 'http://localhost:8080/admin' -H 'Host: example.com' -d '{}' -w '\\nHTTP %{http_code}\\n'"
  echo ""
  echo "或在本脚本中执行: $0 trigger"
}

trigger_request() {
  echo "[验收] 触发 POST /admin（等待审批，最多 125 秒）..."
  curl -s -X POST "http://localhost:8080/admin" -H "Host: example.com" -d '{}' --max-time 125 -w "\nHTTP %{http_code}\n" -o /tmp/diting_acceptance_response.txt || true
  echo "[验收] 响应："
  cat /tmp/diting_acceptance_response.txt
}

stop_server() {
  if [[ -f /tmp/diting_acceptance.pid ]]; then
    local pid
    pid=$(cat /tmp/diting_acceptance.pid)
    kill "$pid" 2>/dev/null || true
    rm -f /tmp/diting_acceptance.pid
    echo "[验收] 已停止服务 (PID $pid)。"
  else
    pkill -f "diting_allinone.*acceptance" 2>/dev/null || true
    echo "[验收] 已尝试停止相关进程。"
  fi
}

case "${1:-start}" in
  start)  start_server ;;
  trigger) trigger_request ;;
  stop)   stop_server ;;
  *)
    echo "用法: $0 { start | trigger | stop }"
    echo "  start   - 释放 8080、启动服务并提示触发命令"
    echo "  trigger - 发送 POST /admin 等待审批（需先在飞书点击）"
    echo "  stop    - 停止验收启动的服务"
    exit 1
    ;;
esac
