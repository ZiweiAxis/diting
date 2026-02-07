@echo off
chcp 65001 >nul
echo 🚀 Sentinel-AI 快速启动脚本
echo.

REM 检查 Go 是否安装
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo ❌ 错误: 未检测到 Go 环境
    echo    请访问 https://go.dev/dl/ 下载安装
    pause
    exit /b 1
)

echo ✓ Go 环境检测通过

REM 检查 Ollama 是否运行
curl -s http://localhost:11434/api/tags >nul 2>nul
if %errorlevel% equ 0 (
    echo ✓ Ollama 服务运行中
) else (
    echo ⚠️  警告: Ollama 未运行 ^(将使用规则引擎模式^)
    echo    启动方法: ollama serve
    echo    下载模型: ollama pull qwen2.5:7b
)

echo.
echo 📦 安装依赖...
go mod download

echo.
echo 🔧 编译程序...
go build -o sentinel-ai.exe main.go

echo.
echo ✅ 启动 Sentinel-AI 治理网关...
echo.
sentinel-ai.exe
