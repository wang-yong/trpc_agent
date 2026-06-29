# 🌌 tRPC-Agent-Platform

> **基于 trpc-agent-go 框架的生产级 AI Agent 平台**  
> 融合 ReAct 思考链、trpc-agent-go 内置工具生态、多租户持久化与可观测性。

---

## 🎨 核心特性一览 (Features)

### 1. 🥇 Trae / Cursor 级三栏式高级视觉控制台
*   **流式 ThinkingChain Timeline 卡片**：支持折叠/展开，流式实时渲染大模型的 Thought 推理步骤、Tool Call 参数详情及绿色 Observation 物理终端形式输出。
*   **右侧 Context & Tasks 面板**：
    *   **待办任务追踪（Todo Tracker）**：发光呼吸灯（Pulsing Bullet）实时显示 Agent 工具派发与思考进度。
    *   **实时上下文水位仪（Context Meter）**：动态进度条显示当前的 Token 用量与 32k 窗口的水位线。
    *   **联网参考资料专区（References）**：大模型执行搜索后，系统会自动提取、解包网页标题和原始链接。**自动识别并贴上来源标签**，鼠标悬停即刻浮现摘要。
*   **极致双主题系统**：完美支持靛蓝亮色 `#4f46e5` 与深邃黑曜暗蓝 `#0f111a`，阴影与卡片对比度严格对齐，圆角全局收拢为高雅圆润的 **`12px`**。

### 2. 🛠️ trpc-agent-go 内置工具生态

本项目使用 **trpc-agent-go 框架内置工具**，无需自研，开箱即用：

#### 📦 Claude Code ToolSet（代码开发工具集）
| 工具名 | 功能说明 |
|--------|---------|
| **Bash** | 执行本地 Shell 命令（go build, npm run, python test.py 等）|
| **Read** | 读取指定文件的全部内容 |
| **Write** | 创建或覆盖文件 |
| **Edit** | 精准局部替换文件中的代码段（old_str → new_str）|
| **Glob** | 按文件名模式（如 *.go, *.py）递归查找文件 |
| **Grep** | 按关键字搜索代码仓库，返回文件名、行号及内容 |
| **WebFetch** | 抓取指定 URL 网页的纯文本正文内容 |
| **WebSearch** | 开放式网页搜索 |
| **NotebookEdit** | 编辑 .ipynb Jupyter 笔记本文件 |
| **TaskStop** | 停止后台任务 |
| **TaskOutput** | 读取后台任务输出 |

#### 🔍 DuckDuckGo 搜索工具
| 工具名 | 功能说明 |
|--------|---------|
| **duckduckgo** | 调用 DuckDuckGo Instant Answer API，获取事实性、百科类信息 |

#### 📋 Todo 任务管理工具
| 工具名 | 功能说明 |
|--------|---------|
| **todo_write** | 发布或更新当前任务计划清单 |
| **todo_declare_blocker** | 声明客观阻塞条件 |

### 3. 🛡️ 坚不可摧的"双端双路 100% 数据自愈"持久化隔离防线
*   **前端 LocalStorage 深度监视器**：Pinia 深度 watch 自动实时同步会话列表、消息历史到浏览器缓存。
*   **后端多租户 Lazy-Load 隔离落盘**：对 Header 携带的 `X-User-Id` 进行高强度防目录逃逸清洗。每个用户在本地拥有独立的物理分片文件 `bin/sessions/sessions_{userID}.json`，在触发请求时按需懒加载并实时持有锁写盘。
*   **双端故障自愈**：哪怕后端因部署重启重置，前端刷新时依然会优先信任并渲染本地历史；发消息时后端根据 ID 自动透明补建档，**任务列表 0 丢失，数据物理级绝不混淆**。

### 4. 📊 极佳的可观测性与测试回归
*   **结构化 I/O 协议拦截调试日志 (`bin/llm_io.log`)**：物理传输层·协议级零损拦截探针，全量 JSON 格式化、美观落盘每次大模型交互真实的上行 Prompt Payload（包含完整的 system 描述、历史消息序列）与下行的 SSE Delta 字节流，杜绝调试幻觉。
*   **全自动集成跑测回归脚本 (`cmd/test_agent/main.go`)**：一键在 Windows 下拉起会话、注入乘加减运算、逐帧拦截检验 thought/tool_call/observation 多事件，并自动读取验证 `bin/llm_io.log` 文件正确性。

### 5. 📂 现代化工作区资源管理器与顶奢级文件预览面板
*   **100% 自由原生双侧拖拽（Resizable Layout）**：彻底粉碎第三方组件库拖拽限制！左侧边栏、右侧面板均配备完全自由的原生拖拉手柄（margin 重合，Hover 触发科技光晕），高度自适应不粘滞，并支持 `localStorage` 刷新/重启自动记忆。
*   **原生态 Windows 10/11 资源管理器大选择弹窗**：物理锚定当前浏览器 HWND（`GetForegroundWindow`），确保弹窗 100% 置顶在浏览器最前端弹出，并呈现最现代大方的宽屏文件夹浏览器样式。
*   **工作区 1.5s 物理指纹热监听哨兵**：每隔 1.5 秒递归计算工作区状态。任何本地、AI 写盘文件的增删改，均会通过 SSE 实时的推送通知前端，实现 100% 无感自动重绘，体验极佳。
*   **自适应内嵌式预览大面板**：点击文件秒级弹出。自动渲染图片（缩放加渐变底色）、PDF（iframe 流式直接翻页阅读）、Markdown（高奢排版、公式高亮）以及普通代码（深曜黑终端 pre 渲染）和二进制（三维立方体元数据卡片）。支持 50% ⇄ 300% 缩放和一键沉浸式全屏，内置 150KB 物理大文件防护防爆自愈。

### 6. ⏳ 液态打字延迟缓冲机与 50% 水位主动摘要滑动压缩
*   **液态打字延迟缓冲机（Liquid Typing Buffer）**：后端传回的分片进入待打印队列，以你可控的 `typing_speed`（在 `safety.yaml` 中配置，免重启热生效）匀速墨水般流淌出来，配合积压加速保护、尾端光标自然消退、呼吸能量胶囊光标和淡入飘入卡片动画，感官体验无懈可击。
*   **50% 满水位黄金警戒线滑动窗口压缩**：深度复用 trpc-agent-go 极具含金量的原生 `summary.NewSummarizer` 与 `inmemory.SessionService`。在会话 Tokens 累加逼近 50% 水位线时，全自动唤起大模型总结 150 字精炼摘要并滑动裁剪陈旧历史，守护 Token 钱袋子。
*   **物理用量中转清洗阀**：自动纠正底层库带来的全局累加污染，确保前端显示、s.tokenRecords 和 `token_stats.json` 落盘账单 100% 为单次干净物理消耗。

---

## 📦 trpc-agent-go 内置工具详细说明

### 📚 Claude Code ToolSet 详解

Claude Code ToolSet 是 trpc-agent-go 框架提供的面向代码工作的工具集，包含以下能力：

#### Bash - 终端命令执行
```go
// 执行本地 Shell 命令
Bash(ctx, "go build ./...", options)
```
- **支持平台**：Windows (cmd.exe), Linux/macOS (bash)
- **超时保护**：内置超时熔断机制
- **安全防护**：高危命令黑名单拦截

#### Read - 文件读取
```go
// 读取指定文件的全部内容
Read(ctx, "path/to/file.go", options)
```
- **大文件保护**：限制最大读取 150KB
- **编码支持**：UTF-8, GBK 等多编码自动检测

#### Write - 文件写入
```go
// 创建或覆盖文件
Write(ctx, "path/to/file.go", content, options)
```
- **原子写入**：使用临时文件 + rename 确保写入原子性
- **目录创建**：自动创建不存在的父目录

#### Edit - 局部代码替换
```go
// 精准替换文件中的代码段
Edit(ctx, "path/to/file.go", oldStr, newStr, options)
```
- **唯一匹配**：确保 old_str 在文件中唯一
- **省 Token**：只传输变更部分，而非整个文件

#### Glob - 文件模式搜索
```go
// 按文件名模式递归查找文件
Glob(ctx, "*.go", options)
```
- **物理过滤**：自动跳过 node_modules, .git, vendor, bin
- **极速响应**：0.5ms 内返回结果

#### Grep - 内容搜索
```go
// 按关键字搜索代码仓库
Grep(ctx, "func main", options)
```
- **行级匹配**：返回文件名、行号及内容
- **50 条上限**：防止结果过多

#### WebFetch - 网页抓取
```go
// 抓取指定 URL 网页的纯文本
WebFetch(ctx, "https://example.com", options)
```
- **HTML 清洗**：自动剥离标签，返回纯文本
- **大小限制**：最大 80KB

#### WebSearch - 网页搜索
```go
// 开放式网页搜索
WebSearch(ctx, "Go 语言最新特性", options)
```
- **多引擎支持**：DuckDuckGo, Google, 百度
- **自动回退**：一个引擎失败自动尝试其他引擎

### 🔍 DuckDuckGo 搜索工具

基于 DuckDuckGo Instant Answer API，提供事实性、百科类信息搜索功能。

```go
// 创建搜索工具
searchTool := duckduckgo.NewTool()
```

**特点**：
- 免费、无需 API Key
- 返回结构化搜索结果
- 支持中英文查询

### 📋 Todo 任务管理工具

为 Agent 提供结构化、可跨轮持久化的任务清单。

```go
// 创建 Todo 工具
todoTool := todo.New()
```

**功能**：
- `todo_write`：发布或更新当前任务计划
- `todo_declare_blocker`：声明客观阻塞条件
- 跨轮持久化：任务清单保存到 session，跨对话保留
- UI 友好：返回结构化结果，便于前端渲染

---

## 📂 项目结构 (Repository Layout)

```
trpc_agent/
├── cmd/
│   ├── server/main.go           # 启动入口（多模型配置与注册）
│   └── test_agent/main.go       # 🟢 自动化集成测试脚本（100% SSE 多事件跑通校验）
├── web/                          # 前端项目（Vue 3 + Vite + Naive UI）
│   ├── src/
│   │   ├── api/                  # API 封装（X-User-Id Header、streamChat SSE 解包）
│   │   ├── stores/               # Pinia 状态管理（localStorage 深度 watch 保存自愈）
│   │   ├── components/           # 细粒度组件
│   │   │   ├── chat/             # 聊天 & 气泡时分戳 (.time-stamp 右对齐美化)
│   │   │   └── agent/            # Agent 组件（ThinkingChain 思考与 Observation 终端渲染）
│   │   └── views/               # 页面视图（ChatView[三栏控制台], StatsView[高精度图表]）
├── internal/
│   ├── agent/
│   │   └── agent.go              # Agent 组装（使用 trpc-agent-go 内置工具）
│   ├── context/                  # 上下文管理（话题检测、智能压缩）
│   ├── embedding/                # 向量嵌入服务
│   └── server/                  # HTTP 服务、SSE 路由多路复用网关
│       ├── server.go            # X-User-Id 过滤、分片 Lazy Load、流 ID 追溯与 I/O 归档
│       └── static/              # 前端编译嵌入层（embed.FS）
├── bin/                         # 可执行二进制及日志区
│   ├── sessions/                # 用户会话元数据分片物理隔离目录（sessions_{userID}.json）
│   ├── server.log               # 运维启动日志
│   └── llm_io.log               # 结构化黑匣子 I/O 调试日志
├── document/                    # 设计文档库
│   ├── README.md                # 文档中心入口与阅读路径
│   ├── 01-Agent框架设计与实现指南.md  # 核心入口：Agent 架构全景图
│   └── 02-核心模块技术深度解析.md    # 源码级技术实现细节
├── scripts/                     # 快捷运维脚本库（start.bat, stop.bat, restart.bat）
├── todo_list.md                 # 📋 项目敏捷开发任务实时看板（Todo List）
├── go.mod
├── Makefile
└── .env                         # 本地环境秘钥配置
```

---

## ⚡ 快速开始 (Quick Start)

### 1. 极速一键部署 (Windows)

本地无需安装 Node.js 与任何前端环境。我们提供了全套编译嵌入脚本，你可以一秒拉起全套服务：

1.  **克隆代码仓库**：
    ```bash
    git clone https://github.com/wang-yong/trpc_agent.git
    cd trpc_agent
    ```
2.  **配置 `.env` 密钥文件**：
    在根目录下创建 `.env` 文件并填入你的硅基流动（或 OpenAI 兼容）密钥与模型：
    ```env
    OPENAI_API_KEY="sk-xxxxxxxx"
    OPENAI_BASE_URL="https://api.siliconflow.cn/v1"
    ```
3.  **一键启动服务**：
    在根目录下双击运行 `scripts\start.bat`。脚本会自动：
    *   读取并加载 `.env` 环境变量；
    *   检查端口 `8080` 占用并安全回收；
    *   执行 `go build` 编译 Go 后端服务；
    *   **后台静默启动服务并自动验证端口监听**！
4.  **访问体验**：
    打开浏览器访问 **[http://localhost:8080](http://localhost:8080)**，即可开启前所未有的 Trae 级完全体 Agent 大片体验！

---

## 🛠️ 快捷运维与自动化测试 (Maintenance & Test)

在项目根目录下，你可以使用以下命令进行极速快捷开发：

```bash
# 一键在后台拉起启动服务
scripts\start.bat

# 一键停止服务并释放端口
scripts\stop.bat

# 一键平滑重启服务
scripts\restart.bat

# 运行全套单元测试
make test

# 运行自动化集成跑测回归脚本（验证 ReAct 多轮流式与 IO 日志）
go run cmd/test_agent/main.go
```

---

## 🧠 路线图与未来规划

### ✅ 已完成
1.  **阶段零：前端重构 (100% ✅)** ➡️ 完全迁移至 Vue 3 + Naive UI 三栏式控制台。
2.  **阶段一：Agent 推理引擎 (100% ✅)** ➡️ 实现 ReAct 思考链、Observation 终端流式展现。
3.  **阶段二：工具生态 (100% ✅)** ➡️ 集成 trpc-agent-go 内置工具（Claude Code ToolSet, DuckDuckGo, Todo）。
4.  **阶段三：自研高级开发工具链 (100% ✅)** ➡️ 工作区资源管理器、文件预览面板、液态打字缓冲机。

### 🔄 进行中
5.  **阶段四：Session/Memory 状态管理 (60% 🔄)** ➡️ 短期记忆、会话持久化、智能压缩。

### 📋 待开发
6.  **阶段五：知识检索与 RAG (0%)** ➡️ 向量数据库、Embedding、文档切片。
7.  **阶段六：GraphAgent 图工作流 (0%)** ➡️ 类型安全的图工作流、多条件路由。
8.  **阶段七：多 Agent 协作 (0%)** ➡️ Chain、Parallel、Cycle 工作流。
9.  **阶段八：Agent Skills 工作流复用 (0%)** ➡️ SKILL.md 格式、Skill 注册中心。
10. **阶段九：Agent 自进化 (0%)** ➡️ 会话复盘、Skill 自动提取。
11. **阶段十：评测与基准 (0%)** ➡️ EvalSet、自动化评测。
12. **阶段十一：协议集成 (0%)** ➡️ A2A、AG-UI。
13. **阶段十二：可观测性 (0%)** ➡️ OpenTelemetry、Langfuse。

---

## 👥 相关链接

*   **trpc-agent-go 官方文档**：[https://trpc-group.github.io/trpc-agent-go/](https://trpc-group.github.io/trpc-agent-go/)
*   **trpc-agent-go GitHub**：[https://github.com/trpc-group/trpc-agent-go](https://github.com/trpc-group/trpc-agent-go)
*   **设计文档库**：更多系统演进、物理工具安全模型与自愈防线请参阅 `document/`。
*   **开发看板**：最新的细粒度开发进度请查看根目录 `todo_list.md`。

*Copyright © 2026. Powered by tRPC-Go & Vue 3.*
