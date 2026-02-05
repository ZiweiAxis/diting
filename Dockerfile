# Sentinel-AI 多阶段构建 Dockerfile
# 支持两种运行模式: Python MVP 和 Go 高性能版

ARG VERSION=1.0.0
ARG BUILD_DATE

# ============ 阶段 1: Python 版本 ============
FROM python:3.12-slim as python-base

WORKDIR /app

# 安装系统依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 安装 Python 依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制代码
COPY sentinel.py .
COPY config/ ./config/

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动命令
CMD ["python", "sentinel.py"]

# ============ 阶段 2: Go 高性能版本 ============
FROM golang:1.21-alpine AS go-builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
    -o sentinel-go ./main.go

# ============ 阶段 3: Go 运行时镜像 ============
FROM alpine:3.19

WORKDIR /app

# 创建非 root 用户
RUN addgroup -g 1000 sentinel && \
    adduser -D -u 1000 -G sentinel sentinel

# 从构建阶段复制二进制文件
COPY --from=go-builder /build/sentinel-go /app/sentinel-ai

# 创建日志目录并设置权限
RUN mkdir -p /app/logs && \
    chown -R sentinel:sentinel /app

# 切换到非 root 用户
USER sentinel

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# 启动命令
CMD ["/app/sentinel-ai"]

# ============ 阶段 4: eBPF 版本 (需要特权) ============
FROM ubuntu:22.04 as ebpf-base

WORKDIR /app

# 安装 eBPF 依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    bpfcc-tools \
    linux-headers-generic \
    clang \
    llvm \
    libbpf-dev \
    make \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 安装 Python BCC
RUN pip3 install --no-cache-dir bcc

# 复制 eBPF 代码
COPY sentinel_ebpf.py .

# 创建日志目录
RUN mkdir -p /app/logs

# eBPF 版本需要特权模式，无法做健康检查
EXPOSE 8080

# 启动命令 (需要 --privileged 运行)
CMD ["python3", "sentinel_ebpf.py"]
