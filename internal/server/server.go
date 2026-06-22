// Package server 提供基于 HTTP/SSE 的 Web 聊天服务，
// 支持多模型切换、会话管理、技能模板等功能。
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"
)

const webUserID = "web-user"

//go:embed static
var staticFS embed.FS

// ModelConfig 定义一个可用于聊天的模型配置。
type ModelConfig struct {
	Name        string        // 唯一标识，如 "deepseek-ai/DeepSeek-V3"
	DisplayName string        // 展示名，如 "DeepSeek V3"
	Runner      runner.Runner
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
	}

	s.mux.HandleFunc("/", s.handleStatic(sub))
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/models", s.handleModels)
	s.mux.HandleFunc("/api/skills", s.handleSkills)
	s.mux.HandleFunc("/api/token-stats", s.handleTokenStats)
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

	events, err := rnr.Run(r.Context(), userID, sessionID, model.NewUserMessage(timeContext))
	if err != nil {
		writeSSE(w, flusher, "error", map[string]string{"message": err.Error()})
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

	for ev := range events {
		if ev.Error != nil {
			writeSSE(w, flusher, "error", map[string]string{"message": ev.Error.Message})
			continue
		}
		if ev.Usage != nil {
			lastUsage = ev.Usage
		}

		// 1. 推送大模型流式推理/深度思考链 (Reasoning/Thinking)
		if len(ev.Choices) > 0 {
			reasoning := ev.Choices[0].Delta.ReasoningContent
			if reasoning != "" {
				accumulatedThought.WriteString(reasoning)
				writeSSE(w, flusher, "thought", map[string]string{"content": reasoning})
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

					writeSSE(w, flusher, "tool_call", map[string]any{
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
				content = strings.TrimSpace(content)

				xmlBuffer.WriteString(content)
				currentText := xmlBuffer.String()

				// 如果尚未进入 XML 丢弃块
				if !inXmlBlock {
					if idx := strings.Index(currentText, "<tool_call>"); idx != -1 {
						// 将 <tool_call> 标签之前的所有合法人类对话文本正常推送
						before := currentText[:idx]
						if before != "" && !isDanglingJsonChunk(before) {
							accumulatedResponse.WriteString(before)
							writeSSE(w, flusher, "delta", map[string]string{"content": before})
						}
						// 状态转换为已进入 XML 丢弃块
						inXmlBlock = true
						xmlBuffer.Reset()
						xmlBuffer.WriteString(currentText[idx:])
					} else {
						// 确认没有工具标签开始符号，且不含有任何大括号等泄漏前缀时，立即安全下发当前内容
						trimmed := strings.TrimSpace(currentText)
						if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "<") {
							accumulatedResponse.WriteString(currentText)
							writeSSE(w, flusher, "delta", map[string]string{"content": currentText})
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
							if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "<") && !isDanglingJsonChunk(currentText) {
								accumulatedResponse.WriteString(currentText)
								writeSSE(w, flusher, "delta", map[string]string{"content": currentText})
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
					writeSSE(w, flusher, "observation", map[string]any{
						"id":      choice.Message.ToolID,
						"name":    choice.Message.ToolName,
						"content": choice.Message.Content,
					})
				}
			}
		}
	}

	// 记录结构化调试与 I/O 日志到 bin/llm_io.log 中，极大方便后续开发和扫描定位
	if logFile, err := os.OpenFile("bin/llm_io.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
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
		record := TokenRecord{
			ID:               s.tokenIDSeq,
			SessionID:        sessionID,
			Model:            modelName,
			PromptTokens:     lastUsage.PromptTokens,
			CompletionTokens: lastUsage.CompletionTokens,
			TotalTokens:      lastUsage.TotalTokens,
			Timestamp:        time.Now().Unix(),
		}
		s.tokenRecords = append(s.tokenRecords, record)
		s.tokenMu.Unlock()

		// 通过 SSE 推送 token 用量给前端
		writeSSE(w, flusher, "usage", map[string]any{
			"prompt_tokens":     lastUsage.PromptTokens,
			"completion_tokens": lastUsage.CompletionTokens,
			"total_tokens":      lastUsage.TotalTokens,
			"model":             modelName,
		})
	}

	writeSSE(w, flusher, "done", map[string]string{})
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

// writeSSE 按 SSE 协议写出一条事件并立即刷新。
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
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
