# 项目开发任务实时看板（Todo List）

> **看板状态说明**：
> 🟢 **已完成 (Completed)**：`[x]`  
> 🟡 **进行中 (In Progress)**：`[/]`  
> ⚪ **待开发 (Backlog)**：`[ ]`

---

## 🎨 阶段零：前端重构（基础设施）- **✅ 已完成 (100%)**
- [x] 初始化 Vue 3 + Vite + Naive UI + Pinia 现代化前端开发环境
- [x] 迁移聊天主页面：消息列表、流式渲染、会话侧边栏、模型选择
- [x] 迁移统计页面：Token 统计高对比度图表（修复 cell 挤压不对齐漏洞）
- [x] 迁移主题切换（明亮/暗色双主题，实现 App.vue 全局变量响应式反射机制）
- [x] 重构用户弹出菜单（支持点击外侧自适应隐藏、选择菜单项后自动关闭弹窗）
- [x] 实现智能双向导航（主页显示“Token统计” ⇄ 统计页显示“返回对话”；侧栏 Logo 支持一键快捷回主页）
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

## 📦 阶段二：工具生态（能力扩展基础）- **✅ 已完成 (100%)**
- [x] **数学计算**：`calculator` 高精度计算器工具，拦截大模型心算，保障计算 100% 正确
- [x] **文件系统**：列出目录树 (`list_directory`)、文本读取 (`read_file`)、写入/创建文件 (`write_file`)
- [x] **网络检索**：网页正文抓取 (`web_scrape`)、通用 HTTP 交互客户端 (`http_request`)
- [x] **网络检索自愈**：`web_search` 搜索引擎自愈。海外 DuckDuckGo 遇 WAF 阻断时，瞬间毫秒级自动降级拉起百度 desktop 实时检索，达率 100%
- [x] **终端执行**：`run_command` 本地受限 Windows cmd.exe 命令行物理驱动工具，自带 30s 硬超时强杀及高危黑名单拦截防线
- [x] **文件修改**：`edit_file` 精细手术刀局部代码段 replacement 替换工具，省 Token 且绝对防止大文件截断损坏
- [x] **文件检索**：`glob_files` 文件全局递归查找，**物理过滤跳过 node_modules, .git, bin, vendor 等超重冗余目录**，0.5 毫秒内出结果

---

## 🚀 阶段三：自研高级开发工具链与安全网 - **🟡 进行中 (已完成 50%)**
- [x] **王牌检索**：`grep_search` 全局正文行级关键字极速匹配（类似 Ripgrep，带 50 条上限保护防爆，自动忽略二进制大文件）
- [x] **安全防护 (Human-in-the-Loop)**：`notify_approval` 机制。当模型调度写文件/改文件/执行 cmd 等高危动作时，挂起 SSE 并在前端弹出阻断确认框，必须用户点击“允许”方可物理派发
- [ ] **代码协作**：`git_commit_helper` 工具。自动执行 git status / git diff 析出代码变更，并 100% 自动按照 Conventional Commit 规范生成并提交 Git Commit
- [ ] **浏览器自动化**：`integrated_browser` 终端执行。集成无头 ChromeDP / Playwright 支持自动爬网、登录及动态 Web 交互自测

---

## 🧠 阶段四：长期记忆、本地 RAG 与知识库构建 - **⚪ 待开发**
- [ ] 短期记忆管理：上下文窗口滑动压缩 + 自动大模型摘要提取
- [ ] 长期记忆管理：sqlite-vss 向量数据库集成，检索历史对话，建立用户偏好画像
- [ ] 私有知识库 RAG：支持上传 PDF/Markdown 知识文件，全自动切片、Embedding 向量化并实现本地引用

---

## 👥 阶段五：多 Agent 自主协作生态（终极团队）- **⚪ 待开发**
- [ ] 多 Agent 角色定义：规划者（Planner）+ 执行者（Coder）+ 单元测试审查者（Reviewer）
- [ ] 多 Agent 自主协作：任务自主拆解分发、认领与内部交接机制
- [ ] 多 Agent 协作面板：前端渲染不同 Agent 头像、状态与协作 Timeline

---

## 💾 全观测性与系统账单记录
- [x] 建立多用户分片 Lazily Load 持久化隔离：按 X-User-Id 清洗物理路径，分用户 sessions_{userID}.json 物理文件分片落盘，任务列表 0 丢失
- [x] 建立结构化黑匣子 I/O 调试日志 `bin/llm_io.log`，清晰归档每一次对话
- [x] 建立全自动集成跑测脚本 `cmd/test_agent/main.go`，一键回归 100% SSE 多事件、格式、自愈通流
- [x] 建立 Trae 风格三栏控制面板（包含 Todo 待办发光呼吸灯、Token 实时水位条以及自动渲染解包的百度/全网参考资料列表）
