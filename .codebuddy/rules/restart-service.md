# 重启服务命令

## 重启 trpc_agent 服务

当用户要求重启服务时，执行以下命令：

```cmd
cd d:/program/trpc_agent && scripts\restart.bat
```

## 脚本说明

`scripts/restart.bat` 会自动完成以下操作：

1. **停止现有服务** - 终止监听端口的进程
2. **构建前端** - 自动编译打包 Vue 3 现代前端
3. **构建后端** - 编译 Go 程序（自动嵌入最新前端网页）
4. **启动服务** - 后台启动 Web 服务

## 服务信息

- **访问地址**: http://localhost:8080
- **日志文件**: bin\server.log
- **停止服务**: `scripts\stop.bat`
- **启动服务**: `scripts\start.bat`

## 其他命令

- **仅停止服务**: `cd d:/program/trpc_agent && scripts\stop.bat`
- **仅启动服务**: `cd d:/program/trpc_agent && scripts\start.bat`
