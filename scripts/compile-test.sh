#!/bin/bash

# 编译测试脚本

echo "🔨 Diting 编译测试"
echo "=================="
echo ""

cd "$(dirname "$0")/../cmd/diting"

# 1. 检查 Go 环境
echo "1. 检查 Go 环境..."
if ! command -v go &> /dev/null; then
    echo "   ❌ Go 未安装"
    echo "   请安装 Go 1.21+: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "   ✅ Go 版本: $GO_VERSION"
echo ""

# 2. 初始化 Go 模块
echo "2. 初始化 Go 模块..."
if [ ! -f "go.mod" ]; then
    echo "   创建 go.mod..."
    go mod init github.com/hulk-yin/diting/cmd/diting
fi
echo "   ✅ go.mod 已存在"
echo ""

# 3. 下载依赖
echo "3. 下载依赖..."
go mod tidy
if [ $? -eq 0 ]; then
    echo "   ✅ 依赖下载成功"
else
    echo "   ❌ 依赖下载失败"
    exit 1
fi
echo ""

# 4. 编译代码
echo "4. 编译代码..."
go build -o diting main.go
if [ $? -eq 0 ]; then
    echo "   ✅ 编译成功"
    ls -lh diting
else
    echo "   ❌ 编译失败"
    exit 1
fi
echo ""

# 5. 检查可执行文件
echo "5. 检查可执行文件..."
if [ -x "diting" ]; then
    echo "   ✅ 可执行文件已生成"
    file diting
else
    echo "   ❌ 可执行文件不存在或无执行权限"
    exit 1
fi
echo ""

echo "=================="
echo "✅ 编译测试通过"
echo ""
echo "运行服务: ./diting"
echo "或: go run main.go"
