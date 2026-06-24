# 项目开发任务实时看板（Todo List）

> **看板状态说明**：
> 🟢 **已完成 (Completed)**：`[x]`  
> 🟡 **进行中 (In Progress)**：`[/]`  
> ⚪ **待开发 (Backlog)**：`[ ]`

---

## 📊 tRPC-Agent-Go 框架功能对照分析

### 已实现 vs 待实现功能总览

| 功能类别 | tRPC-Agent-Go 特性 | 实现状态 | 完成度 |
|---------|-------------------|---------|--------|
| **Go-Native Agent Runtime** | 流式 Runner、context cancel | ✅ 已实现 | 100% |
| **LLM Agent** | 基础 Agent、工具调用、ReAct 推理 | ✅ 已实现 | 100% |
| **Session/Memory** | 短期记忆、会话持久化 | ✅ 已实现 | 60% |
| **知识检索** | RAG、向量数据库 | ❌ 未实现 | 0% |
| **GraphAgent** | 图工作流、多条件路由 | ❌ 未实现 | 0% |
| **多 Agent 协作** | Chain、Parallel、Cycle 工作流 | ❌ 未实现 | 0% |
| **Agent Skills** | SKILL.md 工作流复用 | ❌ 未实现 | 0% |
| **Agent 自进化** | Hermes-style 会话复盘 | ❌ 未实现 | 0% |
| **Prompt Caching** | 自动优化成本 | ❌ 未实现 | 0% |
| **评测与基准** | EvalSet + Metric | ❌ 未实现 | 0% |
| **协议集成** | A2A、AG-UI、MCP | ❌ 未实现 | 0% |
| **可观测性** | OpenTelemetry tracing、metrics | ❌ 未实现 | 0% |

---

## 🎨 阶段零：前端重构（基础设施）- **✅ 已完成 (100%)**

- [x] 初始化 Vue 3 + Vite + Naive UI + Pinia 现代化前端开发环境
- [x] 迁移聊天主页面：消息列表、流式渲染、会话侧边栏、模型选择
- [x] 迁移统计页面：Token 统计高对比度图表（修复 cell 挤压不对齐漏洞）
- [x] 迁移主题切换（明亮/暗色双主题，实现 App.vue 全局变量响应式反射机制）
- [x] 重构用户弹出菜单（支持点击外侧自适应隐藏、选择菜单项后自动关闭弹窗）
- [x] 实现智能双向导航（主页显示"Token统计" ⇄ 统计页显示"返回对话"；侧栏 Logo 支持一键快捷回主页）
- [x] 接入 Go embed 静态打包机制（scripts/restart.bat 一键热重载生产产物）
- [x] 实现 Vite 开发跨域代理，打通 localhost:5173 联调
- [x] 全面清理并彻底删除旧的简陋 HTML 静态历史文件

---

## ⚡ 阶段一：Agent 推理引擎（可视化）- **✅ 已完成 (100%)**

- [x] 深度打通 ReAct（Thought-Action-Observation）自驱推理循环内核
- [x] 设定 MaxToolIterations = 5 / MaxLLMCalls = 10 的安全熔断保护网
- [x] 升级 SSE 智能路由网关：多路复用 thought, tool_call, observation, delta 核心事件
- [x] 建立流式 ID 追溯引擎（Index -> ID 匹配），解决 arguments 分词丢失 ID 导致前端不重绘的硬伤
- [x] 前端开发高颜值 `ThinkingChain.vue` Timeline 推理卡片（支持折叠、入参 Arguments 渲染及物理 Observation 终端形式输出）
- [x] 建立双端损坏 JSON 自愈模块：剔除多余的 `{{` 和 `}}`，自适应纠错为合法 JSON 串
- [x] 挂载普通文本流 delta 极致剪枝器：拦截剔除 `\n}`、`},`、`],` 等由于前回合残留的非自然语言符号

---

## 📦 阶段二：工具生态（Function Tool）- **✅ 已完成 (100%)**

- [x] **数学计算**：`calculator` 高精度计算器工具，拦截大模型心算，保障计算 100% 正确
- [x] **文件系统**：列出目录树 (`list_directory`)、文本读取 (`read_file`)、写入/创建文件 (`write_file`)
- [x] **网络检索**：网页正文抓取 (`web_scrape`)、通用 HTTP 交互客户端 (`http_request`)
- [x] **网络检索自愈**：`web_search` 搜索引擎自愈。海外 DuckDuckGo 遇 WAF 阻断时，瞬间毫秒级自动降级拉起百度 desktop 实时检索，达率 100%
- [x] **终端执行**：`run_command` 本地受限 Windows cmd.exe 命令行物理驱动工具，自带 30s 硬超时强杀及高危黑名单拦截防线
- [x] **文件修改**：`edit_file` 精细手术刀局部代码段 replacement 替换工具，省 Token 且绝对防止大文件截断损坏
- [x] **文件检索**：`glob_files` 文件全局递归查找，**物理过滤跳过 node_modules, .git, bin, vendor 等超重冗余目录**，0.5 毫秒内出结果
- [x] **王牌检索**：`grep_search` 全局正文行级关键字极速匹配（类似 Ripgrep，带 50 条上限保护防爆，自动忽略二进制大文件）
- [x] **安全防护 (Human-in-the-Loop)**：`notify_approval` 机制。当模型调度写文件/改文件/执行 cmd 等高危动作时，挂起 SSE 并在前端弹出阻断确认框，必须用户点击"允许"方可物理派发

---

## 🌐 阶段 2.5：协议集成（MCP Tool）- **⚪ 待开发**

> 对应 tRPC-Agent-Go：MCP Tool 接入工具生态

- [ ] **MCP Server 集成**：实现 Model Context Protocol 客户端，接入外部 MCP 工具服务
- [ ] **MCP Tool 注册**：支持动态注册和发现 MCP 工具
- [ ] **MCP 资源订阅**：支持订阅 MCP 提供的资源和提示模板
- [ ] **MCP 传输层**：支持 stdio、HTTP SSE、WebSocket 三种传输方式

---

## 🧠 阶段三：Session/Memory 状态管理 - **🟡 进行中 (60%)**

> 对应 tRPC-Agent-Go：Session、Memory、Artifacts 持久化状态

- [x] **短期记忆管理**：深度集成 `summary.NewSummarizer` 与 `inmemory.SessionService`，实现 50% 满水位警戒线自动触发大模型 150 字精炼摘要与滑动窗口物理事件截断裁剪
- [x] **会话持久化**：多用户分片 Lazily Load 持久化隔离，按 X-User-Id 分用户 sessions_{userID}.json 物理文件分片落盘
- [x] **智能边界压缩**：话题切换时立即触发压缩，三层策略（强关联/弱关联/无关）
- [ ] **Memory 长期记忆**：sqlite-vss 向量数据库集成，检索历史对话，建立用户偏好画像
- [ ] **Artifacts 管理**：支持存储和管理 Agent 生成的文件、图片等产物
- [ ] **Memory 工具化**：提供 `save_memory`、`load_memory`、`update_memory` 等工具供 Agent 调用

---

## 🔍 阶段四：知识检索与 RAG - **⚪ 待开发**

> 对应 tRPC-Agent-Go：知识检索、私有知识库 RAG

- [ ] **向量数据库集成**：集成 sqlite-vss 或本地向量存储，支持 Embedding 存储和检索
- [ ] **Embedding 服务**：集成 sentence-transformers 或 OpenAI Embedding API
- [ ] **文档切片**：支持 PDF、Markdown、TXT 文档自动切片
- [ ] **RAG 检索工具**：`search_knowledge` 工具，语义检索知识库
- [ ] **引用溯源**：检索结果带上下文引用，支持在回答中标注来源
- [ ] **知识库管理**：支持上传、删除、更新知识文档

---

## 📊 阶段五：Prompt Caching 与成本优化 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：Prompt Caching，自动优化成本，缓存内容最高可节省 90%

- [ ] **System Prompt 缓存**：检测 System Prompt 变化，支持 OpenAI/Anthropic Prompt Caching
- [ ] **历史消息去重**：识别重复的历史消息，复用 Embedding 缓存
- [ ] **Token 使用统计**：按模型、按用户统计 Token 消耗和成本
- [ ] **成本预警**：设置每日/每月 Token 预算上限，超限自动告警

---

## 🎯 阶段六：GraphAgent 图工作流 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：GraphAgent，类型安全的图工作流，支持多条件路由，功能对标 LangGraph

- [ ] **StateGraph 定义**：实现类型安全的状态图定义 DSL
- [ ] **Node 节点系统**：支持多种节点类型（LLM 节点、工具节点、条件节点）
- [ ] **Edge 边系统**：支持无条件边、条件边、多路分支
- [ ] **Checkpointer**：支持检查点保存和恢复，实现断点续跑
- [ ] **可视化编辑器**：前端 Graph 可视化编辑器（类似 LangGraph Studio）
- [ ] **Graph 序列化**：支持 Graph 定义的 JSON/YAML 序列化和反序列化

---

## 👥 阶段七：多 Agent 协作 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：多 Agent 协作，Chain、Parallel 和 Cycle 工作流

- [ ] **多 Agent 角色定义**：规划者（Planner）+ 执行者（Coder）+ 单元测试审查者（Reviewer）
- [ ] **Chain 工作流**：顺序执行多个 Agent，前一个 Agent 的输出作为后一个的输入
- [ ] **Parallel 工作流**：并行执行多个 Agent，等待所有完成后汇总结果
- [ ] **Cycle 工作流**：循环执行 Agent 直到满足终止条件
- [ ] **Supervisor 模式**：一个 Supervisor Agent 负责调度多个子 Agent
- [ ] **多 Agent 协作面板**：前端渲染不同 Agent 头像、状态与协作 Timeline

---

## 🛠️ 阶段八：Agent Skills 工作流复用 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：Agent Skills，可复用的 SKILL.md 工作流，支持安全执行

- [ ] **SKILL.md 格式定义**：定义 Agent Skill 的 Markdown 格式规范
- [ ] **Skill 注册中心**：支持注册、发现、版本管理 Skill
- [ ] **Skill 执行引擎**：安全执行 Skill 定义的工作流
- [ ] **Skill 市场**：支持导入/导出 Skill，建立团队级 Skill 共享
- [ ] **Skill 评测**：支持对 Skill 进行自动化评测和打分

---

## 🔄 阶段九：Agent 自进化 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：Hermes-style 会话复盘，自动提取、门禁校验并发布可复用 SKILL.md 工作流

- [ ] **会话复盘引擎**：分析历史会话，提取有价值的交互模式
- [ ] **Skill 自动提取**：从成功的会话中自动提取可复用 Skill
- [ ] **门禁校验**：自动校验提取的 Skill 质量（覆盖率、成功率、安全性）
- [ ] **自动发布**：通过门禁的 Skill 自动发布到 Skill 注册中心
- [ ] **反馈循环**：收集 Skill 执行反馈，持续优化 Skill 质量

---

## 📈 阶段十：评测与基准 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：EvalSet + Metric 用于长期质量度量

- [ ] **EvalSet 评测集**：定义标准评测问题集，支持导入/导出
- [ ] **自动化评测**：定期运行评测集，生成评测报告
- [ ] **Metric 指标体系**：定义质量指标（准确率、相关性、安全性等）
- [ ] **评测 Dashboard**：可视化展示评测结果和趋势
- [ ] **A/B 测试**：支持不同模型/配置的对比评测

---

## 📡 阶段十一：协议集成 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：AG-UI 对接前端、A2A 实现 Agent 互通

- [ ] **AG-UI 协议**：实现 Agent-GUI 协议，支持前端动态渲染 Agent 状态
- [ ] **A2A 协议**：实现 Agent-to-Agent 协议，支持不同 Agent 服务间的互通
- [ ] **Agent Card**：定义 Agent 能力卡片，支持 Agent 发现和能力协商
- [ ] **协议适配层**：统一的协议适配层，支持多种协议的无缝切换

---

## 👁️ 阶段十二：可观测性 - **⚪ 待开发**

> 对应 tRPC-Agent-Go：OpenTelemetry tracing、metrics 与 Langfuse

- [ ] **OpenTelemetry Tracing**：集成 OTEL tracing，记录每次 LLM 调用、工具执行的链路
- [ ] **OpenTelemetry Metrics**：集成 OTEL metrics，暴露 Token 使用、延迟、错误率等指标
- [ ] **Langfuse 集成**：接入 Langfuse LLM 可观测平台
- [ ] **Trace 可视化**：前端展示 LLM 调用链路和耗时分析
- [ ] **告警规则**：配置异常检测和告警规则

---

## 🚀 阶段十三：高级开发工具链 - **🟡 进行中 (70%)**

- [x] **工作区大升级**：支持自定义 Agent 运行根目录，支持侧栏/面板 100% 毫无限制的原生极速拖拽拉伸
- [x] **原生文件夹弹窗**：支持在网页一键点击拉起 Windows 操作系统原生的文件夹选择弹窗
- [x] **全自动实时文件树**：设计 1.5 秒后端轻量级指纹轮询，文件有变动时通过 SSE 实时静默强推，前端实现 100% 无感全自动加载刷新
- [ ] **代码协作**：`git_commit_helper` 工具。自动执行 git status / git diff 析出代码变更，并 100% 自动按照 Conventional Commit 规范生成并提交 Git Commit
- [ ] **浏览器自动化**：`integrated_browser` 集成无头 ChromeDP / Playwright 支持自动爬网、登录及动态 Web 交互自测

---

## 💾 全观测性与系统账单记录 - **✅ 已完成 (100%)**

- [x] 建立多用户分片 Lazily Load 持久化隔离：按 X-User-Id 清洗物理路径，分用户 sessions_{userID}.json 物理文件分片落盘，任务列表 0 丢失
- [x] 建立结构化黑匣子 I/O 调试日志 `bin/llm_io.log`，清晰归档每一次对话
- [x] 建立全自动集成跑测脚本 `cmd/test_agent/main.go`，一键回归 100% SSE 多事件、格式、自愈通流
- [x] 建立 Trae 风格三栏控制面板（包含 Todo 待办发光呼吸灯、Token 实时水位条以及自动渲染解包的百度/全网参考资料列表）

---

## 📋 功能实现优先级建议

### P0 - 核心能力（建议立即开始）
1. **MCP Tool 集成** - 扩展工具生态，接入第三方工具
2. **向量数据库 + RAG** - 实现知识检索能力
3. **Memory 长期记忆** - 完善记忆系统

### P1 - 协作能力（建议第二阶段）
4. **多 Agent 协作** - Chain、Parallel 工作流
5. **GraphAgent 图工作流** - 类型安全的复杂工作流
6. **OpenTelemetry 可观测性** - 生产级监控

### P2 - 高级能力（建议第三阶段）
7. **Agent Skills** - 工作流复用
8. **Agent 自进化** - 自动优化
9. **评测与基准** - 质量保障

### P3 - 协议与集成（建议第四阶段）
10. **A2A 协议** - Agent 互通
11. **AG-UI 协议** - 前端动态渲染
12. **Prompt Caching** - 成本优化
