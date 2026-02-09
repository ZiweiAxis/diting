#!/bin/bash

echo "=== 谛听飞书集成测试 ==="
echo ""

# 检查配置（收敛后：config.yaml + .env）
echo "1. 检查配置文件..."
if [ -f config.yaml ] || [ -f config.example.yaml ]; then
    echo "   ✓ config.yaml 或 config.example.yaml 存在"
else
    echo "   ✗ 请复制 config.example.yaml 为 config.yaml"
    exit 1
fi
if [ -f .env ]; then
    echo "   ✓ .env 存在"
else
    echo "   ⚠ .env 不存在，请 cp .env.example .env 并填写 DITING_FEISHU_*"
fi

# 检查可执行文件（推荐 All-in-One）
echo "2. 检查可执行文件..."
if [ -f bin/diting ]; then
    echo "   ✓ bin/diting (All-in-One) 存在"
    BINARY="./bin/diting"
elif [ -f diting_feishu ]; then
    echo "   ✓ diting_feishu 存在（旧入口，现也使用 config.yaml + .env）"
    BINARY="./diting_feishu"
else
    echo "   ✗ 请先执行: make build 或 go build -o bin/diting ./cmd/diting_allinone"
    exit 1
fi

echo ""
echo "=== 准备启动 Diting ==="
echo ""
echo "启动命令: $BINARY"
echo ""
echo "测试步骤："
echo "1. 启动 Diting: $BINARY"
echo "2. 新终端触发审批: curl -v http://localhost:8080/admin"
echo "3. 在飞书中点击卡片的「批准」或「拒绝」"
echo ""
echo "按 Enter 启动..."
read

$BINARY
