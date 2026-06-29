//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TokenRecord 与主程序中的结构保持一致
type TokenRecord struct {
	ID             int64  `json:"id"`
	SessionID      string `json:"session_id"`
	Model          string `json:"model"`
	PromptTokens   int    `json:"prompt_tokens"`
	CompletionTokens int  `json:"completion_tokens"`
	TotalTokens    int    `json:"total_tokens"`
	Timestamp      int64  `json:"timestamp"`
	ReadableTime   string `json:"readable_time"`
	Question       string `json:"question,omitempty"`
}

func main() {
	// 读取现有文件
	data, err := os.ReadFile("bin/log/token_stats.json")
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		return
	}

	// 解析 JSON
	var records []TokenRecord
	if err := json.Unmarshal(data, &records); err != nil {
		fmt.Printf("解析 JSON 失败: %v\n", err)
		return
	}

	// 迁移：为每条记录添加可读时间
	for i := range records {
		if records[i].ReadableTime == "" && records[i].Timestamp > 0 {
			t := time.Unix(records[i].Timestamp, 0)
			records[i].ReadableTime = t.Format("2006-01-02 15:04:05")
		}
	}

	// 写回文件
	output, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		fmt.Printf("序列化 JSON 失败: %v\n", err)
		return
	}

	if err := os.WriteFile("bin/log/token_stats.json", output, 0666); err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		return
	}

	fmt.Printf("成功迁移 %d 条记录，已添加可读时间\n", len(records))
}
