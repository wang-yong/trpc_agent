@echo off
chcp 65001 >nul 2>&1
setlocal enabledelayedexpansion
cd /d "%~dp0\.."

echo ========================================
echo   trpc_agent Web 服务启动
echo ========================================

REM 加载 .env 环境变量
if not exist .env (
    echo [错误] 未找到 .env 配置文件，请先创建！
    exit /b 1
)

for /f "usebackq eol=# tokens=1,* delims==" %%a in (".env") do (
    if not "%%b"=="" set "%%a=%%b"
)

REM 确认 API Key
if "%OPENAI_API_KEY%"=="" (
    echo [错误] .env 中未设置 OPENAI_API_KEY
    exit /b 1
)

REM 确认 bin 目录存在
if not exist bin mkdir bin

REM 确认 bin/log 目录存在
if not exist bin\log mkdir bin\log

REM 解析端口
set PORT=8080
if not "%SERVER_ADDR%"=="" (
    for /f "tokens=2 delims=:" %%p in ("%SERVER_ADDR%") do set PORT=%%p
)

REM 检查端口是否已被占用
netstat -ano | findstr ":%PORT% " | findstr "LISTENING" >nul 2>&1
if !errorlevel!==0 (
    echo [警告] 端口 %PORT% 已被占用，请先执行 scripts\stop.bat 停止已有服务
    exit /b 1
)

REM 构建前端 Web (Vue 3)
echo [1/3] 正在全自动编译打包 Vue 3 现代前端...
cd /d web
call npm run build
if !errorlevel! neq 0 (
    echo [错误] 前端打包编译失败！请检查 web 目录。
    cd /d ..
    exit /b 1
)
cd /d ..

REM 构建服务
echo [2/3] 构建后端 Go 程序 (自动嵌入最新前端网页)...
go build -o bin\trpc_agent_server.exe ./cmd/server
if !errorlevel! neq 0 (
    echo [错误] 构建失败！
    exit /b 1
)

REM 后台启动服务（日志重定向到文件）
echo [3/3] 启动 Web 服务...
start "trpc_agent_server" /B bin\trpc_agent_server.exe > bin\log\server.log 2>&1

REM 等待服务就绪（用 ping 替代 timeout，避免输入重定向报错）
echo 等待服务启动...
ping -n 3 127.0.0.1 >nul 2>&1

REM 验证端口监听
netstat -ano | findstr ":%PORT% " | findstr "LISTENING" >nul 2>&1
if !errorlevel!==0 (
    echo.
    echo [成功] Web 服务已启动！
    echo   访问地址: http://localhost:%PORT%
    echo   日志目录: bin\log\
    echo   停止服务: scripts\stop.bat
) else (
    echo [警告] 服务可能未正常启动，请查看日志: bin\log\server.log
)
echo.
endlocal
