# Diting All-in-One 构建与运行（Phase 2）
# 默认使用 cmd/diting 下的 All-in-One 入口
DITING_DIR := cmd/diting
BINARY := $(DITING_DIR)/bin/diting

.PHONY: build run watch watch-entr clean docker-diting
build:
	cd $(DITING_DIR) && go build -o bin/diting ./cmd/diting_allinone

run: build
	cd $(DITING_DIR) && ./bin/diting

# 清理 diting 构建产物（bin/ 及 cmd/diting 根目录残留）
clean:
	rm -rf $(DITING_DIR)/bin/
	rm -f $(DITING_DIR)/diting $(DITING_DIR)/diting-* $(DITING_DIR)/diting_*

# Watch 模式（优先 air）：代码/配置变更自动重新编译并重启（自动带 $HOME/go/bin）
watch:
	@PATH="$${HOME}/go/bin:$$PATH" command -v air >/dev/null 2>&1 && (cd $(DITING_DIR) && PATH="$${HOME}/go/bin:$$PATH" air) || (echo "未检测到 air。安装: cd cmd/diting && ./scripts/install-dev-deps.sh"; echo "或使用: make watch-entr（需系统有 entr）"; exit 1)

# Watch 备用：使用 entr 监听文件变化（多数 Linux 已带或 apt install entr）
watch-entr:
	@command -v entr >/dev/null 2>&1 || { echo "未检测到 entr。安装: apt install entr 或 brew install entr"; exit 1; }
	cd $(DITING_DIR) && find . -path ./bin -prune -o -path ./tmp -prune -o -path ./data -prune -o \( -name '*.go' -o -name '*.yaml' -o -name '*.yml' \) -print | entr -r -s 'go build -o bin/diting ./cmd/diting_allinone && ./bin/diting'

# 构建 Diting All-in-One 镜像（需在仓库根目录执行，且已安装 Docker）
docker-diting:
	docker build -f deployments/docker/Dockerfile.diting -t diting:latest .
