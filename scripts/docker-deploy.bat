@echo off
chcp 65001 >nul
echo 🚀 Sentinel-AI Docker 快速部署
echo ================================
echo.

REM 检查 Docker 是否运行
docker info >nul 2>nul
if %errorlevel% neq 0 (
    echo ❌ 错误: Docker 未运行
    echo    请启动 Docker Desktop
    pause
    exit /b 1
)

echo ✓ Docker 运行正常
echo.

echo 请选择部署模式:
echo   1. Python 版本 ^(MVP，推荐快速测试^)
echo   2. Go 版本 ^(高性能，生产推荐^)
echo   3. 完整部署 ^(Web + Ollama + PostgreSQL^)
echo   4. 停止所有服务
echo   5. 查看日志
echo.
set /p choice="请输入选项 (1-5): "

if "%choice%"=="1" goto python
if "%choice%"=="2" goto go
if "%choice%"=="3" goto full
if "%choice%"=="4" goto stop
if "%choice%"=="5" goto logs

echo ❌ 无效选项
pause
exit /b 1

:python
echo.
echo 📦 构建 Python 版本...
docker-compose build python-base
echo.
echo ✅ 启动 Python 版本...
docker-compose --profile python up -d
echo.
echo ✓ Sentinel-AI 已启动
echo   代理地址: http://localhost:8080
echo   查看日志: docker-compose logs -f sentinel-python
pause
exit /b 0

:go
echo.
echo 📦 构建 Go 版本...
docker-compose build alpine
echo.
echo ✅ 启动 Go 版本...
docker-compose --profile go up -d
echo.
echo ✓ Sentinel-AI 已启动
echo   代理地址: http://localhost:8080
echo   查看日志: docker-compose logs -f sentinel-go
pause
exit /b 0

:full
echo.
echo 📦 构建所有服务...
docker-compose build
echo.
echo ✅ 启动完整部署...
docker-compose --profile python --profile ollama --profile postgres --profile redis up -d
echo.
echo ✓ Sentinel-AI 完整部署已启动
echo   代理地址: http://localhost:8080
echo   Web 界面: http://localhost:8081
echo   Ollama API: http://localhost:11434
echo.
echo   下载 Ollama 模型:
echo     docker exec ollama ollama pull qwen2.5:7b
pause
exit /b 0

:stop
echo.
echo ⏹️  停止所有服务...
docker-compose down
echo.
echo ✓ 所有服务已停止
pause
exit /b 0

:logs
echo.
echo 📋 可用的服务:
docker-compose ps
echo.
set /p service="输入服务名称查看日志 (默认 sentinel-python): "
if "%service%"=="" set service=sentinel-python
echo.
echo 📜 日志 ^(Ctrl+C 退出^):
docker-compose logs -f %service%
pause
exit /b 0
