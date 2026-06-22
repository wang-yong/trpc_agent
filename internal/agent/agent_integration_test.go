package agent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
)

// AgentIntegrationTestSuite Agent 集成测试套件
type AgentIntegrationTestSuite struct {
	suite.Suite
	runner runner.Runner
	agent  *llmagent.LLMAgent
	apiKey string
	ctx    context.Context
	cancel context.CancelFunc
}

// SetupSuite 测试套件初始化
func (s *AgentIntegrationTestSuite) SetupSuite() {
	s.apiKey = os.Getenv("OPENAI_API_KEY")
	s.Require().NotEmpty(s.apiKey, "OPENAI_API_KEY 环境变量未设置")

	s.ctx, s.cancel = context.WithTimeout(context.Background(), 60*time.Second)

	// 创建模型
	modelInstance := openai.New("deepseek-chat",
		openai.WithVariant(openai.VariantDeepSeek),
		openai.WithAPIKey(s.apiKey),
	)

	// 创建带计算器工具的 Agent
	s.agent = New("test-assistant", modelInstance)

	s.runner = runner.NewRunner("test-app", s.agent)
}

// TearDownSuite 测试套件清理
func (s *AgentIntegrationTestSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

// TestAgent_CalculatorIntegration 测试 Agent 调用计算器工具
func (s *AgentIntegrationTestSuite) TestAgent_CalculatorIntegration() {
	testCases := []struct {
		name       string
		userInput  string
		assertions func(t *testing.T, response string)
	}{
		{
			name:      "简单加法",
			userInput: "计算 2+3 等于多少",
			assertions: func(t *testing.T, response string) {
				assert.NotEmpty(t, response, "Agent 应该返回响应")
				assert.Contains(t, response, "5", "响应应该包含计算结果 5")
			},
		},
		{
			name:      "简单乘法",
			userInput: "计算 4 乘以 5 等于多少",
			assertions: func(t *testing.T, response string) {
				assert.NotEmpty(t, response, "Agent 应该返回响应")
				assert.Contains(t, response, "20", "响应应该包含计算结果 20")
			},
		},
		{
			name:      "简单减法",
			userInput: "计算 10 减去 3 等于多少",
			assertions: func(t *testing.T, response string) {
				assert.NotEmpty(t, response, "Agent 应该返回响应")
				assert.Contains(t, response, "7", "响应应该包含计算结果 7")
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 执行对话
			events, err := s.runner.Run(s.ctx, "test-user", "test-session", model.NewUserMessage(tc.userInput))
			assert.NoError(t, err, "运行 Agent 应该成功")

			// 收集响应
			var response string
			for event := range events {
				if event.Response != nil && len(event.Response.Choices) > 0 {
					response += event.Response.Choices[0].Delta.Content
				}
			}

			tc.assertions(t, response)
		})
	}
}

// TestAgent_StreamOutput 测试流式输出
func (s *AgentIntegrationTestSuite) TestAgent_StreamOutput() {
	events, err := s.runner.Run(s.ctx, "test-user", "test-session-2", model.NewUserMessage("1+1等于多少"))
	s.Require().NoError(err, "运行 Agent 应该成功")

	chunkCount := 0
	var fullResponse string

	for event := range events {
		if event.Response != nil && len(event.Response.Choices) > 0 {
			content := event.Response.Choices[0].Delta.Content
			if content != "" {
				chunkCount++
				fullResponse += content
			}
		}
	}

	s.Greater(chunkCount, 0, "应该收到至少一个内容 chunk")
	s.NotEmpty(fullResponse, "应该有完整的响应")
}

// TestAgent_MultipleToolCalls 测试多次工具调用
func (s *AgentIntegrationTestSuite) TestAgent_MultipleToolCalls() {
	userMessage := "请帮我计算：1) 5+3 2) 10-4 3) 2*3"

	events, err := s.runner.Run(s.ctx, "test-user", "test-session-3", model.NewUserMessage(userMessage))
	s.Require().NoError(err, "运行 Agent 应该成功")

	var response string
	for event := range events {
		if event.Response != nil && len(event.Response.Choices) > 0 {
			response += event.Response.Choices[0].Delta.Content
		}
	}

	s.NotEmpty(response, "Agent 应该返回响应")
}

// TestAgent_ToolDefinition 测试工具定义
func (s *AgentIntegrationTestSuite) TestAgent_ToolDefinition() {
	tools := s.agent.UserTools()
	s.NotEmpty(tools, "Agent 应该有工具")
	s.Len(tools, 1, "应该只有一个工具")

	decl := tools[0].Declaration()
	s.Equal("calculator", decl.Name, "工具名称应该是 calculator")
	s.Contains(decl.Description, "加减乘除", "工具描述应该包含运算说明")
}

// TestRunIntegrationTests 集成测试入口
func TestRunIntegrationTests(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	suite.Run(t, new(AgentIntegrationTestSuite))
}

// TestMain 测试主函数
func TestMain(m *testing.M) {
	code := m.Run()
	fmt.Printf("测试完成，退出码: %d\n", code)
	os.Exit(code)
}
