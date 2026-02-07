#!/bin/bash

# Diting HTTPS 代理验证脚本

echo "🔍 Diting HTTPS 代理验证"
echo "========================"
echo ""

# 检查 Go 环境
echo "1. 检查 Go 环境..."
if ! command -v go &> /dev/null; then
    echo "   ❌ Go 未安装"
    echo "   请安装 Go 1.21+: https://golang.org/dl/"
    exit 1
else
    GO_VERSION=$(go version)
    echo "   ✅ $GO_VERSION"
fi
echo ""

# 检查代码语法
echo "2. 检查代码语法..."
cd "$(dirname "$0")/../cmd/diting"
if go build -o /tmp/diting-test main.go 2>&1; then
    echo "   ✅ 代码编译成功"
    rm -f /tmp/diting-test
else
    echo "   ❌ 代码编译失败"
    exit 1
fi
echo ""

# 启动服务（后台）
echo "3. 启动 Diting 服务..."
go run main.go > /tmp/diting.log 2>&1 &
DITING_PID=$!
echo "   ✅ 服务已启动 (PID: $DITING_PID)"
sleep 3
echo ""

# 测试 HTTP 代理
echo "4. 测试 HTTP 代理..."
HTTP_RESPONSE=$(curl -s -x http://localhost:8080 http://httpbin.org/get -m 5 2>&1)
if [ $? -eq 0 ]; then
    echo "   ✅ HTTP 代理工作正常"
else
    echo "   ❌ HTTP 代理失败"
    echo "   错误: $HTTP_RESPONSE"
fi
echo ""

# 测试 HTTPS 代理
echo "5. 测试 HTTPS 代理..."
HTTPS_RESPONSE=$(curl -s -x http://localhost:8080 https://api.github.com/zen -m 5 2>&1)
if [ $? -eq 0 ]; then
    echo "   ✅ HTTPS 代理工作正常"
    echo "   响应: $HTTPS_RESPONSE"
else
    echo "   ⚠️  HTTPS 代理可能需要人工审批"
    echo "   提示: 如果服务在等待审批，请在另一个终端输入 y"
fi
echo ""

# 检查审计日志
echo "6. 检查审计日志..."
if [ -f "../../logs/audit.jsonl" ]; then
    LOG_COUNT=$(wc -l < ../../logs/audit.jsonl)
    echo "   ✅ 审计日志已创建 ($LOG_COUNT 条记录)"
    echo "   最后一条:"
    tail -n 1 ../../logs/audit.jsonl | jq '.' 2>/dev/null || tail -n 1 ../../logs/audit.jsonl
else
    echo "   ⚠️  审计日志未创建"
fi
echo ""

# 停止服务
echo "7. 停止服务..."
kill $DITING_PID 2>/dev/null
echo "   ✅ 服务已停止"
echo ""

echo "========================"
echo "✅ 验证完成"
echo ""
echo "📝 查看完整日志: cat /tmp/diting.log"
echo "📝 查看审计日志: cat logs/audit.jsonl"
