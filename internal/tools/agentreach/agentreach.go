package agentreach

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// AgentReachInput 是 Agent-Reach 工具的结构化入参
type AgentReachInput struct {
	Action   string `json:"action" description:"具体操作类型：'search'(搜索平台内容，例如视频、帖子等) 或 'read'(深度阅读详情或提取视频字幕)"`
	Platform string `json:"platform" description:"目标社交/媒体/视频平台，必须是以下之一: 'bilibili', 'youtube', 'xiaohongshu', 'twitter', 'github', 'v2ex', 'reddit', 'xueqiu'"`
	Target   string `json:"target" description:"具体检索内容或URL：如果是 search 则是搜索关键字；如果是 read 则是对应的视频 URL/视频ID/文章URL"`
}

// AgentReachOutput 是 Agent-Reach 工具的出参
type AgentReachOutput struct {
	Stdout   string `json:"stdout" description:"工具执行后返回的标准输出，包含抓取的内容、字幕或搜索结果"`
	Success  bool   `json:"success" description:"工具是否执行成功"`
	Message  string `json:"message" description:"状态消息"`
}

// ExecuteAgentReach 将高层 action/platform/target 映射并物理执行底层真正的 CLI 动作
func ExecuteAgentReach(ctx context.Context, input *AgentReachInput) (AgentReachOutput, error) {
	action := strings.ToLower(strings.TrimSpace(input.Action))
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	target := strings.TrimSpace(input.Target)

	if action == "" || platform == "" || target == "" {
		return AgentReachOutput{
			Success: false,
			Message: "参数 action, platform 和 target 均不能为空",
		}, nil
	}

	// 1. 建立 45s 超时限制，防抓取防风控导致的长时间挂起
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	// 2. 将动作和平台映射到底层安装并配置好的实际命令
	switch platform {
	case "bilibili", "b站", "bili":
		if action == "search" {
			// 使用 bili-cli: bili search "QUERY" --type video -n 5
			cmd = exec.CommandContext(runCtx, "bili", "search", target, "--type", "video", "-n", "5")
		} else if action == "read" {
			// 读取视频详情：bili video BVxxx
			// 如果传入的是完整 URL，提取其中的 BV 号
			bv := target
			if strings.Contains(target, "bilibili.com/video/") {
				parts := strings.Split(target, "bilibili.com/video/")
				if len(parts) > 1 {
					bv = strings.Split(parts[1], "/")[0]
					bv = strings.Split(bv, "?")[0]
				}
			}
			cmd = exec.CommandContext(runCtx, "bili", "video", bv)
		} else {
			return AgentReachOutput{Success: false, Message: "Bilibili 暂不支持此类操作，目前仅支持 'search'(搜索) 和 'read'(读取BV号或URL)"}, nil
		}

	case "youtube", "yt":
		if action == "read" {
			// 下载 YouTube 双语字幕，使用 yt-dlp
			cmd = exec.CommandContext(runCtx, "yt-dlp", "--write-sub", "--write-auto-sub", "--sub-lang", "zh-Hans,zh,en", "--skip-download", "-o", "%TEMP%\\%(id)s", target)
		} else if action == "search" {
			// YouTube 搜索：yt-dlp --dump-json "ytsearch5:query"
			cmd = exec.CommandContext(runCtx, "yt-dlp", "--dump-json", "ytsearch5:"+target)
		} else {
			return AgentReachOutput{Success: false, Message: "YouTube 目前仅支持 'search' 或 'read'(提取视频字幕)"}, nil
		}

	case "v2ex":
		// V2EX API 直连，使用 curl: curl -s "..." -H "User-Agent: agent-reach/1.0"
		url := "https://www.v2ex.com/api/topics/hot.json"
		if action == "read" {
			if strings.Contains(target, "v2ex.com/t/") {
				parts := strings.Split(target, "/t/")
				if len(parts) > 1 {
					id := strings.Split(parts[1], "#")[0]
					id = strings.Split(id, "?")[0]
					url = "https://www.v2ex.com/api/topics/show.json?id=" + id
				}
			} else {
				url = "https://www.v2ex.com/api/topics/show.json?id=" + target
			}
		}
		cmd = exec.CommandContext(runCtx, "curl", "-s", url, "-H", "User-Agent: agent-reach/1.0")

	case "twitter", "x":
		if action == "search" {
			cmd = exec.CommandContext(runCtx, "opencli", "twitter", "search", target, "-f", "yaml")
		} else if action == "read" {
			cmd = exec.CommandContext(runCtx, "opencli", "twitter", "tweet", target, "-f", "yaml")
		} else {
			return AgentReachOutput{Success: false, Message: "Twitter 目前仅支持 'search' 或 'read'"}, nil
		}

	case "xiaohongshu", "xhs", "小红书":
		if action == "search" {
			cmd = exec.CommandContext(runCtx, "opencli", "xiaohongshu", "search", target, "-f", "yaml")
		} else if action == "read" {
			cmd = exec.CommandContext(runCtx, "opencli", "xiaohongshu", "note", target, "-f", "yaml")
		} else {
			return AgentReachOutput{Success: false, Message: "小红书目前仅支持 'search' 或 'read'"}, nil
		}

	default:
		// 3. 兜底策略：如果传入的是链接，直接走通用 Jina Reader 网页阅读：curl -s "https://r.jina.ai/URL"
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			cmd = exec.CommandContext(runCtx, "curl", "-s", "https://r.jina.ai/"+target)
		} else {
			return AgentReachOutput{
				Success: false,
				Message: fmt.Sprintf("暂未支持对平台 '%s' 进行 '%s' 操作，且输入内容不是有效的网页链接", platform, action),
			}, nil
		}
	}

	// 如果上下文注入了工作区根目录，将执行上下文切换至该目录
	if wd, ok := ctx.Value("workspace_root").(string); ok && wd != "" {
		cmd.Dir = wd
	}

	stdoutBuf := &strings.Builder{}
	stderrBuf := &strings.Builder{}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err := cmd.Run()

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return AgentReachOutput{
				Stdout:  stdoutStr,
				Success: false,
				Message: "执行超时：抓取请求耗时过长（已达 45s 超时限制），已被系统强行熔断中断！建议重新发起或检查您的网络代理。",
			}, nil
		}
		// 检查底层 CLI 是否未找到
		if strings.Contains(err.Error(), "executable file not found") {
			return AgentReachOutput{
				Stdout:  "",
				Success: false,
				Message: fmt.Sprintf("执行失败：未在系统环境中找到该操作依赖的底层命令行工具。请确保该工具（例如: bili, opencli, yt-dlp, curl）已被正常安装且处于系统的 PATH 环境变量中！"),
			}, nil
		}
		return AgentReachOutput{
			Stdout:  stdoutStr + "\n[错误详情]\n" + stderrStr,
			Success: false,
			Message: "执行失败：" + err.Error(),
		}, nil
	}

	return AgentReachOutput{
		Stdout:  stdoutStr,
		Success: true,
		Message: "执行成功",
	}, nil
}

// NewAgentReachTool 实例化 Agent-Reach 深度网络抓取/检索工具
func NewAgentReachTool() tool.Tool {
	return function.NewFunctionTool(
		ExecuteAgentReach,
		function.WithName("agent_reach"),
		function.WithDescription("社交媒体与视频平台深度抓取/检索工具（Agent-Reach）。免 API 费用获取 B站视频（详情/搜索）、YouTube（搜索/字幕）、小红书（帖子）、Twitter、V2EX、GitHub、Reddit、雪球等社交或媒体平台的公开内容。"),
	)
}
