#!/bin/bash

echo "=== 谛听飞书集成测试 ==="
echo ""

# 检查配置
echo "1. 检查配置文件..."
if [ -f config.json ]; then
    echo "   ✓ config.json 存在"
else
    echo "   ✗ config.json 不存在"
    exit 1
fi

# 检查可执行文件
echo "2. 检查可执行文件..."
if [ -f diting_feishu ]; then
    echo "   ✓ diting_feishu 存在"
else
    echo "   ✗ diting_feishu 不存在"
    exit 1
fi

echo ""
echo "=== 准备启动 Diting ==="
echo ""
echo "启动命令: ./diting_feishu"
echo ""
echo "测试步骤："
echo "1. 启动 Diting: ./diting_feishu"
echo "2. 新终端测试低风险: curl -x http://localhost:8081 http://httpbin.org/get"
echo "3. 新终端测试高风险: curl -x http://localhost:8081 -X DELETE http://httpbin.org/delete"
echo "4. 在飞书中回复 '批准' 或 '拒绝'"
echo ""
echo "按 Enter 启动..."
read

./diting_feishu
