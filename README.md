# tRPC-Agent-Go 测试用例

基于 [tRPC-Agent-Go](https://github.com/trpc-group/trpc-agent-go) 框架的测试用例。

## 项目结构

```
.
├── cmd/
│   ├── agent/
│   │   └── main.go                    # 命令行主程序入口
│   └── server/
│       └── main.go                    # Web 服务入口
├── internal/
│   ├── calculator/
│   │   ├── calculator.go              # 计算器领域逻辑
│   │   └── calculator_test.go         # 计算器单元测试
│   ├── agent/
│   │   ├── agent.go                   # Agent 组装逻辑
│   │   └── agent_integration_test.go  # Agent 集成测试
│   └── server/
│       ├── server.go                  # HTTP + SSE 流式聊天服务
│       └── static/
│           └── index.html             # Web 聊天界面
├── vendor/                            # 第三方依赖缓存
├── go.mod                             # Go module 文件
├── go.sum                             # 依赖校验和
├── Makefile                           # 构建和测试脚本
├── .gitignore                         # Git 忽略规则
└── README.md                          # 说明文档
```

## 前置条件

- Go 1.21 或更高版本
- LLM 提供商 API 密钥（OpenAI、DeepSeek 等）

## 快速开始

### 1. 配置环境变量

```bash
# Linux/macOS
export OPENAI_API_KEY="your-api-key-here"
export OPENAI_BASE_URL="your-base-url-here"  # 可选

# Windows PowerShell
$env:OPENAI_API_KEY="your-api-key-here"
$env:OPENAI_BASE_URL="your-base-url-here"

# Windows CMD
set OPENAI_API_KEY=your-api-key-here
set OPENAI_BASE_URL=your-base-url-here
```

### 2. 运行测试

```bash
# 运行所有单元测试（跳过集成测试）
make test

# 或直接使用 go test
go test -v -short ./...
```

### 3. 运行集成测试

```bash
# 运行集成测试（需要 API Key）
make test-integration

# 或直接使用 go test
go test -v ./... -run TestRunIntegrationTests
```

### 4. 运行主程序

```bash
# 运行主程序
make run

# 或直接使用 go run
go run ./cmd/agent
```

### 5. 启动 Web 聊天界面

```bash
# 启动 Web 服务（默认监听 :8080）
make serve

# 或直接使用 go run
go run ./cmd/server
```

启动后浏览器访问 [http://localhost:8080](http://localhost:8080)，即可在网页上与 AI 对话。

可通过环境变量自定义监听地址：

```bash
# Windows CMD
set SERVER_ADDR=:9000

# Linux/macOS
export SERVER_ADDR=:9000
```

Web 服务说明：
- `GET /`：返回聊天界面（静态页面已通过 `embed` 打包进二进制）
- `POST /api/chat`：聊天接口，请求体 `{"message": "...", "session_id": "..."}`，以 **SSE（text/event-stream）** 流式返回 AI 回复
- 同一 `session_id` 的多次请求会保留对话上下文，点击「新对话」会生成新会话

## 测试说明

### 单元测试 (calculator_test.go)

测试计算器工具的各个功能：

- `TestCalculator_Add` - 测试加法运算
- `TestCalculator_Subtract` - 测试减法运算
- `TestCalculator_Multiply` - 测试乘法运算
- `TestCalculator_Divide` - 测试除法运算
- `TestCalculator_DivideByZero` - 测试除以零的错误处理
- `TestCalculator_InvalidOperation` - 测试无效运算类型
- `TestCalculator_AllOperations` - 测试所有运算的综合场景

### 集成测试 (agent_integration_test.go)

测试 Agent 与 LLM 的集成：

- `TestAgent_CalculatorIntegration` - 测试 Agent 调用计算器工具
- `TestAgent_StreamOutput` - 测试流式输出
- `TestAgent_MultipleToolCalls` - 测试多次工具调用
- `TestAgent_ToolDefinition` - 测试工具定义

## 常用命令

```bash
# 显示帮助
make help

# 运行单元测试
make test

# 运行集成测试
make test-integration

# 运行详细测试
make test-verbose

# 生成覆盖率报告
make coverage

# 构建程序
make build

# 清理构建文件
make clean

# 格式化代码
make fmt

# 检查代码格式
make check

# 下载依赖
make deps

# 整理依赖
make tidy

# 验证依赖
make verify
```

## 测试覆盖率

```bash
# 生成 HTML 覆盖率报告
make coverage

# 打开报告
open coverage.html  # macOS
xdg-open coverage.html  # Linux
start coverage.html  # Windows
```

## 项目特点

1. **完整的测试覆盖**：单元测试和集成测试全覆盖
2. **清晰的代码结构**：代码组织清晰，易于维护
3. **灵活的测试配置**：支持跳过集成测试，方便 CI/CD
4. **现代化的工具链**：使用 Go 1.21+ 和 Makefile
5. **详细的文档**：完善的注释和文档

## 注意事项

- 运行集成测试需要有效的 API Key
- 集成测试会消耗 API 额度
- 建议在 CI/CD 中跳过集成测试（使用 `-short` 标志）
- 测试超时设置为 60 秒，可根据需要调整

## 相关链接

- [tRPC-Agent-Go 官方文档](https://trpc-group.github.io/trpc-agent-go/)
- [GitHub 仓库](https://github.com/trpc-group/trpc-agent-go)
- [GoDoc 文档](https://pkg.go.dev/trpc.group/trpc-go/trpc-agent-go)
