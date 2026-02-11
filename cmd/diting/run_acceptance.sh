#!/usr/bin/env bash
# 闭环验收脚本：启动服务、检查日志、触发审批请求（人工在飞书点击）
set -e
cd "$(dirname "$0")"
LOG_FILE="${LOG_FILE:-/tmp/diting_acceptance.log}"
BINARY="${BINARY:-./bin/diting}"
CONFIG="${CONFIG:-}"
LISTEN="${LISTEN:-:8080}"

# 读取 .env 里的 KEY=VALUE（去掉引号）；不存在或空返回空字符串
get_env_val() {
  local key="$1"
  if [[ -f .env ]]; then
    grep "^${key}=" .env 2>/dev/null | sed "s/^${key}=//" | tr -d '"' | tr -d "'" | head -1
  fi
}

# 推断有效的配置文件路径（与二进制默认逻辑一致：优先 config.yaml，其次 config.example.yaml）
effective_config() {
  if [[ -n "$CONFIG" ]]; then
    echo "$CONFIG"
    return
  fi
  if [[ -f "config.yaml" ]]; then
    echo "config.yaml"
  else
    echo "config.example.yaml"
  fi
}

# 飞书闭环验收预检：确保「能发出飞书审批消息」的必要条件满足
preflight() {
  echo "[验收] 预检：飞书与 CHEQ 配置..."
  local fail=0
  local cfg
  cfg="$(effective_config)"
  if [[ ! -f "$cfg" ]]; then
    echo "  ✗ 配置文件不存在: $cfg"
    fail=1
  fi
  if [[ ! -f .env ]]; then
    echo "  ✗ 缺少 .env（请复制 .env.example 为 .env 并填写飞书参数）"
    fail=1
  fi
  local app_id app_secret approval_user chat_id
  app_id="${DITING_FEISHU_APP_ID:-$(get_env_val DITING_FEISHU_APP_ID)}"
  app_secret="${DITING_FEISHU_APP_SECRET:-$(get_env_val DITING_FEISHU_APP_SECRET)}"
  approval_user="${DITING_FEISHU_APPROVAL_USER_ID:-$(get_env_val DITING_FEISHU_APPROVAL_USER_ID)}"
  chat_id="${DITING_FEISHU_CHAT_ID:-$(get_env_val DITING_FEISHU_CHAT_ID)}"
  if [[ -z "$app_id" ]]; then
    echo "  ✗ 未设置 DITING_FEISHU_APP_ID"
    fail=1
  fi
  if [[ -z "$app_secret" ]]; then
    echo "  ✗ 未设置 DITING_FEISHU_APP_SECRET"
    fail=1
  fi
  if [[ -z "$approval_user" && -z "$chat_id" ]]; then
    echo "  ✗ 未设置 DITING_FEISHU_APPROVAL_USER_ID 或 DITING_FEISHU_CHAT_ID（审批消息发往何处）"
    fail=1
  fi
  if [[ -f "$cfg" ]]; then
    # 飞书 E2E 必须启用 CHEQ 持久化（否则 review 会走占位行为或无法形成闭环等待）
    if grep -q 'persistence_path: *""' "$cfg" 2>/dev/null || ! grep -q 'persistence_path:.*[./a-zA-Z0-9]' "$cfg" 2>/dev/null; then
      echo "  ✗ 飞书 E2E 必须配置 cheq.persistence_path 为非空（例如 ./data/cheq）"
      fail=1
    fi
  fi
  if [[ "$fail" -eq 1 ]]; then
    echo ""
    echo "修复建议："
    echo "  - cp .env.example .env"
    echo "  - 在 .env 中填写 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET、DITING_FEISHU_APPROVAL_USER_ID（或 DITING_FEISHU_CHAT_ID）"
    echo "  - 确保 config 中 cheq.persistence_path 非空（如 ./data/cheq）"
    exit 1
  fi
  echo "  ✓ 预检通过"
}

# 查找占用 $port 的 PID（兼容 8080 或 :8080）
get_pid_on_port() {
  local port="${1:-8080}"
  port="${port#:}"
  (ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null) | awk -v p=":${port}" '$4 ~ p { gsub(/.*pid=/,""); sub(/,.*/,""); if ($0 != "") print; exit }'
}

start_server() {
  preflight
  echo "[验收] 释放端口 8080..."
  local pid
  pid=$(get_pid_on_port 8080)
  if [[ -n "$pid" ]]; then
    kill "$pid" 2>/dev/null || true
    sleep 2
  fi
  # 验收时强烈建议使用最新二进制，避免“代码已修复但仍在跑旧 bin/diting”导致的误判。
  # 如需跳过编译，可设置 SKIP_BUILD=1。
  if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
    echo "[验收] 编译最新二进制..."
    go build -o bin/diting ./cmd/diting_allinone
  elif [[ ! -f "$BINARY" ]]; then
    echo "[验收] SKIP_BUILD=1 但未找到 $BINARY，无法继续。"
    exit 1
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
    echo "[验收] 提示：若出现 open_id cross app，请改用该应用下的 user_id，或配置 DITING_FEISHU_CHAT_ID 发到群聊。"
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
  echo "  原有审理: $0 trigger        # POST /admin → 飞书审批"
  echo "  新逻辑:   $0 trigger_exec   # POST /auth/exec (exec:sudo) → 飞书审批"
  echo ""
}

trigger_request() {
  echo "[验收] 触发 POST /admin（原有审理：HTTP 代理，等待审批，最多 125 秒）..."
  curl -s -X POST "http://localhost:8080/admin" -H "Host: example.com" -d '{}' --max-time 125 -w "\nHTTP %{http_code}\n" -o /tmp/diting_acceptance_response.txt || true
  echo "[验收] 响应："
  cat /tmp/diting_acceptance_response.txt
}

# 触发执行层审批（新逻辑）：POST /auth/exec 且 action=exec:sudo → review → 飞书卡片，与原有审理共用 CHEQ/飞书
trigger_exec_request() {
  echo "[验收] 触发 POST /auth/exec（新逻辑：执行层审批，exec:sudo → review，等待审批，最多 125 秒）..."
  curl -s -X POST "http://localhost:8080/auth/exec" \
    -H "Content-Type: application/json" \
    -d '{"subject":"test","action":"exec:sudo","resource":"local://host","command_line":"sudo echo ok"}' \
    --max-time 125 -w "\nHTTP %{http_code}\n" -o /tmp/diting_acceptance_exec_response.txt || true
  echo "[验收] 响应："
  cat /tmp/diting_acceptance_exec_response.txt
}

full_flow() {
  start_server
  echo ""
  echo "[验收] 已启动服务，接下来将触发审批请求；请在 120 秒内到飞书点击批准/拒绝。"
  echo ""
  trigger_request
  echo ""
  echo "[验收] 若需要停止服务：$0 stop"
}

# 飞书双路径验收：先触发原有审理（/admin），等用户点击后再触发新逻辑（/auth/exec exec:sudo）
full_flow_feishu() {
  start_server
  echo ""
  echo "[验收] === Phase 1：原有审理（HTTP 代理 POST /admin）==="
  echo "[验收] 请在收到飞书卡片后点击「批准」或「拒绝」，请求将在点击后返回。"
  echo ""
  trigger_request
  echo ""
  echo "[验收] === Phase 2：新逻辑审批（执行层 POST /auth/exec exec:sudo）==="
  echo "[验收] 请再次到飞书点击新卡片的「批准」或「拒绝」。"
  echo ""
  trigger_exec_request
  echo ""
  echo "[验收] 飞书双路径验收完成。若需查审计：./query_audit.sh -n 5 或按 trace_id 查询。"
  echo "[验收] 若需要停止服务：$0 stop"
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
  preflight)      preflight ;;
  start)          start_server ;;
  trigger)        trigger_request ;;
  trigger_exec)   trigger_exec_request ;;
  full)           full_flow ;;
  full_feishu)    full_flow_feishu ;;
  stop)           stop_server ;;
  *)
    echo "用法: $0 { preflight | start | trigger | trigger_exec | full | full_feishu | stop }"
    echo "  preflight    - 检查 .env 与配置是否满足飞书收消息"
    echo "  start        - 释放 8080、启动服务并提示触发命令"
    echo "  trigger      - 原有审理：POST /admin，等待飞书点击"
    echo "  trigger_exec - 新逻辑：POST /auth/exec (exec:sudo)，等待飞书点击"
    echo "  full         - start + trigger（需在 125 秒内到飞书点击批准/拒绝）"
    echo "  full_feishu  - 飞书双路径验收：先 trigger 再 trigger_exec（两次飞书点击）"
    echo "  stop         - 停止验收启动的服务"
    exit 1
    ;;
esac
