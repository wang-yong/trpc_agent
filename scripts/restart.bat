@echo off
chcp 65001 >nul 2>&1
cd /d "%~dp0\.."

echo ========================================
echo   trpc_agent Web 服务重启
echo ========================================
echo.

echo [1/2] 停止现有服务...
call scripts\stop.bat

REM 短暂等待端口释放（用 ping 替代 timeout）
ping -n 2 127.0.0.1 >nul 2>&1

echo [2/2] 重新启动服务...
call scripts\start.bat
