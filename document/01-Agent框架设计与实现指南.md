# trpc_agent: 从零打造一个 AI Agent 的技术指南

> 本指南旨在以最清晰、最精炼的方式，向开发者展示如何基于腾讯开源框架 `trpc-agent-go`，从零开始构建一个具备**工业级稳定性、自主推理能力、安全沙箱隔离**的通用 AI Agent 平台。

---

## 🎯 一、什么是 AI Agent？它与普通 ChatBot 的本质区别

| 维度 | ChatBot (普通大模型对话) | AI Agent (智能体) |
|------|-------------------------|-------------------|
| **核心能力** | 仅能基于上下文生成文本 | 能够**感知环境、规划目标、调用外部工具并执行物理动作** |
| **交互模式** | 一问一答 | **推理-行动-观测 (ReAct) 自驱循环** |
| **工具调用** | ❌ 不支持 | ✅ 支持函数调用 (Function Calling)、文件读写、终端执行、网络检索等 |
| **安全边界** | 无 | ✅ 沙箱隔离、权限控制、人工审批门禁 (HITL) |
| **记忆系统** | 短期上下文 | ✅ 短期滑动压缩 + 长期向量检索 (RAG) |

---

## 🏗️ 二、trpc_agent 整体架构

本项目采用**前后端一体化 + 协程多路复用**的高性能架构，整体分为 **UI 展示层**、**Agent 推理层**、**LLM 模型层** 三大职责域：

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          🖥️ UI 展示层 (Vue 3 前端控制台)                         │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  职责：会话管理 · 消息渲染 · 流式打字效果 · ThinkingChain 可视化 · 状态同步    │  │
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────────┐  │  │
│  │  │   会话侧栏     │  │   聊天主区     │  │      Context 面板             │  │  │
│  │  │  (Sessions)   │  │   (Chat)      │  │    (ThinkingChain)            │  │  │
│  │  └───────┬───────┘  └───────┬───────┘  └──────────┬───────────────────┘  │  │
│  └──────────┼──────────────────┼──────────────────────┼──────────────────────┘  │
└─────────────┼──────────────────┼──────────────────────┼─────────────────────────┘
              │                  │                      │
              └──────────────────┼──────────────────────┘
                                 │ SSE (Server-Sent Events)
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          🤖 Agent 推理层 (Go + trpc-agent-go)                    │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  职责：请求编排 · Prompt 组装 · 工具调度 · ReAct 循环 · 上下文压缩 · 安全拦截    │  │
│  │                                                                          │  │
│  │  ┌────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    Go HTTP-SSE 网关                                │  │  │
│  │  │   事件解包/多路复用 · ID 追溯引擎 · TeeReader 用量嗅探 · 双轨持久化   │  │  │
│  │  └──────────────────────────────────┬─────────────────────────────────┘  │  │
│  │                                     ▼                                    │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────────────┐  │  │
│  │  │   LLMAgent   │  │  SessionSvc  │  │        Tools Registry          │  │  │
│  │  │   (ReAct)    │  │  (50% 滑窗)  │  │        (10+ 工具)             │  │  │
│  │  └───────┬──────┘  └──────────────┘  └────────────────────────────────┘  │  │
│  └──────────┼───────────────────────────────────────────────────────────────┘  │
└─────────────┼───────────────────────────────────────────────────────────────────┘
              │ Function Calling / Chat Completion
              ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          🧠 LLM 模型层 (大语言模型)                              │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  职责：自然语言理解 · 推理生成 · Tool Call 解析 · Token 计费 · 模型能力边界     │  │
│  │                                                                          │  │
│  │  ┌────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                                                                    │  │  │
│  │  │              ┌─────────────────────────┐                          │  │  │
│  │  │              │      大语言模型 (LLM)    │                          │  │  │
│  │  │              │                         │                          │  │  │
│  │  │              │   ┌─────────────────┐   │                          │  │  │
│  │  │              │   │   GPT-4o /      │   │                          │  │  │
│  │  │              │   │   Claude /      │   │                          │  │  │
│  │  │              │   │   DeepSeek /    │   │                          │  │  │
│  │  │              │   │   通义千问 等    │   │                          │  │  │
│  │  │              │   └─────────────────┘   │                          │  │  │
│  │  │              │                         │                          │  │  │
│  │  │              │   💡 "大脑" - 提供推理能力与工具调用决策               │  │  │
│  │  │              └─────────────────────────┘                          │  │  │
│  │  │                                                                    │  │  │
│  │  └────────────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 📊 架构分层职责一览

| 层级 | 技术栈 | 核心职责 | 不负责 |
|------|--------|----------|--------|
| **🖥️ UI 展示层** | Vue 3 + Pinia | 会话管理、消息渲染、流式打字效果、ThinkingChain 可视化 | 推理逻辑、工具调用、模型交互 |
| **🤖 Agent 推理层** | Go + trpc-agent-go | Prompt 组装、ReAct 循环、工具调度、上下文压缩、安全拦截 | 具体模型能力、Token 计费 |
| **🧠 LLM 模型层** | OpenAI / Anthropic / 阿里 / DeepSeek | 自然语言理解、推理生成、Tool Call 解析、Token 计费 | 业务逻辑、工具实现 |

---

## 🧠 三、四大核心技术模块深度解析

### 1. Prompt 组装机制 (Prompt Assembly)

大模型的输出质量，100% 取决于输入 Prompt 的精准度。本框架设计了**三层动态装配链**：

```go
// 1. 静态系统指令 (System Instruction) - 定义角色与工具规范
const defaultInstruction = `
你是一个专业的 AI 助手，具备以下工具能力：
- calculator: 数学四则运算
- read_file / write_file / edit_file: 文件读写与局部修改
- web_search / web_scrape: 全网检索与正文提取
- run_command: 受限终端命令执行

【硬约束】
- 涉及现实世界交互，必须调用对应工具，禁止凭空编造
- 参数格式必须严格符合 JSON Schema
`

// 2. 动态环境上下文 (Dynamic Context) - 注入实时时空信息
context := map[string]any{
    "current_time":   time.Now().Format("2006-01-02 15:04:05 Monday"),
    "workspace_root": "/path/to/project",
}

// 3. 工具 Schema 自动生成 (JSON Schema Reflection)
// 框架通过 Go 反射机制，自动解析 Input 结构体的 json tag 与 description，
// 在运行时生成符合 OpenAI 规范的 JSON Schema，无需手动维护！
type ReadFileInput struct {
    Path string `json:"path" description:"目标文件的相对路径"`
}
```

---

### 2. Context 管理与 Token 压缩 (Context Management)

**痛点**：长对话中，历史消息会无限累加，导致：
- 上下文超出模型窗口限制 (如 32K tokens)
- API 调用成本暴增 (一次提问消耗 10 万 tokens)
- 响应延迟飙升

**解决方案：50% 满水位自动摘要 + 滑动窗口压缩**

```go
// 深度复用 trpc-agent-go 原生能力
summarizer := summary.NewSummarizer(model,
    summary.WithContextThreshold(
        summary.WithContextThresholdRatio(0.5), // 50% 黄金警戒线
    ),
    summary.WithMaxSummaryWords(150),           // 精炼摘要不超过 150 字
)

sessionSvc := inmemory.NewSessionService(
    inmemory.WithSummarizer(summarizer),
    inmemory.WithSessionEventLimit(100),         // 限制最大事件数
)

// 效果：会话 Tokens 永远稳定在 2000-3000 的清水位，自动裁剪陈旧历史！
```

---

### 3. Tools 调用流程 (Tool Calling Pipeline)

工具调用是 Agent 的"手和脚"，本框架实现了**全生命周期的安全加固**：

```
用户提问
    │
    ▼
┌─────────────────────────────────────┐
│  1. LLM 生成 Tool Call 指令         │
│     {"name": "read_file",           │
│      "arguments": {"path": "src/main.go"}}
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  2. 强类型参数反序列化               │
│     json.Unmarshal → ReadFileInput  │
│     触发字段格式校验                 │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  3. 安全沙箱边界校验                 │
│     ensureSafePath(path)            │
│     - 路径标准化 (filepath.Clean)   │
│     - 前缀锁定 (禁止 ../../ 逃逸)   │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  4. 人工审批门禁 (HITL)              │
│     - 写操作 (write_file) → 弹窗确认 │
│     - 命令执行 (run_command) → 弹窗  │
│     - 读操作 → 直接放行              │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  5. 执行工具函数                     │
│     os.Open / os.WriteFile / exec.Command
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  6. 结果封装为 ToolMessage          │
│     触发下一轮 ReAct 思考           │
└─────────────────────────────────────┘
```

---

### 4. ReAct 自驱推理循环 (ReAct Design Pattern)

ReAct (Reasoning + Acting) 是 Agent 的"大脑"，实现**自主拆解复杂任务、在挫折中自我调试**：

```
                    ┌─────────────────────────┐
                    │     用户目标 (Goal)     │
                    └────────────┬────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                     🔍 思考 (Thought)
                     "我需要先读取配置文件..."
                    └────────────┬────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                     🛠️ 行动 (Action)
                     调用 read_file("config.yaml")
                    └────────────┬────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                     👁️ 观测 (Observation)
                     "port: 8080, mode: debug"
                    └────────────┬────────────┘
                                 │
                                 ▼
                     是否达成目标？
                     ├── 否 (NO) → 回到 🔍 思考，调整策略
                     └── 是 (YES) → 输出 Final Answer
```

**自愈容错机制**：
- **参数错误**：工具返回错误堆栈 → 作为 Observation 塞回上下文 → LLM 自动修正参数并重试
- **熔断保护**：`MaxToolIterations = 5` + `MaxLLMCalls = 10`，防止死循环

---

## 🛡️ 四、安全防线体系

| 防线 | 机制 | 实现 |
|------|------|------|
| **沙箱隔离** | 路径前缀锁定 | `ensureSafePath()` 强制校验，禁止 `../../` 逃逸 |
| **命令熔断** | 超时强杀 | `exec.CommandContext(runCtx)`，30s 自动终止 |
| **高危拦截** | 黑名单过滤 | 拦截 `rmdir /s`、`del`、`format` 等破坏性指令 |
| **人工审批** | HITL Gate | 写操作必须用户点击"允许"，前端行内卡片确认 |
| **用量清洗** | 协议探针 | HTTP 层拦截真实用量，清洗底层库的累加污染 |

---

## 📊 五、可观测性与调试

### 1. 协议级流量探针 (`bin/llm_io.log`)

在 `cmd/server/main.go` 中劫持全局 `http.DefaultTransport`，部署 `LoggingRoundTripper`：

```go
// 上行 Prompt 完整落盘 (含 System、历史消息、工具 Schema)
[LLM REQUEST PROMPT PAYLOAD (UP)]
{
  "model": "deepseek-ai/DeepSeek-V3",
  "messages": [...],
  "tools": [...]
}

// 下行 SSE 流式逐帧记录
[LLM RESPONSE STREAM (DOWN)]
data: {"choices":[{"delta":{"content":"一"}}]}
data: {"choices":[{"delta":{"content":"箱"}}]}
```

### 2. 自动化集成测试 (`cmd/test_agent/main.go`)

一键验证 ReAct 多轮推理、SSE 事件格式、以及 IO 日志正确性。

---

## 🚀 六、前端流式渲染优化

### 1. 液态打字缓冲机 (Liquid Typing Buffer)

```typescript
// 核心思想：后端吐字 → 队列缓冲 → 匀速流出
const typingQueue: string[] = []
let typingTimer = setInterval(() => {
    const char = typingQueue.shift()
    renderedText += char
    // 积压加速保护：积压 > 80 字 → 5 倍速狂飙
}, settings.typingSpeed) // 可配置：15ms / 25ms / 30ms
```

### 2. 呼吸能量胶囊光标

```css
.typing-cursor {
    background: linear-gradient(135deg, var(--primary-color), #b794f6);
    animation: cursor-breath 0.75s ease-in-out infinite alternate;
    box-shadow: 0 0 6px var(--primary-color);
}
```

---

## 📂 七、项目核心文件索引

| 文件路径 | 职责 |
|----------|------|
| `cmd/server/main.go` | 启动入口、多模型配置、SessionSummarizer 初始化 |
| `internal/agent/agent.go` | Agent 组装、系统提示词、工具注册 |
| `internal/tools/file/file.go` | 文件系统工具 (list/read/write/edit/glob/grep) |
| `internal/tools/web/web.go` | 网络检索工具 (web_search/web_scrape/http_request) |
| `internal/tools/command/command.go` | 终端命令执行工具 (run_command) |
| `internal/server/server.go` | HTTP-SSE 网关、事件多路复用、安全审批 |
| `web/src/stores/chat.ts` | Pinia 状态管理、流式缓冲队列 |
| `web/src/components/agent/ThinkingChain.vue` | 推理步骤 Timeline 卡片 |

---

## 🎓 八、快速上手 Checklist

- [ ] 克隆项目并配置 `.env` (API Key)
- [ ] 运行 `scripts\start.bat` 一键启动
- [ ] 访问 `http://localhost:8080` 验证三栏控制台
- [ ] 发送消息，观察 ThinkingChain 推理卡片展开
- [ ] 触发 `write_file`，验证前端人工审批弹窗
- [ ] 查看 `bin/llm_io.log`，确认协议级探针正常工作

---

*Copyright © 2026. Powered by tRPC-Go & Vue 3.*
