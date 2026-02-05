@echo off
chcp 65001 >nul
echo.
echo ╔════════════════════════════════════════════════════════╗
echo ║           Sentinel-AI 自动化测试脚本                  ║
echo ╚════════════════════════════════════════════════════════╝
echo.

echo 请确保 Sentinel-AI 已经在另一个终端运行!
echo 按任意键开始测试...
pause >nul

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试 1: 安全查询 (应该自动放行)
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
curl -X GET http://localhost:8080/get
echo.
timeout /t 3 >nul

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试 2: HEAD 请求 (应该自动放行)
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
curl -I http://localhost:8080/get
echo.
timeout /t 3 >nul

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试 3: POST 请求 (应该需要审批)
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 提示: 在 Sentinel-AI 终端输入 'n' 拒绝此请求
echo.
timeout /t 2 >nul
curl -X POST http://localhost:8080/post -H "Content-Type: application/json" -d "{\"test\": \"data\"}"
echo.
timeout /t 3 >nul

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试 4: DELETE 请求 (应该需要审批)
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 提示: 在 Sentinel-AI 终端输入 'n' 拒绝此请求
echo.
timeout /t 2 >nul
curl -X DELETE http://localhost:8080/delete
echo.
timeout /t 3 >nul

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试 5: 危险路径 (应该需要审批)
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 提示: 在 Sentinel-AI 终端输入 'n' 拒绝此请求
echo.
timeout /t 2 >nul
curl -X PUT http://localhost:8080/api/production/config -H "Content-Type: application/json" -d "{\"setting\": \"value\"}"
echo.
timeout /t 3 >nul

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试完成! 查看审计日志:
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

if exist logs\audit.jsonl (
    echo 最近 3 条审计记录:
    echo.
    powershell -Command "Get-Content logs\audit.jsonl -Tail 3 | ForEach-Object { $_ | ConvertFrom-Json | ConvertTo-Json -Compress }"
) else (
    echo 未找到审计日志文件
)

echo.
echo 按任意键退出...
pause >nul
