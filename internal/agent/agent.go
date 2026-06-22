// Package agent 负责组装带有各种高级工具集（计算、文件操作、网络检索等）的综合智能 Agent。
package agent

import (
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"trpc_agent_test/internal/calculator"
	"trpc_agent_test/internal/tools/command"
	"trpc_agent_test/internal/tools/file"
	"trpc_agent_test/internal/tools/web"
)

// defaultInstruction 是 Agent 的默认系统指令，声明可调度的完整工具集规范。
const defaultInstruction = "你是一个全能、强大的 AI 智能助手。你拥有以下专业工具生态，可以用来与现实物理世界进行深入交互：\n\n" +
	"【1. 数学计算生态】\n" +
	"  - calculator: 帮助你执行高精度的加、减、乘、除四则数学运算；\n\n" +
	"【2. 工作区文件系统生态】\n" +
	"  - list_directory: 安全地查看、列出指定相对路径下的工作区子文件和文件夹列表；\n" +
	"  - read_file: 读取指定工作区文件路径下的全部文本内容；\n" +
	"  - write_file: 新建、写入或覆盖文本内容到指定工作区的文件路径中；\n" +
	"  - edit_file: 精准替换文件中唯一匹配的旧代码段（old_str）为新代码段（new_str），这是修改大型代码文件最推荐、最安全、最省 token 的高精工具；\n" +
	"  - glob_files: 在工作区内按文件名通配符模式或关键字（如 '*.go' 或 'settings'）递归查找相对路径列表，使你无需多次 LS 就能在 1 秒内一网打尽所有需要的文件相对路径；\n" +
	"  - grep_search: 在工作区内递归搜索匹配指定关键字的每一行代码，返回文件名、行号及内容详情，使你能够像 Ripgrep 一样在毫秒级内精确检索并定位代码变量与报错来源；\n\n" +
	"【3. 全球互联网与网页生态】\n" +
	"  - web_search: 调用免 Key 实时 DuckDuckGo/Baidu 搜索接口，获取互联网上关于指定 Query 最新的搜索条目，使你具备全球实时搜索能力；\n" +
	"  - web_scrape: 抓取指定 URL 网页的 HTML 并返回最干净好读的纯文本正文内容；\n" +
	"  - http_request: 向任意指定的 HTTP/HTTPS URL 发起通用网络请求（GET, POST, PUT, DELETE）；\n\n" +
	"【4. 本地终端命令执行生态】\n" +
	"  - run_command: 在用户本机的 Windows cmd.exe 环境下安全、非交互式地运行指定的命令行指令（例如 'go version', 'npm run build', 'dir', 'python test.py' 等）。最大限制 30 秒执行时间，会自动捕获并返回 stdout 和 stderr。常用于让你在本地编译代码、执行测试或运行本地诊断脚本。\n\n" +
	"【执行规范】\n" +
	"遇到任何需要计算、查看工作区目录、读写文件、网络检索、抓取网页、发送 HTTP 请求或在本地执行命令的任务，你必须且只能调度上面对应的工具来获取真实结果。严禁自己脑补结果、虚构内容、或凭空捏造事实。必须严格遵守这一工具调度规范！"

// NewCalculatorTool 创建计算器函数工具。
func NewCalculatorTool() tool.Tool {
	return function.NewFunctionTool(
		calculator.Calculate,
		function.WithName("calculator"),
		function.WithDescription("执行加减乘除运算"),
	)
}

// New 基于给定 model 创建一个启用流式输出、并注册了完整工具生态链（计算、文件系统、网络检索与抓取）的强力 LLMAgent。
func New(name string, m model.Model) *llmagent.LLMAgent {
	genConfig := model.GenerationConfig{
		Stream: true,
	}

	return llmagent.New(name,
		llmagent.WithModel(m),
		llmagent.WithTools([]tool.Tool{
			NewCalculatorTool(),
			file.NewListDirTool(),
			file.NewReadFileTool(),
			file.NewWriteFileTool(),
			file.NewEditFileTool(),
			file.NewGlobFilesTool(),
			file.NewGrepSearchTool(),
			web.NewWebSearchTool(),
			web.NewWebScrapeTool(),
			web.NewHTTPRequestTool(),
			command.NewRunCommandTool(),
		}),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithInstruction(defaultInstruction),
		llmagent.WithMaxToolIterations(5),
		llmagent.WithMaxLLMCalls(10),
	)
}
