#!/usr/bin/env bash
# 执行层验证脚本：构建、启动服务、调用 /auth/exec 与 /auth/sandbox-profile、运行 3af-exec、停止服务。
set -e
cd "$(dirname "$0")/.."
export CONFIG_PATH="${CONFIG_PATH:-config.example.yaml}"
# 使用临时端口避免与已有服务冲突
export DITING_PROXY_LISTEN="${DITING_PROXY_LISTEN:-:18080}"
mkdir -p ./data
go build -o bin/diting ./cmd/diting_allinone
go build -o bin/3af-exec ./cmd/3af_exec
BASE="http://127.0.0.1:18080"
./bin/diting &
PID=$!
trap "kill $PID 2>/dev/null || true" EXIT
sleep 2
# 若 18080 被占用则可能启动了 8080，尝试两者
if ! curl -s -o /dev/null -w "%{http_code}" "$BASE/healthz" 2>/dev/null | grep -q 200; then
  BASE="http://127.0.0.1:8080"
fi
echo "=== POST /auth/exec (allow) ==="
curl -s -X POST "$BASE/auth/exec" \
  -H "Content-Type: application/json" \
  -d '{"subject":"test","action":"exec:run","resource":"local://host","command_line":"echo ok"}'
echo ""
echo "=== GET /auth/sandbox-profile ==="
curl -s "$BASE/auth/sandbox-profile?resource=local://host"
echo ""
echo "=== 3af-exec echo ok ==="
DITING_3AF_URL="$BASE" ./bin/3af-exec echo ok
echo "=== 验证通过 ==="
