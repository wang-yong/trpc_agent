package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"

	agentpkg "trpc_agent_test/internal/agent"
)

func main() {
	// 从环境变量获取 API 配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量")
	}

	// Base URL，默认使用硅基流动（OpenAI 兼容接口）
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	// 模型名，默认使用硅基流动上的 DeepSeek-V3
	modelName := os.Getenv("OPENAI_MODEL")
	if modelName == "" {
		modelName = "deepseek-ai/DeepSeek-V3"
	}

	// 创建模型实例（OpenAI 兼容模式）
	modelInstance := openai.New(modelName,
		openai.WithAPIKey(apiKey),
		openai.WithBaseURL(baseURL),
	)

	// 创建带计算器工具的 Agent
	assistant := agentpkg.New("assistant", modelInstance)

	// 创建 Runner
	r := runner.NewRunner("calculator-app", assistant)

	// 执行对话
	ctx := context.Background()
	userMessage := model.NewUserMessage("计算 2+3 等于多少")

	events, err := r.Run(ctx, "user-001", "session-001", userMessage)
	if err != nil {
		log.Fatal(err)
	}

	// 处理事件流
	fmt.Print("AI 回复: ")
	for event := range events {
		if event.Object == "chat.completion.chunk" && len(event.Response.Choices) > 0 {
			content := event.Response.Choices[0].Delta.Content
			if content != "" {
				fmt.Print(content)
			}
		}
	}
	fmt.Println()
}
