#!/usr/bin/env bash
# 安装本地开发依赖（默认使用国内镜像，无需外网直连）
# 使用：在 cmd/diting 目录下执行 ./scripts/install-dev-deps.sh，或从仓库根：cd cmd/diting && ./scripts/install-dev-deps.sh
set -e
cd "$(dirname "$0")/.."

# 使用国内镜像，避免 proxy.golang.org 超时（可覆盖：GOPROXY=xxx ./scripts/install-dev-deps.sh）
export GOPROXY="${GOPROXY:-https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct}"

echo "[1/3] 下载 Go 模块 (GOPROXY=$GOPROXY)..."
go mod download

echo "[2/3] 整理 go.mod..."
go mod tidy

echo "[3/3] 安装 air（watch 模式）..."
go install github.com/air-verse/air@latest

AIR_PATH=$(go env GOPATH)/bin/air
if [[ -x "$AIR_PATH" ]]; then
  echo "OK air 已安装: $AIR_PATH"
  echo "请确保 \$(go env GOPATH)/bin 在 PATH 中，然后可执行: make watch 或 air"
else
  echo "警告: 未找到 air，请检查 GOPATH 并确认 PATH 包含 \$GOPATH/bin"
fi
