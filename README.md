# 🌌 tRPC-Agent-Platform

> **完全自研、Trae / Cursor 风格三栏式完全体 AI Agent 视觉开发平台**  
> 基于腾讯 [trpc-agent-go] 框架深度打通，融合 ReAct 思考链、10 大王牌本地/云端物理工具箱、多租户 Lazy-Load 物理文件分片持久化与双端双轨自愈保存安全网。

---

## 🎨 核心特性一览 (Features)

### 1. 🥇 Trae / Cursor 级三栏式高级视觉控制台
*   **流式 ThinkingChain Timeline 卡片**：支持折叠/展开，流式实时渲染大模型的 Thought 推理步骤、Tool Call 参数详情及绿色 Observation 物理终端形式输出。
*   **右侧 Context & Tasks 面板**：
    *   **待办任务追踪（Todo Tracker）**：发光呼吸灯（Pulsing Bullet）实时显示 Agent 工具派发与思考进度。
    *   **实时上下文水位仪（Context Meter）**：动态进度条显示当前的 Token 用量与 32k 窗口的水位线。
    *   **联网参考资料专区（References）**：大模型执行 `web_search` 后，系统会自动提取、解包网页标题和原始链接。**自动识别并贴上 [百度] 红色和 [全网] 蓝色来源标签**，鼠标悬停即刻浮现摘要。
*   **极致双主题系统**：完美支持靛蓝亮色 `#4f46e5` 与深邃黑曜暗蓝 `#0f111a`，阴影与卡片对比度严格对齐，圆角全局收拢为高雅圆润的 **`12px`**。

### 2. 🔌 10 大王牌本地/云端物理工具箱
*   **数学四则计算 (`calculator`)**：拦截大模型心算权，保障计算 100% 绝对精确。
*   **目录遍历工具 (`list_directory`)**：安全浏览指定相对路径下的工作区目录。
*   **文本读取工具 (`read_file`)**：读取并返回指定文本文件的全部内容。
*   **文本写入工具 (`write_file`)**：整包写入或覆盖目标文本文件。
*   **局部精密修改 (`edit_file`)**：支持大文件局部 replacement 替换修改，省 Token 且绝对防止大文件截断损坏。
*   **闪电递归检索 (`glob_files`)**：**物理过滤跳过 node_modules, .git, vendor, bin** 等超重冗余目录，0.5 毫秒内出全项目文件名匹配结果。
*   **百度/DDG自愈检索 (`web_search`)**：海外 DuckDuckGo 遇 WAF 阻断时，瞬间毫秒级自动降级拉起百度 desktop 实时检索，成功率 100%。
*   **网页文本提取 (`web_scrape`)**：抓取 HTML 纯文本并提取剥离干净的正文，最大限流 80KB 防止撑爆上下文。
*   **通用 HTTP 客户端 (`http_request`)**：向任意 API 发送通用 POST/GET 调试请求。
*   **本地终端执行器 (`run_command`)**：Windows 受限 cmd 命令行执行器（如 go test、npm build），**自带 30s 自动强杀硬超时**及高危黑名单拦截防线。

### 3. 🛡️ 坚不可摧的“双端双路 100% 数据自愈”持久化隔离防线
*   **前端 LocalStorage 深度监视器**：Pinia 深度 watch 自动实时同步会话列表、消息历史到浏览器缓存。
*   **后端多租户 Lazy-Load 隔离落盘**：对 Header 携带的 `X-User-Id` 进行高强度防目录逃逸清洗。每个用户在本地拥有独立的物理分片文件 `bin/sessions/sessions_{userID}.json`，在触发请求时按需懒加载并实时持有锁写盘。
*   **双端故障自愈**：哪怕后端因部署重启重置，前端刷新时依然会优先信任并渲染本地历史；发消息时后端根据 ID 自动透明补建档，**任务列表 0 丢失，数据物理级绝不混淆**。

### 4. 📊 极佳的可观测性与测试回归
*   **结构化 I/O 黑匣子调试日志 (`bin/llm_io.log`)**：规整地记录下每一次提问的 TIMESTAMP、SESSIONID、MODEL、TOKEN_USAGE、THINKING_CHAIN、每一轮 TOOL_CALLS 入参与对应的 OBSERVATIONS、以及最终答案。
*   **全自动集成跑测回归脚本 (`cmd/test_agent/main.go`)**：一键在 Windows 下拉起会话、注入乘加减运算、逐帧拦截检验 thought/tool_call/observation 多事件，并自动读取验证 `bin/llm_io.log` 文件正确性。

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
│   ├── agent/                   # Agent 组装与系统提示词规范
│   ├── tools/                   # 物理工具实现
│   │   ├── calculator/          # 🟢 经典高精度计算器工具
│   │   ├── file/
│   │   │   └── file.go          # 🟢 安全沙箱文件工具（list_directory, read_file, write_file, edit_file, glob_files, grep_search）
│   │   ├── web/
│   │   │   └── web.go           # 🟢 互联网检索（web_search[百度自愈], web_scrape, http_request）
│   │   └── command/
│   │       └── command.go       # 🟢 终端命令执行工具（run_command 30s超时熔断）
│   ├── server/                  # HTTP 服务、SSE 路由多路复用网关
│   │   ├── server.go            # 🟢 X-User-Id 过滤、分片 Lazy Load、流 ID 追溯与 I/O 归档
│   │   └── static/              # 前端编译嵌入层（embed.FS）
├── bin/                         # 可执行二进制及日志区
│   ├── sessions/                # 🟢 用户会话元数据分片物理隔离目录（sessions_{userID}.json）
│   ├── server.log               # 运维启动日志
│   └── llm_io.log               # 🟢 结构化黑匣子 I/O 调试日志
├── document/                    # 完整设计文档库
│   ├── 01-整体目标规划.md        # 📋 系统架构演进路线图
│   ├── 02-通用Agent设计方案与规划.md
│   └── 03-已实现Agent功能清单与架构.md
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

## 🧠 路线图与未来自研规划

目前我们的系统路线规划如下：
1.  **阶段零：前端重构 (100% ✅)** ➡️ 完全迁移至 Vue 3 + Naive UI 三栏式控制台。
2.  **阶段一：Agent 推理引擎 (100% ✅)** ➡️ 实现 ReAct 思考链、Observation 终端流式展现。
3.  **阶段二：物理工具生态 (100% ✅)** ➡️ calculator, file, web, command 10 大王牌工具完全并入。
4.  **阶段三：自研高级开发工具链与安全网 (进行中 🔄)** ➡️ 主攻 `grep_search` (已交付)、`notify_approval` 弹窗审批、`git_commit_helper` 以及 `integrated_browser` 自动化。
5.  **阶段四：长期记忆与 RAG 知识库 (规划中)** ➡️ 主攻 sqlite-vss 向量检索与 PDF/Markdown RAG 挂载。

---

## 👥 相关链接

*   **设计文档库**：更多系统演进、物理工具安全模型与自愈防线请参阅 `document/`。
*   **开发看板**：最新的细粒度开发进度请查看根目录 `todo_list.md`。

*Copyright © 2026. Powered by tRPC-Go & Vue 3.*
