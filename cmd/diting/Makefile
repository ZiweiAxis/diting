# 在 cmd/diting 目录下可直接 make watch / make build / make run
# 或从仓库根目录执行: make watch（使用根目录 Makefile）

.PHONY: build run watch clean
build:
	go build -o bin/diting ./cmd/diting_allinone

run: build
	./bin/diting

# 清理构建产物（统一在 bin/，顺带删根目录可能残留的 diting*）
clean:
	rm -rf bin/
	rm -f diting diting-* diting_*

watch:
	@PATH="$${HOME}/go/bin:$$PATH" command -v air >/dev/null 2>&1 && PATH="$${HOME}/go/bin:$$PATH" air || (echo "未检测到 air。安装: ./scripts/install-dev-deps.sh 或从仓库根执行 make watch"; exit 1)
