package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   trpc-agent-go 自动化全流式 ReAct 引擎测试脚本")
	fmt.Println("==================================================")

	// 1. 创建 Session
	sessionPayload := map[string]string{
		"title": "自动化测试对话",
		"model": "Qwen/Qwen2.5-72B-Instruct",
	}
	sessionBytes, _ := json.Marshal(sessionPayload)
	resp, err := http.Post("http://localhost:8080/api/sessions", "application/json", bytes.NewBuffer(sessionBytes))
	if err != nil {
		fmt.Printf("[🔴 失败] 无法连接到本地服务: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var session struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&session)
	if session.ID == "" {
		fmt.Println("[🔴 失败] 创建 Session 失败，未能获取 ID")
		os.Exit(1)
	}
	fmt.Printf("[🟢 成功] 成功创建测试会话 ID: %s\n", session.ID)

	// 2. 发起流式 Chat 请求
	chatPayload := map[string]any{
		"message":    "算一下：125 * 38 加 452 减 12 是多少？",
		"session_id": session.ID,
		"model":      "Qwen/Qwen2.5-72B-Instruct",
		"skill_id":   nil,
	}
	chatBytes, _ := json.Marshal(chatPayload)
	chatResp, err := http.Post("http://localhost:8080/api/chat", "application/json", bytes.NewBuffer(chatBytes))
	if err != nil {
		fmt.Printf("[🔴 失败] 无法调用 Chat API: %v\n", err)
		os.Exit(1)
	}
	defer chatResp.Body.Close()

	fmt.Println("[⏳ 等待] 正在接收并解析 SSE 动态决策流...")
	buffer := make([]byte, 1024)
	var fullResponse strings.Builder
	var eventsReceived []string

	for {
		n, err := chatResp.Body.Read(buffer)
		if n > 0 {
			chunk := string(buffer[:n])
			fullResponse.WriteString(chunk)
			for _, line := range strings.Split(chunk, "\n") {
				if strings.HasPrefix(line, "event:") {
					eventsReceived = append(eventsReceived, line)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("[🔴 错误] 读取流异常: %v\n", err)
			break
		}
	}

	fmt.Println("\n[🔍 验证] 正在对事件流和日志数据结构进行多维度校验...")

	// 验证事件流是否包含 thought, tool_call, observation, delta, done
	hasThought := false
	hasToolCall := false
	hasObservation := false
	hasDelta := false
	hasDone := false

	for _, ev := range eventsReceived {
		if strings.Contains(ev, "thought") { hasThought = true }
		if strings.Contains(ev, "tool_call") { hasToolCall = true }
		if strings.Contains(ev, "observation") { hasObservation = true }
		if strings.Contains(ev, "delta") { hasDelta = true }
		if strings.Contains(ev, "done") { hasDone = true }
	}

	fmt.Printf("  ├─ 是否收到 thought (推理思维链) 信号: %t\n", hasThought)
	fmt.Printf("  ├─ 是否收到 tool_call (工具调用指令) 信号: %t\n", hasToolCall)
	fmt.Printf("  ├─ 是否收到 observation (工具返回结果) 信号: %t\n", hasObservation)
	fmt.Printf("  ├─ 是否收到 delta (文本输出) 信号: %t\n", hasDelta)
	fmt.Printf("  └─ 是否收到 done (会话正常结束) 信号: %t\n", hasDone)

	// 3. 读取并校验 bin/llm_io.log
	time.Sleep(500 * time.Millisecond) // 等待日志异步文件写入
	logData, err := os.ReadFile("bin/llm_io.log")
	if err != nil {
		fmt.Printf("[🔴 失败] 无法读取 bin/llm_io.log: %v\n", err)
		os.Exit(1)
	}

	logStr := string(logData)
	if !strings.Contains(logStr, session.ID) {
		fmt.Println("[🔴 失败] 日志中缺失当前测试会话 ID 信息！")
		os.Exit(1)
	}

	if !strings.Contains(logStr, "[TOOL OBSERVATIONS]") {
		fmt.Println("[🔴 失败] 日志中缺失物理工具观测记录栏 [TOOL OBSERVATIONS]！")
		os.Exit(1)
	}

	if !strings.Contains(logStr, "5190") {
		fmt.Println("[🔴 失败] 计算输出存在异常，未能检索到核心计算结果 5190！")
		os.Exit(1)
	}

	fmt.Println("\n==================================================")
	fmt.Println(" [🎉 自动化测试通过] 恭喜老公！流式响应无瑕疵，日志完美规整！")
	fmt.Println("==================================================")
}
