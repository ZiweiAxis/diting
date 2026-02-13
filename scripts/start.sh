#!/bin/bash

echo "🚀 Sentinel-AI 快速启动脚本"
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未检测到 Go 环境"
    echo "   请访问 https://go.dev/dl/ 下载安装"
    exit 1
fi

echo "✓ Go 环境检测通过"

# 检查 Ollama 是否运行
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "✓ Ollama 服务运行中"
else
    echo "⚠️  警告: Ollama 未运行 (将使用规则引擎模式)"
    echo "   启动方法: ollama serve"
    echo "   下载模型: ollama pull qwen2.5:7b"
fi

echo ""
echo "📦 安装依赖..."
go mod download

echo ""
echo "🔧 编译程序..."
go build -o diting main.go

echo ""
echo "✅ 启动 Sentinel-AI 治理网关..."
echo ""
./diting
