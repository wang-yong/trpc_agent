package command

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// CommandInput 是运行命令工具的入参
type CommandInput struct {
	Command string `json:"command" description:"要在当前工作区执行的终端命令（非交互式，例如 'go version', 'npm run build'）"`
}

// CommandOutput 是运行命令工具的出参
type CommandOutput struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}

// RunCommand 物理执行终端命令
func RunCommand(ctx context.Context, input *CommandInput) (CommandOutput, error) {
	cmdStr := strings.TrimSpace(input.Command)
	if cmdStr == "" {
		return CommandOutput{Success: false, Message: "命令不能为空"}, nil
	}

	// 1. 安全防护防线：禁止执行某些具有超强毁灭性或不可逆的系统根目录删除格式化指令
	lowerCmd := strings.ToLower(cmdStr)
	dangerousKeywords := []string{
		"del /", "rmdir /", "format ", "mkfs", "rm -rf", "shred", "dd ",
	}
	for _, kw := range dangerousKeywords {
		if strings.Contains(lowerCmd, kw) {
			return CommandOutput{
				Command: cmdStr,
				Success: false,
				Message: "安全沙箱硬拒绝：检测到包含危险命令关键字 " + kw,
			}, nil
		}
	}

	// 2. 超时防线：建立 30s 自动硬熔断上下文，防止交互式等待命令永久卡死后台协程进程
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Windows 环境下调用 cmd /c
	cmd := exec.CommandContext(runCtx, "cmd", "/c", cmdStr)

	stdoutBuf := &strings.Builder{}
	stderrBuf := &strings.Builder{}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err := cmd.Run()

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
	exitCode := 0

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return CommandOutput{
				Command:  cmdStr,
				Stdout:   stdoutStr,
				Stderr:   stderrStr + "\n[SECURITY WARNING] 命令执行超时（最长 30s 限制），已被后台强制熔断强杀！",
				ExitCode: -1,
				Success:  false,
				Message:  "命令执行超时被终止",
			}, nil
		}
		// 获取真实的退出代码
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return CommandOutput{
		Command:  cmdStr,
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
		ExitCode: exitCode,
		Success:  err == nil,
		Message:  "命令执行成功",
	}, nil
}

// NewRunCommandTool 实例化终端命令调用工具
func NewRunCommandTool() tool.Tool {
	return function.NewFunctionTool(
		RunCommand,
		function.WithName("run_command"),
		function.WithDescription("在用户本机的 Windows cmd.exe 环境下安全、非交互式地运行指定的命令行指令（例如 'go version', 'npm run build', 'dir' 等）"),
	)
}
