package main

import (
	"log"
	"net/http"
	"os"

	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"

	agentpkg "trpc_agent_test/internal/agent"
	"trpc_agent_test/internal/server"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// 可用模型列表（硅基流动平台）
	modelDefs := []struct {
		Name        string
		DisplayName string
	}{
		{"deepseek-ai/DeepSeek-V3", "DeepSeek V3"},
		{"Qwen/Qwen2.5-72B-Instruct", "Qwen 2.5 72B"},
		{"Qwen/Qwen2.5-Coder-32B-Instruct", "Qwen Coder 32B"},
	}

	// 为每个模型创建独立的 Runner
	var configs []server.ModelConfig
	for _, md := range modelDefs {
		m := openai.New(md.Name,
			openai.WithAPIKey(apiKey),
			openai.WithBaseURL(baseURL),
		)
		assistant := agentpkg.New("assistant", m)
		r := runner.NewRunner("web-"+md.Name, assistant)
		configs = append(configs, server.ModelConfig{
			Name:        md.Name,
			DisplayName: md.DisplayName,
			Runner:      r,
		})
	}

	// 创建 Web 服务（默认使用 DeepSeek V3）
	srv, err := server.New(configs, "deepseek-ai/DeepSeek-V3")
	if err != nil {
		log.Fatalf("初始化 Web 服务失败: %v", err)
	}

	log.Printf("Web 服务已启动，访问 http://localhost%s", addr)
	log.Printf("可用模型: %d 个，默认: DeepSeek V3", len(modelDefs))
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("服务异常退出: %v", err)
	}
}
