@echo off
chcp 65001 >nul
echo.
echo ╔════════════════════════════════════════════════════════╗
echo ║      Sentinel-AI 快速启动 - Python 版本               ║
echo ╚════════════════════════════════════════════════════════╝
echo.

REM 检查 Python 是否安装
where python >nul 2>nul
if %errorlevel% neq 0 (
    echo ❌ 错误: 未检测到 Python 环境
    echo    请访问 https://www.python.org/downloads/ 下载安装
    pause
    exit /b 1
)

echo ✓ Python 环境检测通过
echo.

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
echo 📦 安装 Python 依赖...
python -m pip install -q -r requirements.txt

echo.
echo ✅ 启动 Sentinel-AI 治理网关...
echo.
python sentinel.py
