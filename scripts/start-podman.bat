@echo off
chcp 65001 >nul
echo.
echo ==========================================
echo    Sentinel-AI Podman 启动脚本
echo    基于 CoreDNS + Nginx + OpenAI API
echo ==========================================
echo.

REM 检查 Podman
podman --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Podman 未运行或未安装
    echo    请先启动 Podman Desktop
    echo    下载: https://podman.io/
    pause
    exit /b 1
)

echo [OK] Podman 运行正常
echo.

REM 检查环境变量
if not exist .env (
    echo [警告] .env 文件不存在
    echo    从 .env.example 创建 .env
    copy .env.example .env
    echo.
    echo    请编辑 .env 文件，设置你的 OPENAI_API_KEY
    echo.
    pause
    exit /b 0
)

echo [OK] 环境变量文件存在
echo.

REM 检查配置文件
if not exist coredns/Corefile (
    echo [错误] coredns/Corefile 不存在
    pause
    exit /b 1
)

if not exist nginx/nginx.conf (
    echo [错误] nginx/nginx.conf 不存在
    pause
    exit /b 1
)

if not exist sentinel-api/main.py (
    echo [错误] sentinel-api/main.py 不存在
    pause
    exit /b 1
)

echo [OK] 所有配置文件存在
echo.

echo.
echo ==========================================
echo    启动 Sentinel-AI 服务...
echo ==========================================
echo.

REM 拉起所有容器
podman-compose -f podman-compose.yml up -d

echo.
echo ==========================================
echo    服务启动完成！
echo ==========================================
echo.
echo.
echo 服务地址:
echo   - DNS: 10.0.0.1:53
echo   - WAF: http://localhost:8080
echo   - API: http://localhost:8000
echo.
echo 查看日志:
echo   - DNS: podman logs -f sentinel-coredns
echo   - WAF: podman logs -f sentinel-nginx
echo   - API: podman logs -f sentinel-api
echo.
echo 停止服务:
echo   - podman-compose -f podman-compose.yml down
echo.
echo.
pause
