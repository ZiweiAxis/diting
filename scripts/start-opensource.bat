@echo off
chcp 65001 >nul
echo.
echo ==========================================
echo  Sentinel-AI 开源版本启动脚本
echo  基于 CoreDNS + Nginx + OpenAI API
echo ==========================================
echo.

REM 检查 Docker
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Docker 未运行或未安装
    echo 请先启动 Docker Desktop
    pause
    exit /b 1
)

echo [OK] Docker 运行正常
echo.

REM 检查环境变量
if not exist .env (
    echo [警告] .env 文件不存在
    echo 从 .env.example 创建 .env
    copy .env.example .env
    echo.
    echo 请编辑 .env 文件，设置你的 OPENAI_API_KEY
    pause
    exit /b 0
)

echo [OK] 环境变量文件存在
echo.

REM 启动服务
echo.
echo ==========================================
echo  启动 Sentinel-AI 服务...
echo ==========================================
echo.

docker-compose -f docker-compose-opensource.yml up -d

echo.
echo ==========================================
echo  服务启动完成！
echo ==========================================
echo.
echo 服务地址:
echo   - DNS: 10.0.0.1:53
echo   - WAF: http://localhost:8080
echo   - API: http://localhost:8000
echo.
echo 健康检查:
echo   - WAF: http://localhost:8080/health
echo   - API: http://localhost:8000/health
echo.
echo 查看日志:
echo   - WAF: docker logs -f nginx-waf
echo   - API: docker logs -f sentinel-api
echo.
echo 停止服务:
echo   docker-compose -f docker-compose-opensource.yml down
echo.
pause
