@echo off
chcp 65001 >nul 2>&1
setlocal enabledelayedexpansion
cd /d "%~dp0\.."

echo ========================================
echo   trpc_agent Web 服务停止
echo ========================================

REM 尝试从 .env 读取端口
set PORT=8080
if exist .env (
    for /f "usebackq eol=# tokens=1,* delims==" %%a in (".env") do (
        if /i "%%a"=="SERVER_ADDR" if not "%%b"=="" (
            for /f "tokens=2 delims=:" %%p in ("%%b") do set PORT=%%p
        )
    )
)

set FOUND=0

REM 查找并终止监听该端口的进程（去重 PID）
set LAST_PID=
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":%PORT% " ^| findstr "LISTENING"') do (
    if "%%a" neq "!LAST_PID!" (
        set LAST_PID=%%a
        echo 正在终止进程 PID=%%a ...
        taskkill /PID %%a /F >nul 2>&1
        if !errorlevel!==0 (
            echo [成功] 已终止进程 PID=%%a
            set FOUND=1
        ) else (
            echo [跳过] 进程 PID=%%a 可能已退出
        )
    )
)

if "!FOUND!"=="0" (
    echo [提示] 未发现运行中的 Web 服务（端口 %PORT% 无监听）
) else (
    echo.
    echo [完成] Web 服务已停止
)
echo.
endlocal
