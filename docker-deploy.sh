#!/bin/bash
# Sentinel-AI Docker 快速部署脚本

echo "🚀 Sentinel-AI Docker 快速部署"
echo "================================"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker 未运行"
    echo "   请启动 Docker Desktop"
    exit 1
fi

echo "✓ Docker 运行正常"
echo ""

# 显示部署选项
echo "请选择部署模式:"
echo "  1. Python 版本 (MVP，推荐快速测试)"
echo "  2. Go 版本 (高性能，生产推荐)"
echo "  3. eBPF 版本 (内核级监控，需要特权)"
echo "  4. 完整部署 (Web + Ollama + PostgreSQL)"
echo "  5. 停止所有服务"
echo "  6. 查看日志"
echo ""
read -p "请输入选项 (1-6): " choice

case $choice in
    1)
        echo ""
        echo "📦 构建 Python 版本..."
        docker-compose build python-base
        echo ""
        echo "✅ 启动 Python 版本..."
        docker-compose --profile python up -d
        echo ""
        echo "✓ Sentinel-AI 已启动"
        echo "  代理地址: http://localhost:8080"
        echo "  查看日志: docker-compose logs -f sentinel-python"
        ;;
    
    2)
        echo ""
        echo "📦 构建 Go 版本..."
        docker-compose build alpine
        echo ""
        echo "✅ 启动 Go 版本..."
        docker-compose --profile go up -d
        echo ""
        echo "✓ Sentinel-AI 已启动"
        echo "  代理地址: http://localhost:8080"
        echo "  查看日志: docker-compose logs -f sentinel-go"
        ;;
    
    3)
        echo ""
        echo "⚠️  eBPF 版本需要:"
        echo "   1. 主机是 Linux"
        echo "   2. root 权限"
        echo "   3. 内核版本 >= 4.10"
        echo ""
        read -p "继续吗? (y/n): " confirm
        if [ "$confirm" = "y" ]; then
            echo ""
            echo "📦 构建 eBPF 版本..."
            docker-compose build ebpf-base
            echo ""
            echo "✅ 启动 eBPF 版本..."
            docker-compose --profile ebpf up -d
            echo ""
            echo "✓ Sentinel-AI eBPF 已启动"
            echo "  查看日志: docker-compose logs -f sentinel-ebpf"
        fi
        ;;
    
    4)
        echo ""
        echo "📦 构建所有服务..."
        docker-compose build
        echo ""
        echo "✅ 启动完整部署..."
        docker-compose --profile python --profile ollama --profile postgres --profile redis up -d
        echo ""
        echo "✓ Sentinel-AI 完整部署已启动"
        echo "  代理地址: http://localhost:8080"
        echo "  Web 界面: http://localhost:8081"
        echo "  Ollama API: http://localhost:11434"
        echo ""
        echo "  下载 Ollama 模型:"
        echo "    docker exec ollama ollama pull qwen2.5:7b"
        ;;
    
    5)
        echo ""
        echo "⏹️  停止所有服务..."
        docker-compose down
        echo ""
        echo "✓ 所有服务已停止"
        ;;
    
    6)
        echo ""
        echo "📋 可用的服务:"
        docker-compose ps
        echo ""
        read -p "输入服务名称查看日志 (默认 sentinel-python): " service
        service=${service:-sentinel-python}
        echo ""
        echo "📜 日志 (Ctrl+C 退出):"
        docker-compose logs -f $service
        ;;
    
    *)
        echo "❌ 无效选项"
        exit 1
        ;;
esac

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "测试命令:"
echo "  curl http://localhost:8080/health"
echo "  curl -X DELETE http://localhost:8080/delete"
echo ""
