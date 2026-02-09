# Diting All-in-One 构建与运行（Phase 2）
# 默认使用 cmd/diting 下的 All-in-One 入口
DITING_DIR := cmd/diting
BINARY := $(DITING_DIR)/bin/diting

.PHONY: build run docker-diting
build:
	cd $(DITING_DIR) && go build -o bin/diting ./cmd/diting_allinone

run: build
	cd $(DITING_DIR) && ./bin/diting -config config.example.yaml

# 构建 Diting All-in-One 镜像（需在仓库根目录执行，且已安装 Docker）
docker-diting:
	docker build -f deployments/docker/Dockerfile.diting -t diting:latest .
