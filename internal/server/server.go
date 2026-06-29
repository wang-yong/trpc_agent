// Package server provides HTTP/SSE-based web chat service with multi-model support,
// session management, and skill templates.
package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"

	"gopkg.in/yaml.v3"
)

const webUserID = "web-user"

// Context keys for cross-package communication
const (
	ApprovalManagerKey = "approval_manager"
	SessionIDKey       = "session_id"
	WriteSSEFuncKey    = "write_sse_func"
	FlushChanKey       = "flush_chan"
)

type ApprovalEvent struct {
	ID        string
	ToolName  string
	Arguments string
}

//go:embed static
var staticFS embed.FS

// ModelConfig defines a model configuration for chat.
type ModelConfig struct {
	Name            string            // Unique identifier, e.g., "deepseek-ai/DeepSeek-V3"
	DisplayName     string            // Display name, e.g., "DeepSeek V3"
	Runner          runner.Runner
	SmartCompressor interface{} // Smart compressor (optional)
}

// SessionInfo 记录会话的元数据。
type SessionInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Model     string `json:"model"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// SkillTemplate 定义快捷场景模板。
type SkillTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Prompt      string `json:"prompt"`
}

// TokenRecord 记录一次对话的 token 消耗。
type TokenRecord struct {
	ID               int64  `json:"id"`
	SessionID        string `json:"session_id"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	Timestamp        int64  `json:"timestamp"`
	ReadableTime     string `json:"readable_time"`
	Question         string `json:"question,omitempty"`
}

// ModelTokenStat 是按模型聚合的 token 统计。
type ModelTokenStat struct {
	Model            string `json:"model"`
	DisplayName      string `json:"display_name"`
	RequestCount     int    `json:"request_count"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

// ApprovalRequest 记录一次高危工具人机审批交互。
type ApprovalRequest struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	ToolName  string    `json:"tool_name"`
	Arguments string    `json:"arguments"`
	Status    string    `json:"status"` // "pending", "approved", "rejected"
	Response  chan bool `json:"-"`
}

// Server 封装 HTTP 路由、多模型 Runner 与会话管理。
type Server struct {
	models       []ModelConfig
	modelMap     map[string]runner.Runner
	defaultModel string
	mux          *http.ServeMux
	mu           sync.RWMutex
	sessions     map[string]map[string]*SessionInfo // 外层 Key 是 userID，内层 Key 是 sessionID
	skills       []SkillTemplate
	// Token 统计
	tokenMu      sync.Mutex
	tokenRecords []TokenRecord
	tokenIDSeq   int64

	// 安全审批注册表
	approvalsMu sync.Mutex
	approvals   map[string]*ApprovalRequest

	// SSE 并发写锁，保证主协程与工具协程在并发推送事件时绝对物理安全，彻底绝杀 Concurrent Write Panic！
	sseMu sync.Mutex

	// 真实用量临时中转寄存器，用来彻底清洗由于底层库带来的全局累加污染，保航前端显示
	realUsageMu sync.Mutex
	realUsage   map[string]*model.Usage // session_id -> usage
}

// chatRequest 是 /api/chat 接口的请求体。
type chatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
	SkillID   string `json:"skill_id"`
}

// New 创建一个 Web 服务实例。
func New(configs []ModelConfig, defaultModel string) (*Server, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("至少需要一个模型配置")
	}

	sub, err := fs.Sub(staticFS, "static/dist")
	if err != nil {
		return nil, fmt.Errorf("加载静态资源失败: %w", err)
	}

	modelMap := make(map[string]runner.Runner)
	for _, c := range configs {
		modelMap[c.Name] = c.Runner
	}

	if defaultModel == "" {
		defaultModel = configs[0].Name
	}

	s := &Server{
		models:       configs,
		modelMap:     modelMap,
		defaultModel: defaultModel,
		mux:          http.NewServeMux(),
		sessions:     make(map[string]map[string]*SessionInfo),
		skills:       defaultSkills(),
		approvals:    make(map[string]*ApprovalRequest),
		realUsage:    make(map[string]*model.Usage),
	}

	s.loadTokenStats() // 加载历史 token 统计

	s.mux.HandleFunc("/", s.handleStatic(sub))
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/models", s.handleModels)
	s.mux.HandleFunc("/api/skills", s.handleSkills)
	s.mux.HandleFunc("/api/token-stats", s.handleTokenStats)
	s.mux.HandleFunc("/api/approvals/respond", s.handleApprovalRespond)
	s.mux.HandleFunc("/api/settings", s.handleSettings)
	s.mux.HandleFunc("/api/workspace/files", s.handleWorkspaceFiles)
	s.mux.HandleFunc("/api/workspace/select-dialog", s.handleSelectWorkspaceDialog)
	s.mux.HandleFunc("/api/workspace/file-preview", s.handleWorkspaceFilePreview)
	s.mux.HandleFunc("/api/workspace/file-raw", s.handleWorkspaceFileRaw)
	return s, nil
}

// defaultSkills 返回默认的快捷场景模板。
func defaultSkills() []SkillTemplate {
	return []SkillTemplate{
		{
			ID:          "app-dev",
			Name:        "应用开发",
			Description: "全栈应用开发助手",
			Icon:        "\xf0\x9f\x92\xbb",
			Prompt:      "你是一个全栈开发助手，擅长 Web、移动端应用开发。请提供清晰的代码实现和架构建议。",
		},
		{
			ID:          "project-understanding",
			Name:        "项目理解",
			Description: "代码分析与项目理解",
			Icon:        "\xf0\x9f\x93\x81",
			Prompt:      "你是一个代码分析助手，擅长理解项目结构、解读代码逻辑、提供技术文档。",
		},
		{
			ID:          "game-creation",
			Name:        "游戏创意",
			Description: "游戏设计与开发",
			Icon:        "\xf0\x9f\x8e\xae",
			Prompt:      "你是一个游戏设计助手，擅长游戏玩法设计、平衡性分析、游戏开发实现方案。",
		},
		{
			ID:          "tool-scripting",
			Name:        "工具脚本",
			Description: "自动化脚本编写",
			Icon:        "\xf0\x9f\x94\xa7",
			Prompt:      "你是一个脚本编写助手，擅长编写自动化脚本、CLI 工具、系统运维脚本。代码请使用 Go 或 Python。",
		},
	}
}

// ServeHTTP 实现 http.Handler 接口。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleChat 处理聊天请求，以 SSE 形式流式返回 AI 回复。
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "web-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	userID := getUserID(r)

	// 确定模型
	modelName := req.Model
	if modelName == "" {
		modelName = s.defaultModel
	}
	rnr, ok := s.modelMap[modelName]
	if !ok {
		rnr = s.modelMap[s.defaultModel]
		modelName = s.defaultModel
	}

	// 应用技能模板
	message := req.Message
	if req.SkillID != "" {
		for _, sk := range s.skills {
			if sk.ID == req.SkillID {
				message = sk.Prompt + "\n\n用户请求：" + req.Message
				break
			}
		}
	}

	// 记录会话
	s.mu.Lock()
	userSessions := s.getUserSessionsLocked(userID)
	sess, exists := userSessions[sessionID]
	if !exists {
		title := req.Message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30]) + "..."
		}
		sess = &SessionInfo{
			ID:        sessionID,
			Title:     title,
			Model:     modelName,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		}
		userSessions[sessionID] = sess
	} else {
		sess.Model = modelName
		sess.UpdatedAt = time.Now().Unix()
	}
	s.saveUserSessionsLocked(userID) // 持久化会话
	s.mu.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 动态注入当前的真实北京时间与星期上下文，解决大模型无法感知时间、拿着过期日期进行网络搜索等经典时空幻觉问题
	timeContext := fmt.Sprintf("[系统时间上下文]\n- 当前真实北京时间: %s\n- 当前星期: %s\n\n[用户提问]\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		time.Now().Weekday().String(),
		message,
	)

	writeSSEFunc := func(event string, payload any) {
		s.writeSSE(w, flusher, event, payload)
	}

	// 深度安全防护：上下文动态装配注入。使用永不断连的 Background Context 挂载运行 ReAct 推理，
	// 彻底免疫任何由于前端长连接挂起、网络抖动超时引起 context canceled 导致的引擎中断！
	workspaceRoot := s.getUserWorkspaceRoot(userID)

	flushChan := make(chan *ApprovalEvent, 10)
	ctx := context.WithValue(context.Background(), ApprovalManagerKey, s)
	ctx = context.WithValue(ctx, WriteSSEFuncKey, writeSSEFunc)
	ctx = context.WithValue(ctx, SessionIDKey, sessionID)
	ctx = context.WithValue(ctx, FlushChanKey, flushChan)
	ctx = context.WithValue(ctx, "workspace_root", workspaceRoot)

	events, err := rnr.Run(ctx, userID, sessionID, model.NewUserMessage(timeContext))
	if err != nil {
		s.writeSSE(w, flusher, "error", map[string]string{"message": err.Error()})
		return
	}

	var (
		lastUsage           *model.Usage
		accumulatedThought  strings.Builder
		accumulatedResponse strings.Builder
		toolCallsMap        = make(map[string]string) // 聚合流式 arguments：tool_id -> arguments
		toolNamesMap        = make(map[string]string) // 聚合流式 tool_name：tool_id -> name
		accumulatedObs      []map[string]string       // 聚合工具执行结果

		// 解决流式 Arguments 传输过程中除第一帧外，ID 和 Name 为空的追溯机制
		indexToID           = make(map[int]string)
		indexToName         = make(map[int]string)

		// 极其稳健的有状态流式 XML/JSON 工具调用剥离器，解决分词碎片穿透过滤器的终极 Bug
		xmlBuffer           strings.Builder
		inXmlBlock          = false
	)

	// 初始化活动工作区目录指纹变动监听哨兵，免去用户手动刷新
	lastFingerprint, _ := getWorkspaceFingerprint(workspaceRoot)
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		var ev *event.Event
		var appEv *ApprovalEvent
		var ok bool

		select {
		case ev, ok = <-events:
			if !ok {
				// events 管道关闭，大模型整体执行流结束
				goto chatDone
			}

			if ev.Error != nil {
				s.writeSSE(w, flusher, "error", map[string]string{"message": ev.Error.Message})
				continue
			}
			if ev.Usage != nil {
				// 优先使用物理套接字拦截到最纯正无污染单次用量，清洗底层库大模型客户端在内存中对所有会话的脏累计错误
				if realU := s.GetRealUsage(sessionID); realU != nil {
					ev.Usage = realU
				}
				lastUsage = ev.Usage
			}

			// 1. 推送大模型流式推理/深度思考链 (Reasoning/Thinking)
			if len(ev.Choices) > 0 {
				reasoning := ev.Choices[0].Delta.ReasoningContent
				if reasoning != "" {
					accumulatedThought.WriteString(reasoning)
					s.writeSSE(w, flusher, "thought", map[string]string{"content": reasoning})
				}
			}

			// 2. 拦截并推送流式工具调用请求 (Tool Call Request)
			if len(ev.Choices) > 0 {
				var tcs []model.ToolCall
				if len(ev.Choices[0].Delta.ToolCalls) > 0 {
					tcs = ev.Choices[0].Delta.ToolCalls
				} else if len(ev.Choices[0].Message.ToolCalls) > 0 {
					tcs = ev.Choices[0].Message.ToolCalls
				}

				if len(tcs) > 0 {
					for _, tc := range tcs {
						idx := 0
						if tc.Index != nil {
							idx = *tc.Index
						}

						// 追溯并映射流式索引对应的真实 ID 和 Name
						if tc.ID != "" {
							indexToID[idx] = tc.ID
						}
						if tc.Function.Name != "" {
							indexToName[idx] = tc.Function.Name
						}

						id := indexToID[idx]
						name := indexToName[idx]
						args := string(tc.Function.Arguments)

						// 自动剔除第三方服务商（如硅基流动）损坏的双大括号 {{ ... }} 为单大括号，确保 valid JSON
						if strings.HasPrefix(args, "{{") && strings.HasSuffix(args, "}}") && !strings.HasPrefix(args, "{{{") {
							args = strings.TrimPrefix(args, "{")
							args = strings.TrimSuffix(args, "}")
						}

						// 累加以便于日志记录
						if id != "" {
							toolCallsMap[id] = toolCallsMap[id] + args
							if name != "" {
								toolNamesMap[id] = name
							}
						}

						s.writeSSE(w, flusher, "tool_call", map[string]any{
							"id":        id,
							"name":      name,
							"arguments": args,
						})
					}
				}
			}

			// 3. 推送大模型流式普通回复文本 (Delta Content)
			if len(ev.Choices) > 0 {
				content := ev.Choices[0].Delta.Content
				if content != "" {
					// 极致清理：过滤流式文本开头可能由于前一回合 ToolCall 结束而残留的闭合大括号、右中括号、逗号等符号
					content = strings.TrimPrefix(content, "}")
					content = strings.TrimPrefix(content, "\n}")
					content = strings.TrimPrefix(content, "\r\n}")
					content = strings.TrimPrefix(content, "],")
					content = strings.TrimPrefix(content, "},")
					content = strings.TrimPrefix(content, ",")

					xmlBuffer.WriteString(content)
					currentText := xmlBuffer.String()

					// 如果尚未进入 XML 丢弃块
					if !inXmlBlock {
						if idx := strings.Index(currentText, "<tool_call>"); idx != -1 {
							// 将 <tool_call> 标签之前的所有合法人类对话文本正常推送
							before := currentText[:idx]
							if before != "" && !isDanglingJsonChunk(before) {
								accumulatedResponse.WriteString(before)
								s.writeSSE(w, flusher, "delta", map[string]string{"content": before})
							}
							// 状态转换为已进入 XML 丢弃块
							inXmlBlock = true
							xmlBuffer.Reset()
							xmlBuffer.WriteString(currentText[idx:])
						} else {
							// 确认没有工具标签开始符号，且不含有小于号泄漏前缀时，立即安全下发当前内容
							trimmed := strings.TrimSpace(currentText)
							if !strings.HasPrefix(trimmed, "<") {
								accumulatedResponse.WriteString(currentText)
								s.writeSSE(w, flusher, "delta", map[string]string{"content": currentText})
								xmlBuffer.Reset()
							}
						}
					}

					// 如果已经处于 XML 丢弃块内，等待结束标签并清洗
					if inXmlBlock {
						currentText = xmlBuffer.String()
						if idx := strings.Index(currentText, "</tool_call>"); idx != -1 {
							// 找到了结束标签，剥离并丢弃工具块，将结束标签之后的内容捞出来
							after := currentText[idx+len("</tool_call>"):]
							inXmlBlock = false
							xmlBuffer.Reset()
							xmlBuffer.WriteString(after)

							// 递归或二次检查 after 中是否包含新的开始标签，没有则直接下发普通干净回答
							currentText = after
							if strings.Index(currentText, "<tool_call>") == -1 {
								trimmed := strings.TrimSpace(currentText)
								if !strings.HasPrefix(trimmed, "<") && !isDanglingJsonChunk(currentText) {
									accumulatedResponse.WriteString(currentText)
									s.writeSSE(w, flusher, "delta", map[string]string{"content": currentText})
									xmlBuffer.Reset()
								}
							}
						}
					}
				}
			}

			// 4. 拦截并推送物理工具执行完成并返回的结果 (Tool Observation)
			if ev.Object == model.ObjectTypeToolResponse && len(ev.Choices) > 0 {
				// 工具执行返回了物理结果，说明当前大模型思考和工具调度回合已 100% 结束
				// 立刻清空并重置有状态的流式 XML 剥离器缓存，防止残留的垃圾逗号、括号污染下一回合！
				xmlBuffer.Reset()
				inXmlBlock = false

				for _, choice := range ev.Choices {
					if choice.Message.Role == model.RoleTool {
						accumulatedObs = append(accumulatedObs, map[string]string{
							"id":      choice.Message.ToolID,
							"name":    choice.Message.ToolName,
							"content": choice.Message.Content,
						})
						s.writeSSE(w, flusher, "observation", map[string]any{
							"id":      choice.Message.ToolID,
							"name":    choice.Message.ToolName,
							"content": choice.Message.Content,
						})
					}
				}
			}

		case appEv = <-flushChan:
			// 【终极唤醒自愈防线】主协程亲自被 flushChan 通道唤醒！
			// 主协程在 100% 绝对清醒、绝对活跃的工作上下文中，亲自执行对 ResponseWriter 的写入和 Flush，
			// 强力逼迫操作系统 TCP 层、Vite 代理层和 nginx 瞬间清空并网发出卡住的 approval_request！
			fmt.Printf("[DEBUG APPROVAL] 主协程亲自接收并推送审批事件: ID=%s, Tool=%s\n", appEv.ID, appEv.ToolName)
			s.writeSSE(w, flusher, "approval_request", map[string]any{
				"id":        appEv.ID,
				"tool_name": appEv.ToolName,
				"arguments": appEv.Arguments,
			})

		case <-ticker.C:
			// 1.5 秒热监听：比对工作区指纹，自动广播变动事件，免去用户手动刷新
			if currentFingerprint, err := getWorkspaceFingerprint(workspaceRoot); err == nil && currentFingerprint != lastFingerprint {
				lastFingerprint = currentFingerprint
				fmt.Printf("[WATCHER] 检测到工作区文件变动，向前端推送 workspace_updated: %s\n", workspaceRoot)
				s.writeSSE(w, flusher, "workspace_updated", map[string]any{"workspace_root": workspaceRoot})
			}
		}
	}
chatDone:
	// 收尾前，确保再清洗一次，覆盖由于底层库最后的结算事件带来的累计污染
	if realU := s.GetRealUsage(sessionID); realU != nil {
		lastUsage = realU
	}

	// 记录结构化调试与 I/O 日志到 bin/log/llm_io.log 中，极大方便后续开发和扫描定位
	if logFile, err := os.OpenFile("bin/log/llm_io.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		defer logFile.Close()
		var logBuilder strings.Builder
		logBuilder.WriteString("========================================================================\n")
		fmt.Fprintf(&logBuilder, "[TIMESTAMP]   %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&logBuilder, "[SESSIONID]   %s\n", sessionID)
		fmt.Fprintf(&logBuilder, "[MODELNAME]   %s\n", modelName)
		fmt.Fprintf(&logBuilder, "[USER INPUT]  %s\n", message)

		if lastUsage != nil {
			fmt.Fprintf(&logBuilder, "[TOKEN USAGE] Prompt: %d | Completion: %d | Total: %d\n",
				lastUsage.PromptTokens, lastUsage.CompletionTokens, lastUsage.TotalTokens)
		} else {
			logBuilder.WriteString("[TOKEN USAGE] Unknown\n")
		}

		if accumulatedThought.Len() > 0 {
			logBuilder.WriteString("[THINKING CHAIN]\n")
			logBuilder.WriteString(accumulatedThought.String())
			logBuilder.WriteString("\n")
		}

		if len(toolCallsMap) > 0 {
			logBuilder.WriteString("[TOOL CALLS]\n")
			for id, args := range toolCallsMap {
				name := toolNamesMap[id]
				fmt.Fprintf(&logBuilder, "  - ID: %s | Tool: %s | Args: %s\n", id, name, args)
			}
		}

		if len(accumulatedObs) > 0 {
			logBuilder.WriteString("[TOOL OBSERVATIONS]\n")
			for _, o := range accumulatedObs {
				fmt.Fprintf(&logBuilder, "  - ID: %s | Tool: %s | Result: %s\n", o["id"], o["name"], o["content"])
			}
		}

		logBuilder.WriteString("[ASSISTANT RESPONSE]\n")
		logBuilder.WriteString(accumulatedResponse.String())
		logBuilder.WriteString("\n")
		logBuilder.WriteString("========================================================================\n\n")

		logFile.WriteString(logBuilder.String())
	}

	// 记录 token 用量
	if lastUsage != nil {
		s.tokenMu.Lock()
		s.tokenIDSeq++
		now := time.Now()
		question := req.Message
		if len(question) > 200 {
			question = question[:200] + "..."
		}
		record := TokenRecord{
			ID:               s.tokenIDSeq,
			SessionID:        sessionID,
			Model:            modelName,
			PromptTokens:     lastUsage.PromptTokens,
			CompletionTokens: lastUsage.CompletionTokens,
			TotalTokens:      lastUsage.TotalTokens,
			Timestamp:        now.Unix(),
			ReadableTime:     now.Format("2006-01-02 15:04:05"),
			Question:         question,
		}
		s.tokenRecords = append(s.tokenRecords, record)
		s.saveTokenStatsLocked()
		s.tokenMu.Unlock()

		// 通过 SSE 推送 token 用量给前端
		s.writeSSE(w, flusher, "usage", map[string]any{
			"prompt_tokens":     lastUsage.PromptTokens,
			"completion_tokens": lastUsage.CompletionTokens,
			"total_tokens":      lastUsage.TotalTokens,
			"model":             modelName,
		})
	}

	s.writeSSE(w, flusher, "done", map[string]string{})
}

// handleSessions 管理会话 CRUD。
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSessions(w, r)
	case http.MethodPost:
		s.createSession(w, r)
	case http.MethodDelete:
		s.deleteSession(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	s.mu.Lock()
	defer s.mu.Unlock()

	userSessions := s.getUserSessionsLocked(userID)
	list := make([]*SessionInfo, 0, len(userSessions))
	for _, sess := range userSessions {
		list = append(list, sess)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})

	writeJSON(w, list)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
		Model string `json:"model"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	sessionID := "web-" + fmt.Sprintf("%d", time.Now().UnixNano())
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新对话"
	}
	mdl := req.Model
	if mdl == "" {
		mdl = s.defaultModel
	}

	sess := &SessionInfo{
		ID:        sessionID,
		Title:     title,
		Model:     mdl,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	userID := getUserID(r)
	s.mu.Lock()
	userSessions := s.getUserSessionsLocked(userID)
	userSessions[sessionID] = sess
	s.saveUserSessionsLocked(userID)
	s.mu.Unlock()

	writeJSON(w, sess)
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "session id is required", http.StatusBadRequest)
		return
	}

	userID := getUserID(r)
	s.mu.Lock()
	userSessions := s.getUserSessionsLocked(userID)
	delete(userSessions, sessionID)
	s.saveUserSessionsLocked(userID)
	s.mu.Unlock()

	writeJSON(w, map[string]bool{"ok": true})
}

// handleModels 返回可用模型列表。
func (s *Server) handleModels(w http.ResponseWriter, _ *http.Request) {
	type modelItem struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	}
	items := make([]modelItem, 0, len(s.models))
	for _, m := range s.models {
		items = append(items, modelItem{
			Name:        m.Name,
			DisplayName: m.DisplayName,
		})
	}
	writeJSON(w, map[string]any{
		"default": s.defaultModel,
		"models":  items,
	})
}

// handleSkills 返回快捷场景模板列表。
func (s *Server) handleSkills(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.skills)
}

// loadTokenStats 从 bin/log/token_stats.json 物理加载所有历史记录并自愈。
func (s *Server) loadTokenStats() {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	data, err := os.ReadFile("bin/log/token_stats.json")
	if err != nil {
		return
	}

	var records []TokenRecord
	if err := json.Unmarshal(data, &records); err == nil {
		s.tokenRecords = records
		for _, r := range records {
			if r.ID > s.tokenIDSeq {
				s.tokenIDSeq = r.ID
			}
		}
	}
}

// saveTokenStatsLocked 将 tokenRecords 同步写入磁盘文件，防丢。调用时需持有 s.tokenMu 锁。
func (s *Server) saveTokenStatsLocked() {
	_ = os.MkdirAll("bin/log", 0755)
	data, err := json.MarshalIndent(s.tokenRecords, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile("bin/log/token_stats.json", data, 0666)
}

// handleTokenStats 返回 token 消耗统计。
func (s *Server) handleTokenStats(w http.ResponseWriter, r *http.Request) {
	s.tokenMu.Lock()
	records := make([]TokenRecord, len(s.tokenRecords))
	copy(records, s.tokenRecords)
	s.tokenMu.Unlock()

	// 构建模型名 → 展示名映射
	displayName := make(map[string]string)
	for _, m := range s.models {
		displayName[m.Name] = m.DisplayName
	}

	// 按模型聚合
	modelMap := make(map[string]*ModelTokenStat)
	var totalPrompt, totalCompletion, totalTotal int
	for _, rec := range records {
		stat, ok := modelMap[rec.Model]
		if !ok {
			stat = &ModelTokenStat{
				Model:       rec.Model,
				DisplayName: displayName[rec.Model],
			}
			if stat.DisplayName == "" {
				stat.DisplayName = rec.Model
			}
			modelMap[rec.Model] = stat
		}
		stat.RequestCount++
		stat.PromptTokens += rec.PromptTokens
		stat.CompletionTokens += rec.CompletionTokens
		stat.TotalTokens += rec.TotalTokens
		totalPrompt += rec.PromptTokens
		totalCompletion += rec.CompletionTokens
		totalTotal += rec.TotalTokens
	}

	modelStats := make([]ModelTokenStat, 0, len(modelMap))
	for _, stat := range modelMap {
		modelStats = append(modelStats, *stat)
	}
	sort.Slice(modelStats, func(i, j int) bool {
		return modelStats[i].TotalTokens > modelStats[j].TotalTokens
	})

	// 最近 50 条记录（倒序）
	recentCount := 50
	if len(records) < recentCount {
		recentCount = len(records)
	}
	recent := make([]TokenRecord, 0, recentCount)
	for i := len(records) - 1; i >= 0 && len(recent) < recentCount; i-- {
		recent = append(recent, records[i])
	}

	writeJSON(w, map[string]any{
		"summary": map[string]int{
			"total_requests":     len(records),
			"total_prompt":        totalPrompt,
			"total_completion":    totalCompletion,
			"total_tokens":        totalTotal,
		},
		"by_model": modelStats,
		"recent":   recent,
	})
}

// handleStatic 返回一个处理静态文件的 handler，支持 SPA 路由回退。
func (s *Server) handleStatic(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查请求的文件是否存在
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(fsys, path); err != nil {
			// 文件不存在，回退到 index.html（SPA 路由）
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

// writeJSON 写出 JSON 响应。
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// writeSSE 按 SSE 协议写出一条事件并立即刷新。内置 sseMu 并发写锁，保证高危工具挂起和普通 delta 输出在协程间绝对排队安全。
func (s *Server) writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)

	// 人机审批挂起防御：由于审批即将面临最高 90 秒的协程长挂起，后续将无任何网络包。
	// 为了防止 Vite Dev Server Proxy / nginx 等本地反向代理对极小数据块进行 Buffer 缓冲（通常为 4KB），
	// 导致事件被卡在缓冲区无法实时传达前端。我们在这里写入 2048 个以 W3C SSE 规范规定的冒号开头的注释丢弃行（共 4KB），
	// 强力“挤爆”中间任何代理的 Buffer 并强制 Flush，直达浏览器。浏览器根据标准会将其作为 Comment 自动忽略丢弃。
	if event == "approval_request" {
		fmt.Fprint(w, strings.Repeat(":\n", 2048))
	}

	flusher.Flush()
}

// isDanglingJsonChunk 辅助判断流式碎片是否为仅包含大括号、逗号等 JSON 边界的冗余符号。
func isDanglingJsonChunk(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return true
	}
	for _, r := range trimmed {
		if !strings.ContainsRune("{}[]:,\"", r) {
			return false
		}
	}
	return true
}

// getUserID 提取 X-User-Id 并进行高强度路径防越权注入清洗。
func getUserID(r *http.Request) string {
	uid := r.Header.Get("X-User-Id")
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "default-user"
	}
	// 只保留安全的字符，防黑客进行任何相对路径 /../ 的提权越权攻击
	var clean []rune
	for _, r := range uid {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			clean = append(clean, r)
		}
	}
	cleaned := string(clean)
	if cleaned == "" {
		return "default-user"
	}
	return cleaned
}

// getUserSessionsLocked 获取并返回该用户的专属会话 Map。懒加载（Lazy Load）机制。调用时需持有 s.mu 写锁。
func (s *Server) getUserSessionsLocked(userID string) map[string]*SessionInfo {
	if s.sessions == nil {
		s.sessions = make(map[string]map[string]*SessionInfo)
	}
	userMap, exists := s.sessions[userID]
	if !exists {
		userMap = make(map[string]*SessionInfo)
		filename := fmt.Sprintf("bin/sessions/sessions_%s.json", userID)
		data, err := os.ReadFile(filename)
		if err == nil {
			_ = json.Unmarshal(data, &userMap)
		}
		s.sessions[userID] = userMap
	}
	return userMap
}

// saveUserSessionsLocked 将该用户的专属会话 Map 写入磁盘。调用时需持有 s.mu 写锁。
func (s *Server) saveUserSessionsLocked(userID string) {
	userMap := s.sessions[userID]
	if userMap == nil {
		return
	}
	_ = os.MkdirAll("bin/sessions", 0755)
	data, err := json.MarshalIndent(userMap, "", "  ")
	if err != nil {
		return
	}
	filename := fmt.Sprintf("bin/sessions/sessions_%s.json", userID)
	_ = os.WriteFile(filename, data, 0666)
}

// SafetyConfig 对应 config/safety.yaml 配置结构体。
type SafetyConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Actions     []string `yaml:"actions"`
	TypingSpeed int      `yaml:"typing_speed"`
}

// loadSafetyConfig 实现在运行时免重启热加载安全审批配置文件。如果读取失败，优雅地使用防呆默认值。
func loadSafetyConfig() SafetyConfig {
	config := SafetyConfig{
		Enabled:     true,
		Actions:     []string{"write_file", "edit_file", "run_command"},
		TypingSpeed: 20,
	}

	data, err := os.ReadFile("config/safety.yaml")
	if err != nil {
		return config
	}

	var loaded SafetyConfig
	if err := yaml.Unmarshal(data, &loaded); err == nil {
		return loaded
	}

	return config
}

// RequestApproval 阻塞式发起人机审批。向当前 Session 的 SSE 推送 approval_request 信号，并挂起等待用户在前端网页中进行点击选择。
func (s *Server) RequestApproval(ctx context.Context, toolName string, arguments string) bool {
	writeSSEFn, hasWriter := ctx.Value(WriteSSEFuncKey).(func(string, any))
	sessionID, hasSession := ctx.Value(SessionIDKey).(string)
	if !hasWriter || !hasSession {
		// 为了防止自动化测试在命令行跑集成测试时因没有 SSE 长连接而卡死挂起，
		// 如果上下文缺乏 SSE 推送句柄，默认放行，保持无干预状态
		return true 
	}

	// === 【免重启热加载安全防护网】===
	cfg := loadSafetyConfig()

	// 1. 全局配置过滤：如果全局安全审批被设置为 false，不弹窗，直接自动批准放行
	if !cfg.Enabled {
		fmt.Printf("[SAFETY CONFIG] 全局人机审批已禁用 (safety.yaml -> enabled: false)，自动放行特权动作: %s\n", toolName)
		return true
	}

	// 2. 行为级细粒度配置过滤：检查当前动作是否在 actions 列表中
	needed := false
	for _, action := range cfg.Actions {
		if action == toolName {
			needed = true
			break
		}
	}
	if !needed {
		fmt.Printf("[SAFETY CONFIG] 当前动作 %s 不在安全审批名单中 (safety.yaml -> actions)，自动默认放行\n", toolName)
		return true
	}
	// ===================================

	reqID := "req-" + fmt.Sprintf("%d", time.Now().UnixNano())
	req := &ApprovalRequest{
		ID:        reqID,
		SessionID: sessionID,
		ToolName:  toolName,
		Arguments: arguments,
		Status:    "pending",
		Response:  make(chan bool, 1),
	}

	s.approvalsMu.Lock()
	if s.approvals == nil {
		s.approvals = make(map[string]*ApprovalRequest)
	}
	s.approvals[reqID] = req
	s.approvalsMu.Unlock()

	// 1. 发送 SSE 事件告诉前端：快弹出审批框阻断！
	if flushChan, ok := ctx.Value(FlushChanKey).(chan *ApprovalEvent); ok {
		fmt.Printf("[DEBUG APPROVAL] 正在通过主协程刷新通道推送审批事件: ID=%s, Tool=%s\n", reqID, toolName)
		flushChan <- &ApprovalEvent{
			ID:        reqID,
			ToolName:  toolName,
			Arguments: arguments,
		}
	} else {
		fmt.Printf("[DEBUG APPROVAL] [警告] 未在上下文中找到主协程 flushChan，降级为并发直接写入 SSE！\n")
		writeSSEFn("approval_request", map[string]any{
			"id":        reqID,
			"tool_name": toolName,
			"arguments": arguments,
		})
	}

	// 2. 阻塞式读取 Response 通道，直到用户在前端网页点击响应或 90 秒安全超时熔断！
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fmt.Printf("[DEBUG APPROVAL] 协程开始阻塞挂起等待审批响应...\n")
	select {
	case approved := <-req.Response:
		fmt.Printf("[DEBUG APPROVAL] 协程被 Response 唤醒，审批决策结果: %t\n", approved)
		return approved
	case <-timeoutCtx.Done():
		// 如果 90 秒内用户没有做出选择（可能人不在电脑前），系统安全强熔断，默认硬拒绝！
		fmt.Printf("[DEBUG APPROVAL] 90秒超时熔断触发，默认硬拒绝！\n")
		s.approvalsMu.Lock()
		delete(s.approvals, reqID)
		s.approvalsMu.Unlock()
		return false
	}
}

// handleApprovalRespond 接收用户在前端网页点击 “允许” 或者是 “拒绝” 的决策响应 API 接口。
func (s *Server) handleApprovalRespond(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID      string `json:"id"`
		Approve bool   `json:"approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("[DEBUG APPROVAL RESPOND] 收到前端审批接口响应: ID=%s, Approve=%t\n", req.ID, req.Approve)

	s.approvalsMu.Lock()
	appReq, exists := s.approvals[req.ID]
	if exists {
		// 消费掉该审批请求并安全移除
		delete(s.approvals, req.ID)
	}
	s.approvalsMu.Unlock()

	if !exists {
		http.Error(w, "approval request not found or expired", http.StatusNotFound)
		return
	}

	// 更改状态
	appReq.Status = "approved"
	if !req.Approve {
		appReq.Status = "rejected"
	}

	// 瞬间向 Go 挂起管道送入信号，秒级激活并唤醒原本阻塞的物理工具执行协程！
	appReq.Response <- req.Approve

	writeJSON(w, map[string]bool{"ok": true})
}

// UserSettings 对应用户的自定义偏好设置（如 WorkspaceRoot 等）。
type UserSettings struct {
	WorkspaceRoot string `json:"workspace_root"`
	TypingSpeed   int    `json:"typing_speed"`
}

// FileNode 表示工作目录下的树形文件结构。
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"` // 相对路径，便于前端交互
	IsDir    bool        `json:"is_dir"`
	Children []*FileNode `json:"children,omitempty"`
}

// handleSettings 管理用户指定的运行根目录（获取和设置）。支持物理文件夹有效性校验和防丢落盘。
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	filename := fmt.Sprintf("bin/sessions/settings_%s.json", userID)

	if r.Method == http.MethodGet {
		wd, _ := os.Getwd()
		cfg := loadSafetyConfig()
		settings := UserSettings{
			WorkspaceRoot: wd,
			TypingSpeed:   cfg.TypingSpeed,
		}

		data, err := os.ReadFile(filename)
		if err == nil {
			var loaded UserSettings
			if err := json.Unmarshal(data, &loaded); err == nil && loaded.WorkspaceRoot != "" {
				settings.WorkspaceRoot = loaded.WorkspaceRoot
			}
		}

		writeJSON(w, settings)
		return
	}

	if r.Method == http.MethodPost {
		var req UserSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		req.WorkspaceRoot = strings.TrimSpace(req.WorkspaceRoot)
		if req.WorkspaceRoot == "" {
			http.Error(w, "workspace_root is required", http.StatusBadRequest)
			return
		}

		// 验证路径的物理存在性，防拼写错误
		if info, err := os.Stat(req.WorkspaceRoot); err != nil || !info.IsDir() {
			http.Error(w, "该文件夹路径在本地不存在，请核对输入！", http.StatusBadRequest)
			return
		}

		_ = os.MkdirAll("bin/sessions", 0755)
		data, err := json.MarshalIndent(req, "", "  ")
		if err == nil {
			_ = os.WriteFile(filename, data, 0666)
		}

		writeJSON(w, map[string]any{"ok": true, "workspace_root": req.WorkspaceRoot})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// getUserWorkspaceRoot 获取该用户的指定运行根目录，若不存在则回退至 Go 进程默认的 wd 工作区。
func (s *Server) getUserWorkspaceRoot(userID string) string {
	filename := fmt.Sprintf("bin/sessions/settings_%s.json", userID)
	data, err := os.ReadFile(filename)
	if err == nil {
		var loaded UserSettings
		if err := json.Unmarshal(data, &loaded); err == nil && loaded.WorkspaceRoot != "" {
			return loaded.WorkspaceRoot
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// handleWorkspaceFiles 极速扫描用户的 Workspace 并组装树形数据返回给前端。
func (s *Server) handleWorkspaceFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	rootPath := s.getUserWorkspaceRoot(userID)

	tree, err := buildFileTree(rootPath, rootPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"workspace_root": rootPath,
		"files":          tree,
	})
}

// buildFileTree 递归遍历物理路径并组装 FileTree，屏蔽 node_modules/git 等巨无霸地带提速百倍，自带 VS Code 风格（文件夹置顶）的高级排序。
func buildFileTree(root, current string) ([]*FileNode, error) {
	entries, err := os.ReadDir(current)
	if err != nil {
		return nil, err
	}

	nodes := make([]*FileNode, 0)
	for _, entry := range entries {
		name := entry.Name()
		if name == "node_modules" || name == ".git" || name == "vendor" || name == ".codebuddy" || name == "bin" {
			continue
		}

		fullPath := filepath.Join(current, name)
		relPath, err := filepath.Rel(root, fullPath)
		if err != nil {
			relPath = name
		}
		relPath = filepath.ToSlash(relPath)

		node := &FileNode{
			Name:  name,
			Path:  relPath,
			IsDir: entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := buildFileTree(root, fullPath)
			if err == nil {
				node.Children = children
			}
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir && !nodes[j].IsDir {
			return true
		}
		if !nodes[i].IsDir && nodes[j].IsDir {
			return false
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})

	return nodes, nil
}

// handleSelectWorkspaceDialog 拉起操作系统（Windows PowerShell）原生文件夹浏览选择弹窗，极大改善输入路径体验。
func (s *Server) handleSelectWorkspaceDialog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 采用微软 100% 原生 COM 免配置、免转义的 Shell.Application 现代化大弹窗方案：
	// 并在调用前利用 Windows 核心 API GetForegroundWindow() 动态锚定当前前台拥有焦点的浏览器（Chrome/Edge）的窗口句柄 (HWND)。
	// 将其作为父窗口句柄传入，在 Windows 的窗口层级 Z-Order 中，子 Dialog 100% 会被强行绝对置顶在浏览器最前端，彻底解决隐藏在背后的瑕疵。
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
		"$sig = '[DllImport(\"user32.dll\")] public static extern IntPtr GetForegroundWindow();'; $type = Add-Type -MemberDefinition $sig -Name 'Win32Win' -Namespace 'Win32' -PassThru; $hwnd = $type::GetForegroundWindow(); $app = New-Object -ComObject Shell.Application; $f = $app.BrowseForFolder($hwnd, '请选择 trpc_agent 智能体的运行根目录', 80, 17); if ($f) { $f.Self.Path }")
	
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[SELECT DIALOG ERR] 执行 PowerShell 弹窗命令失败: %v, 输出日志: %s\n", err, string(out))
		http.Error(w, fmt.Sprintf("调用系统目录选择弹窗失败: %v, 详情: %s", err, string(out)), http.StatusInternalServerError)
		return
	}

	path := strings.TrimSpace(string(out))
	if path == "" {
		// 用户点击了取消或者没选，优雅返回 canceled
		writeJSON(w, map[string]any{"ok": true, "canceled": true})
		return
	}

	// 转换下划线斜杠，确保路径格式在各端呈现优美一致
	path = filepath.Clean(path)

	writeJSON(w, map[string]any{
		"ok":       true,
		"canceled": false,
		"path":     path,
	})
}

// getWorkspaceFingerprint 极其高效地对工作区生成一个唯一的、极轻量的状态指纹（总文件数 + 最后一个修改时间 + 大小和），从而不需要消耗 CPU 进行深层内容 Hash
func getWorkspaceFingerprint(root string) (string, error) {
	var totalFiles int
	var maxModTime int64
	var totalSize int64

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 容错，有些文件可能无法读取，直接略过
		}
		name := info.Name()
		// 屏蔽巨无霸排除区，提速千倍，同时防止循环递归
		if name == "node_modules" || name == ".git" || name == "vendor" || name == ".codebuddy" || name == "bin" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			totalFiles++
			totalSize += info.Size()
			if info.ModTime().Unix() > maxModTime {
				maxModTime = info.ModTime().Unix()
			}
		} else {
			// 文件夹的修改也算
			if info.ModTime().Unix() > maxModTime {
				maxModTime = info.ModTime().Unix()
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d-%d-%d", totalFiles, maxModTime, totalSize), nil
}

// FilePreviewResponse 对应文件内容预览信息载荷。
type FilePreviewResponse struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ModTime   int64  `json:"mod_time"`
	IsDir     bool   `json:"is_dir"`
	IsBinary  bool   `json:"is_binary"`
	Extension string `json:"extension"`
	Content   string `json:"content"`
	IsTrunc   bool   `json:"is_truncated"` // 标志是否发生了大文件截断保护
}

// handleWorkspaceFileRaw 用于流式直接返回原始文件（对图片、PDF 预览尤为关键，支持 Range 断点续传）。
func (s *Server) handleWorkspaceFileRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	rootPath := s.getUserWorkspaceRoot(userID)
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	absPath := filepath.Clean(filepath.Join(rootPath, relPath))
	if !strings.HasPrefix(absPath, rootPath) {
		http.Error(w, "安全沙箱拦截：拒绝非法目录穿透", http.StatusForbidden)
		return
	}

	// 100% 官方原生 ServeFile 机制，自动完成 Etag / MIME Type 匹配、极速输出流
	http.ServeFile(w, r, absPath)
}

// handleWorkspaceFilePreview 智能判别二进制，并读取部分分片字符返回，实现大文件内存与渲染无感防卡自愈。
func (s *Server) handleWorkspaceFilePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	rootPath := s.getUserWorkspaceRoot(userID)
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	absPath := filepath.Clean(filepath.Join(rootPath, relPath))
	if !strings.HasPrefix(absPath, rootPath) {
		http.Error(w, "安全沙箱拦截：拒绝非法目录穿透", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("文件在本地物理不存在: %v", err), http.StatusNotFound)
		return
	}

	resp := FilePreviewResponse{
		Name:      info.Name(),
		Size:      info.Size(),
		ModTime:   info.ModTime().Unix(),
		IsDir:     info.IsDir(),
		Extension: strings.ToLower(filepath.Ext(info.Name())),
	}

	if info.IsDir() {
		writeJSON(w, resp)
		return
	}

	// 1. 读取前 1024 字节，智能检测是否是二进制格式（含有 0x00 字节）
	file, err := os.Open(absPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件流失败: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	buf := make([]byte, 1024)
	n, _ := file.Read(buf)
	isBinary := false
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			isBinary = true
			break
		}
	}
	resp.IsBinary = isBinary

	// 2. 如果不是二进制文本，载入文本并开启 150KB 的大文件防御屏障
	if !isBinary {
		const maxPreviewBytes = 150 * 1024 // 150KB 大文件物理防火墙
		if info.Size() > maxPreviewBytes {
			resp.IsTrunc = true
			contentBuf := make([]byte, maxPreviewBytes)
			_, _ = file.ReadAt(contentBuf, 0)
			resp.Content = string(contentBuf) + "\n\n... [⚙️ 大文件自愈防护保护] 侦测到该文件体积巨大，当前仅按需预览前 150KB。如需完整检索/编辑，请唤醒 AI 编写脚本进行批量物理修改！..."
		} else {
			contentBytes, err := os.ReadFile(absPath)
			if err == nil {
				resp.Content = string(contentBytes)
			}
		}
	}

	writeJSON(w, resp)
}

// SetRealUsage 记录某次物理网络请求返回的最真实的 Usage，清洗全局累计用量污染，确保 100% 精准单次账单
func (s *Server) SetRealUsage(sessionID string, u *model.Usage) {
	s.realUsageMu.Lock()
	defer s.realUsageMu.Unlock()
	if s.realUsage == nil {
		s.realUsage = make(map[string]*model.Usage)
	}
	s.realUsage[sessionID] = u
}

// GetRealUsage 获取该会话最新一次最干净、最精确的单次物理用量
func (s *Server) GetRealUsage(sessionID string) *model.Usage {
	s.realUsageMu.Lock()
	defer s.realUsageMu.Unlock()
	if s.realUsage == nil {
		return nil
	}
	return s.realUsage[sessionID]
}



