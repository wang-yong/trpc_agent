package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	"trpc.group/trpc-go/trpc-agent-go/session/summary"

	agentpkg "trpc_agent_test/internal/agent"
	"trpc_agent_test/internal/context"
	"trpc_agent_test/internal/server"
)

// model 包引入
import modelpkg "trpc.group/trpc-go/trpc-agent-go/model"

// LoggingReadCloser 同步、零卡顿地将大模型流式 SSE 字节流复制写入日志文件（TeeReader 机制）
type LoggingReadCloser struct {
	original  io.ReadCloser
	tee       io.Reader
	logFile   *os.File
	server    *server.Server
	sessionID string
}

func (l *LoggingReadCloser) Read(p []byte) (n int, err error) {
	n, err = l.tee.Read(p)
	if n > 0 && l.server != nil && l.sessionID != "" {
		chunk := string(p[:n])
		// 1.5s 零阻塞极速嗅探：通过定位 "usage" 直接取出 SiliconFlow 吐回的最真实的单次用量，过滤底层库自带的全局累加污染
		if idx := strings.Index(chunk, `"usage"`); idx != -1 {
			sub := chunk[idx:]
			prompt := extractInt(sub, `"prompt_tokens"`)
			comp := extractInt(sub, `"completion_tokens"`)
			total := extractInt(sub, `"total_tokens"`)
			if prompt > 0 {
				l.server.SetRealUsage(l.sessionID, &modelpkg.Usage{
					PromptTokens:     prompt,
					CompletionTokens: comp,
					TotalTokens:      total,
				})
			}
		}
	}
	return n, err
}

func (l *LoggingReadCloser) Close() error {
	err := l.original.Close()
	if l.logFile != nil {
		_, _ = l.logFile.WriteString("\n=== [LLM TRANSACTION DONE] ===\n\n")
		_ = l.logFile.Close()
	}
	return err
}

// extractInt 用于从子串中以绝对 0-CPU 额外开销的姿态快速抠出指定的数字
func extractInt(str, key string) int {
	idx := strings.Index(str, key)
	if idx == -1 {
		return 0
	}
	sub := str[idx+len(key):]
	start := -1
	for i, c := range sub {
		if c >= '0' && c <= '9' {
			if start == -1 {
				start = i
			}
		} else if start != -1 {
			var val int
			_, _ = fmt.Sscanf(sub[start:i], "%d", &val)
			return val
		}
	}
	return 0
}

// LoggingRoundTripper 拦截 http.DefaultTransport 传输层，极速拦截、解析大模型接口交互
type LoggingRoundTripper struct {
	Proxied http.RoundTripper
}

func (l *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if l.Proxied == nil {
		l.Proxied = http.DefaultTransport
	}

	// 智能断属：判断当前 HTTP 交互是否属于硅基流动、OpenAI 或者是其他的 Chat API 的上行通信
	isLLM := strings.Contains(req.URL.Host, "siliconflow") || strings.Contains(req.URL.Host, "openai") || strings.Contains(req.URL.Path, "/chat/completions")

	var reqBodyBytes []byte
	if isLLM && req.Body != nil {
		var err error
		reqBodyBytes, err = io.ReadAll(req.Body)
		if err == nil {
			// 一定要回写 Body，否则后续库就无法读取了
			req.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		}
	}

	resp, err := l.Proxied.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if isLLM && resp.Body != nil {
		var s *server.Server
		if val := req.Context().Value("approval_manager"); val != nil {
			s, _ = val.(*server.Server)
		}
		var sessionID string
		if val := req.Context().Value("session_id"); val != nil {
			sessionID, _ = val.(string)
		}

		_ = os.MkdirAll("bin", 0755)
		logFile, logErr := os.OpenFile("bin/llm_io.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if logErr == nil {
			_, _ = logFile.WriteString("\n========================================================================\n")
			fmt.Fprintf(logFile, "[TIMESTAMP]   %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Fprintf(logFile, "[LLM REQ API] %s %s\n", req.Method, req.URL.String())
			
			if len(reqBodyBytes) > 0 {
				_, _ = logFile.WriteString("[LLM REQUEST PROMPT PAYLOAD (UP)]\n")
				var prettyJSON bytes.Buffer
				if jsonErr := json.Indent(&prettyJSON, reqBodyBytes, "", "  "); jsonErr == nil {
					logFile.Write(prettyJSON.Bytes())
				} else {
					logFile.Write(reqBodyBytes)
				}
				_, _ = logFile.WriteString("\n")
			}
			_, _ = logFile.WriteString("[LLM RESPONSE STREAM (DOWN)]\n")

			// 利用 TeeReader 机制进行物理拦截，零拖慢、零影响打字打字机体验
			resp.Body = &LoggingReadCloser{
				original:  resp.Body,
				tee:       io.TeeReader(resp.Body, logFile),
				logFile:   logFile,
				server:    s,
				sessionID: sessionID,
			}
		}
	}

	return resp, nil
}

func main() {
	// 部署底层大模型协议拦截探针，全自动、100% 毫无保留落盘所有历史上下文 Payload 与下行流
	http.DefaultTransport = &LoggingRoundTripper{Proxied: http.DefaultTransport}

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

	// 为每个模型创建独立的 Runner，集成框架原生大模型自动摘要、50%警戒线满水位滑动窗口截断压缩，极力守护 Token 账单
	var configs []server.ModelConfig
	for _, md := range modelDefs {
		m := openai.New(md.Name,
			openai.WithAPIKey(apiKey),
			openai.WithBaseURL(baseURL),
		)
		assistant := agentpkg.New("assistant", m)

		// 1. 创建原生的 SessionSummarizer 摘要生成器
		summarizer := summary.NewSummarizer(m,
			summary.WithContextThreshold(
				summary.WithContextThresholdRatio(0.5), // 黄金 50% 满水位警戒线，一旦达到，立即触发大模型自动摘要
			),
			summary.WithName("session-compressor"),
			summary.WithMaxSummaryWords(150), // 限制摘要描述在 150 字以内，精炼不失重点
		)

		// 1.5. 创建智能边界压缩器（话题切换时立即触发压缩）
		smartCompressor := context.WrapSummarizer(summarizer, m,
			context.SmartCompressorConfig{
				Enabled:  true,
				DebugMode: false, // 生产环境建议关闭
				CompressionThresholds: map[context.TopicRelation]context.CompressionThreshold{
					context.TopicUnrelated: {
						TokenThreshold:      1500, // 话题完全无关时，更低的触发阈值
						EventThreshold:      8,
						SummaryWords:        100,
						PreserveRecentCount: 2,
					},
					context.TopicWeakRelated: {
						TokenThreshold:      3000,
						EventThreshold:      15,
						SummaryWords:        150,
						PreserveRecentCount: 3,
					},
					context.TopicStrongRelated: {
						TokenThreshold:      5000,
						EventThreshold:      20,
						SummaryWords:        200,
						PreserveRecentCount: 5,
					},
				},
			},
		)

		// 2. 创建配置了 Summarizer 的 inmemory Session 历史管理器
		sessionSvc := inmemory.NewSessionService(
			inmemory.WithSummarizer(summarizer),
			inmemory.WithSessionEventLimit(100), // 限制每个 Session 最大的缓存完整事件数，防止无限积压
		)

		// 3. 构建 Runner 时传入我们高抗压、高自愈的 sessionSvc
		r := runner.NewRunner("web-"+md.Name, assistant,
			runner.WithSessionService(sessionSvc),
		)

		configs = append(configs, server.ModelConfig{
			Name:        md.Name,
			DisplayName: md.DisplayName,
			Runner:      r,
			SmartCompressor: smartCompressor, // 附加智能压缩器
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
