#!/bin/bash

# 飞书 WebSocket 长连接快速修复脚本
# 使用方法: ./quick_fix.sh

set -e

echo "========================================="
echo "飞书 WebSocket 长连接快速修复"
echo "========================================="
echo ""

PROJECT_DIR="/home/dministrator/workspace/sentinel-ai/cmd/diting"
cd "$PROJECT_DIR"

# 1. 运行诊断
echo "步骤 1: 运行诊断工具"
echo "-----------------------------------------"
if [ -f "diagnose_feishu.sh" ]; then
    ./diagnose_feishu.sh
else
    echo "❌ 诊断工具不存在"
    exit 1
fi

echo ""
read -p "诊断结果是否显示 404? (y/n): " is_404

if [ "$is_404" = "y" ] || [ "$is_404" = "Y" ]; then
    echo ""
    echo "⚠️  检测到 404 错误"
    echo ""
    echo "需要在飞书开放平台启用长连接："
    echo "1. 访问: https://open.feishu.cn/app"
    echo "2. 选择应用: cli_a90d5a960cf89cd4"
    echo "3. 进入「事件订阅」-> 「长连接」"
    echo "4. 启用长连接功能"
    echo "5. 添加事件: im.message.receive_v1"
    echo ""
    read -p "已完成配置? (y/n): " configured
    
    if [ "$configured" != "y" ] && [ "$configured" != "Y" ]; then
        echo "请先完成配置后再运行此脚本"
        exit 0
    fi
    
    echo ""
    echo "重新运行诊断..."
    ./diagnose_feishu.sh
fi

echo ""
echo "步骤 2: 备份原文件"
echo "-----------------------------------------"
if [ -f "main_ws.go" ]; then
    if [ ! -f "main_ws.backup.go" ]; then
        cp main_ws.go main_ws.backup.go
        echo "✅ 已备份: main_ws.backup.go"
    else
        echo "⚠️  备份文件已存在，跳过"
    fi
fi

echo ""
echo "步骤 3: 应用修复"
echo "-----------------------------------------"
if [ -f "main_ws_fixed.go" ]; then
    cp main_ws_fixed.go main_ws.go
    echo "✅ 已应用修复版本"
else
    echo "❌ 修复文件不存在: main_ws_fixed.go"
    exit 1
fi

echo ""
echo "步骤 4: 验证修复"
echo "-----------------------------------------"
echo "请手动运行以下命令测试："
echo ""
echo "  cd $PROJECT_DIR"
echo "  go run main_ws.go"
echo ""
echo "期望看到："
echo "  ✓ 获取 endpoint 成功"
echo "  ✓ WebSocket 连接已建立"
echo ""

echo "========================================="
echo "修复完成！"
echo "========================================="
echo ""
echo "如果仍有问题，请查看:"
echo "  - FEISHU_WEBSOCKET_FIX.md (详细文档)"
echo "  - TEST_REPORT.md (测试报告)"
echo ""
