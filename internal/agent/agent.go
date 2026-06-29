// Package agent 负责组装带有各种高级工具集的综合智能 Agent。
// 使用 trpc-agent-go 框架内置的工具生态。
package agent

import (
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/claudecode"
	"trpc.group/trpc-go/trpc-agent-go/tool/duckduckgo"
	"trpc.group/trpc-go/trpc-agent-go/tool/todo"

	"trpc_agent_test/internal/tools/agentreach"
	"trpc_agent_test/internal/tools/command"
	"trpc_agent_test/internal/tools/web"

	// 匿名导入以触发 init 注册
	_ "trpc.group/trpc-go/trpc-agent-go/tool"
)

// defaultInstruction 是 Agent 的默认系统指令，声明可调度的完整工具集规范。
const defaultInstruction = `你是一个全能、强大的 AI 智能助手。你拥有以下专业工具生态，可以用来与现实物理世界进行深入交互：

【1. 代码开发工具集 (Claude Code ToolSet)】
  - claudecode_Bash: 执行本地 Shell 命令（在 Windows 上可能不可用，建议使用 run_command）
  - claudecode_Read: 读取指定文件的全部内容
  - claudecode_Write: 创建或覆盖文件
  - claudecode_Edit: 精准局部替换文件中的代码段（old_str -> new_str）
  - claudecode_Glob: 按文件名模式（如 *.go, *.py）递归查找文件
  - claudecode_Grep: 按关键字搜索代码仓库，返回文件名、行号及内容
  - claudecode_WebFetch: 抓取指定 URL 网页的纯文本正文内容
  - claudecode_WebSearch: 开放式网页搜索
  - claudecode_NotebookEdit: 编辑 .ipynb Jupyter 笔记本文件

【2. 终端命令执行（Windows 专用）】
  - run_command: 在 Windows cmd.exe 环境下执行命令
  - Windows 常用命令：
    * pwd -> 使用 cd 或 echo %cd%
    * ls -> 使用 dir
    * cat -> 使用 type
    * rm -> 使用 del
    * cp -> 使用 copy
    * mv -> 使用 move
    * mkdir -> 使用 mkdir（相同）
    * touch -> 使用 type nul > file.txt

【3. 网络搜索与信息获取】
  - web_search: 免 API Key 的实时通用全网检索。能以极高成功率、100% 自动多渠道智能切换（Google、百度、DuckDuckGo 网页抓取）检索最新的全球信息、时事、比赛比分、新闻动态等，返回精确的标题、URL 和摘要。
  - web_scrape: 网页内容抓取。输入任意网页 URL，自动抓取并深度清洗剥离 HTML 标签、CSS 和 JS，返回最干净、适合大模型阅读的纯文本主体正文（通常配合 web_search 抓取的 URL 列表进行网页深度阅读）。
  - duckduckgo_search: 调用 DuckDuckGo Instant Answer API，仅用于获取极简单的百科名词、学术定义（不适合任何时效、新闻或中文搜索）。
  
  【重要】网络搜索策略指导：
  1. 查找**最新时事、世界杯比赛结果、最近新闻、实时数据**等：必须直接且首选使用 **web_search** 工具！
  2. 查找**社交动态、视频字幕、UP主讨论**：必须直接且首选使用 **agent_reach**！
  3. 想要阅读具体的网页/文章详情时，使用 **web_scrape** 传入对应 URL 深度抓取主体文本。

【4. 任务管理工具 (Todo)】
  - todo_write: 发布或更新当前任务计划清单
  - todo_declare_blocker: 声明客观阻塞条件

【5. 社交与媒体平台深度抓取 (Agent-Reach)】
  - agent_reach: 抓取或检索 B站视频（详情/字幕）、YouTube（字幕）、小红书（帖子）、Twitter、V2EX、GitHub、Reddit、雪球等社交与媒体平台的公开内容。
    * 当用户需要提取特定视频的字幕、搜索小红书贴子、或阅读特定推文时，必须优先调度此工具。

【执行规范】
遇到任何需要执行命令、读写文件、搜索代码、网络检索、社交媒体抓取或任务管理的任务，你必须且只能调度上面对应的工具来获取真实结果。严禁自己脑补结果、虚构内容、或凭空捏造事实。必须严格遵守这一工具调度规范！`

// New 基于给定 model 创建一个启用流式输出、并注册了完整工具生态链的强力 LLMAgent。
// 使用 trpc-agent-go 框架内置的工具：
// - claudecode.ToolSet: 代码开发工具集（Bash, Read, Write, Edit, Glob, Grep, WebFetch, WebSearch）
// - duckduckgo.Tool: DuckDuckGo 网络搜索工具（百科/事实性信息）
// - todo.Tool: 任务管理工具
// - command.NewRunCommandTool: Windows CMD 命令执行工具
// - agentreach.NewAgentReachTool: Agent-Reach 社交媒体内容抓取工具
func New(name string, m model.Model) *llmagent.LLMAgent {
	genConfig := model.GenerationConfig{
		Stream: true,
	}

	// 创建 Claude Code ToolSet
	codeToolSet, err := claudecode.NewToolSet(
		claudecode.WithBaseDir("."),     // 工作目录
		claudecode.WithReadOnly(false),  // 启用写入能力
	)
	if err != nil {
		panic("Failed to create claudecode toolset: " + err.Error())
	}

	// 创建 DuckDuckGo 搜索工具
	duckduckgoTool := duckduckgo.NewTool()

	// 创建 Todo 任务工具
	todoTool := todo.New()

	// 创建 Windows CMD 命令执行工具（备用方案）
	runCommandTool := command.NewRunCommandTool()

	// 创建 Agent-Reach 社交媒体内容抓取工具
	agentReachTool := agentreach.NewAgentReachTool()

	// 创建自研免 Key 网页搜索与网页抓取工具 (比 claudecode 自带未配 Key 的更靠谱！)
	webSearchTool := web.NewWebSearchTool()
	webScrapeTool := web.NewWebScrapeTool()

	return llmagent.New(name,
		llmagent.WithModel(m),
		llmagent.WithToolSets([]tool.ToolSet{codeToolSet}), // 使用 ToolSet
		llmagent.WithTools([]tool.Tool{duckduckgoTool, todoTool, runCommandTool, agentReachTool, webSearchTool, webScrapeTool}),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithInstruction(defaultInstruction),
		llmagent.WithMaxToolIterations(5),
		llmagent.WithMaxLLMCalls(10),
	)
}

// Close 关闭 Agent 相关资源（如 ToolSet）
func Close() error {
	return nil
}
