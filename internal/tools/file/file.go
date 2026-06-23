// Package file 提供文件系统操作工具，使 Agent 能够读取、写入文件并浏览目录。
package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// ListDirInput 目录遍历工具输入。
type ListDirInput struct {
	Path string `json:"path" description:"要浏览的目录路径（相对路径，如 . 或 ./web，严禁超出工作区范围）"`
}

// 跨包 context key 映射
const (
	ApprovalManagerKey = "approval_manager"
)

// 隐式接口定义，避免包循环导入依赖
type approvalManager interface {
	RequestApproval(ctx context.Context, toolName string, arguments string) bool
}

// FileInfo 结构化的文件和目录信息。
type FileInfo struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// ListDirOutput 目录遍历工具输出。
type ListDirOutput struct {
	Path  string     `json:"path" description:"当前目录绝对路径"`
	Files []FileInfo `json:"files" description:"目录下的文件和文件夹列表"`
}

// EditFileInput 文件局部修改工具输入。
type EditFileInput struct {
	Path   string `json:"path" description:"要修改的目标文件相对路径（如 internal/agent/agent.go）"`
	OldStr string `json:"old_str" description:"要被替换掉的唯一、精细的旧代码段落（必须包含完整的换行与缩进，且在文件中唯一存在）"`
	NewStr string `json:"new_str" description:"替换成的新代码内容"`
}

// EditFileOutput 文件局部修改工具输出。
type EditFileOutput struct {
	Success bool   `json:"success" description:"是否精准替换成功"`
	Message string `json:"message" description:"状态与排错提示消息"`
}

// GlobFilesInput 文件递归检索输入。
type GlobFilesInput struct {
	Pattern string `json:"pattern" description:"通配符检索模式或关键字（如 '*.go'、'TheSidebar.vue' 或 'main'），支持模糊匹配"`
}

// GlobFilesOutput 文件递归检索输出。
type GlobFilesOutput struct {
	Pattern string   `json:"pattern"`
	Matches []string `json:"matches" description:"所有匹配成功的文件相对工作区路径列表"`
	Count   int      `json:"count" description:"匹配到的文件总数"`
}

// GrepSearchInput 全局代码正文检索输入。
type GrepSearchInput struct {
	Pattern string `json:"pattern" description:"要检索匹配的关键字（例如 'EnsureSafePath' 或 'New'）"`
	Ext     string `json:"ext" description:"可选，限定只在特定后缀名的文件中检索（例如 '.go' 或 '.vue'），留空则在所有可读文本文件中检索"`
}

// GrepMatch 检索匹配到的行详情。
type GrepMatch struct {
	Path    string `json:"path" description:"匹配到的文件相对路径"`
	Line    int    `json:"line" description:"匹配的行号（从 1 开始）"`
	Content string `json:"content" description:"匹配到的该行代码内容（已自动 Trim 左右多余空白）"`
}

// GrepSearchOutput 全局代码正文检索输出。
type GrepSearchOutput struct {
	Pattern string      `json:"pattern"`
	Matches []GrepMatch `json:"matches" description:"匹配到的代码行列表，限制最多返回前 50 条，防止返回过多"`
	Count   int         `json:"count" description:"全局实际匹配到的代码行总数"`
}

// ReadFileInput 文件读取工具输入。
type ReadFileInput struct {
	Path string `json:"path" description:"要读取的文件路径（相对路径，如 ./internal/agent/agent.go）"`
}

// ReadFileOutput 文件读取工具输出。
type ReadFileOutput struct {
	Content string `json:"content" description:"文件全部内容"`
}

// WriteFileInput 文件写入工具输入。
type WriteFileInput struct {
	Path    string `json:"path" description:"要写入的目标文件路径（相对路径）"`
	Content string `json:"content" description:"要写入的文件内容"`
}

// WriteFileOutput 文件写入工具输出。
type WriteFileOutput struct {
	Success bool   `json:"success" description:"是否写入成功"`
	Message string `json:"message" description:"状态消息"`
}

// ensureSafePath 安全检测：确保路径在当前工作区内，禁止穿透（如 ../..）访问系统根目录。
func ensureSafePath(inputPath string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("无法获取当前工作目录: %v", err)
	}

	// 默认为当前工作目录
	if inputPath == "" || inputPath == "." {
		return wd, nil
	}

	// 解析为绝对路径
	var targetPath string
	if filepath.IsAbs(inputPath) {
		targetPath = filepath.Clean(inputPath)
	} else {
		targetPath = filepath.Clean(filepath.Join(wd, inputPath))
	}

	// 必须以当前工作目录（工作区）为前缀
	if !strings.HasPrefix(targetPath, wd) {
		return "", fmt.Errorf("安全沙箱拦截：路径 '%s' 超出当前工作区范围", inputPath)
	}

	return targetPath, nil
}

// ListDirectory 列出指定目录下的文件和文件夹。
func ListDirectory(ctx context.Context, input ListDirInput) (ListDirOutput, error) {
	safePath, err := ensureSafePath(input.Path)
	if err != nil {
		return ListDirOutput{}, err
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		return ListDirOutput{}, fmt.Errorf("读取目录失败: %v", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		files = append(files, FileInfo{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	return ListDirOutput{
		Path:  safePath,
		Files: files,
	}, nil
}

// ReadFileContent 读取指定文件的全部文本内容。
func ReadFileContent(ctx context.Context, input ReadFileInput) (ReadFileOutput, error) {
	safePath, err := ensureSafePath(input.Path)
	if err != nil {
		return ReadFileOutput{}, err
	}

	contentBytes, err := os.ReadFile(safePath)
	if err != nil {
		return ReadFileOutput{}, fmt.Errorf("读取文件失败: %v", err)
	}

	return ReadFileOutput{
		Content: string(contentBytes),
	}, nil
}

// WriteFileContent 写入文本内容到指定文件。
func WriteFileContent(ctx context.Context, input WriteFileInput) (WriteFileOutput, error) {
	safePath, err := ensureSafePath(input.Path)
	if err != nil {
		return WriteFileOutput{Success: false, Message: err.Error()}, nil
	}

	// 人机协同安全网拦截：如果上下文注入了审批管理器，进行阻塞式审批挂起
	if mgr, ok := ctx.Value(ApprovalManagerKey).(approvalManager); ok {
		approved := mgr.RequestApproval(ctx, "write_file", fmt.Sprintf("写入文件: %s", input.Path))
		if !approved {
			return WriteFileOutput{
				Success: false,
				Message: "安全防护拦截：用户在前端弹窗中点击了[拒绝]，此高危写盘指令被强行硬性拦截终止，未执行任何物理写盘！",
			}, nil
		}
	}

	// 确保父目录存在
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return WriteFileOutput{Success: false, Message: fmt.Sprintf("创建文件夹失败: %v", err)}, nil
	}

	err = os.WriteFile(safePath, []byte(input.Content), 0644)
	if err != nil {
		return WriteFileOutput{Success: false, Message: fmt.Sprintf("写入文件失败: %v", err)}, nil
	}

	return WriteFileOutput{
		Success: true,
		Message: fmt.Sprintf("文件已成功写入到: %s", input.Path),
	}, nil
}

// EditFileContent 精准局部替换修改文件中的代码段。
func EditFileContent(ctx context.Context, input EditFileInput) (EditFileOutput, error) {
	safePath, err := ensureSafePath(input.Path)
	if err != nil {
		return EditFileOutput{Success: false, Message: err.Error()}, nil
	}

	// 人机协同安全网拦截：如果上下文注入了审批管理器，进行阻塞式审批挂起
	if mgr, ok := ctx.Value(ApprovalManagerKey).(approvalManager); ok {
		approved := mgr.RequestApproval(ctx, "edit_file", fmt.Sprintf("精密修改文件: %s", input.Path))
		if !approved {
			return EditFileOutput{
				Success: false,
				Message: "安全防护拦截：用户在前端弹窗中点击了[拒绝]，此高危修改文件指令被强行硬性拦截终止，未修改任何物理代码文件！",
			}, nil
		}
	}

	contentBytes, err := os.ReadFile(safePath)
	if err != nil {
		return EditFileOutput{Success: false, Message: fmt.Sprintf("读取文件失败: %v", err)}, nil
	}
	content := string(contentBytes)

	oldStr := input.OldStr
	if oldStr == "" {
		return EditFileOutput{Success: false, Message: "错误：要替换的旧代码段 'old_str' 不能为空"}, nil
	}

	// 验证 old_str 在文件中的唯一性，防止大模型误伤或模糊替换
	count := strings.Count(content, oldStr)
	if count == 0 {
		return EditFileOutput{
			Success: false,
			Message: "修改失败：未在目标文件中找到匹配的旧代码段 'old_str'。请先用 read_file 精确读取文件，确保换行、空格和缩进与原代码一模一样再重试！",
		}, nil
	}
	if count > 1 {
		return EditFileOutput{
			Success: false,
			Message: fmt.Sprintf("修改失败：在文件中找到了 %d 处完全相同的旧代码段。为了保障安全，请提供更多前后的上下文字符，使 old_str 在该文件中唯一！", count),
		}, nil
	}

	// 执行精准 1 次替换
	newContent := strings.Replace(content, oldStr, input.NewStr, 1)
	err = os.WriteFile(safePath, []byte(newContent), 0644)
	if err != nil {
		return EditFileOutput{Success: false, Message: fmt.Sprintf("写入修改内容失败: %v", err)}, nil
	}

	return EditFileOutput{
		Success: true,
		Message: fmt.Sprintf("文件 '%s' 已成功精准修改并替换 1 处！", input.Path),
	}, nil
}

// GlobFiles 在工作区内，递归进行极速文件查找过滤。
func GlobFiles(ctx context.Context, input GlobFilesInput) (GlobFilesOutput, error) {
	wd, err := os.Getwd()
	if err != nil {
		return GlobFilesOutput{}, err
	}

	pattern := strings.ToLower(input.Pattern)
	var matches []string

	err = filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略局部无法访问的文件，确保搜索不中断
		}
		if info.IsDir() {
			// 核心提速防线：跳过超重的 node_modules、git 缓存及 vendor 构建目录，效率提升百倍！
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == ".codebuddy" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}

		// 转为相对路径，便于大模型跨平台阅读
		rel, err := filepath.Rel(wd, path)
		if err != nil {
			return nil
		}

		relLower := strings.ToLower(filepath.ToSlash(rel))

		// 智能自适应通配符过滤
		if strings.HasPrefix(pattern, "*.") {
			ext := pattern[1:] // 如 ".go"
			if strings.HasSuffix(relLower, ext) {
				matches = append(matches, rel)
			}
		} else if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, relLower)
			if matched || strings.Contains(relLower, strings.ReplaceAll(pattern, "*", "")) {
				matches = append(matches, rel)
			}
		} else {
			// 模糊路径检索
			if strings.Contains(relLower, pattern) {
				matches = append(matches, rel)
			}
		}

		return nil
	})

	return GlobFilesOutput{
		Pattern: input.Pattern,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

// GrepSearch 扫描工作区所有文本文件，对每行文本进行关键字检索。
func GrepSearch(ctx context.Context, input GrepSearchInput) (GrepSearchOutput, error) {
	wd, err := os.Getwd()
	if err != nil {
		return GrepSearchOutput{}, err
	}

	pattern := input.Pattern
	if pattern == "" {
		return GrepSearchOutput{}, fmt.Errorf("检索关键字 'pattern' 不能为空")
	}

	targetExt := strings.ToLower(strings.TrimSpace(input.Ext))
	if targetExt != "" && !strings.HasPrefix(targetExt, ".") {
		targetExt = "." + targetExt
	}

	var matches []GrepMatch
	totalCount := 0

	err = filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// 跳过垃圾冗余目录
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == ".codebuddy" || name == "bin" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if targetExt != "" && ext != targetExt {
			return nil
		}

		// 忽略常见的非文本、大型二进制文件或打包产物
		if ext == ".exe" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".zip" || ext == ".tar" || ext == ".gz" || ext == ".log" || ext == ".db" || ext == ".sqlite" {
			return nil
		}

		rel, err := filepath.Rel(wd, path)
		if err != nil {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(contentBytes), "\n")
		for i, line := range lines {
			if strings.Contains(line, pattern) {
				totalCount++
				// 安全防护：至多收集前 50 条匹配，其余只做计数，避免大模型被暴增数据撑爆
				if len(matches) < 50 {
					matches = append(matches, GrepMatch{
						Path:    rel,
						Line:    i + 1,
						Content: strings.TrimSpace(line),
					})
				}
			}
		}

		return nil
	})

	return GrepSearchOutput{
		Pattern: pattern,
		Matches: matches,
		Count:   totalCount,
	}, nil
}

// NewListDirTool 创建目录浏览工具。
func NewListDirTool() tool.Tool {
	return function.NewFunctionTool(
		ListDirectory,
		function.WithName("list_directory"),
		function.WithDescription("列出指定目录路径下的所有文件和文件夹列表"),
	)
}

// NewReadFileTool 创建文件读取工具。
func NewReadFileTool() tool.Tool {
	return function.NewFunctionTool(
		ReadFileContent,
		function.WithName("read_file"),
		function.WithDescription("读取并返回指定文件路径下的全部文本内容"),
	)
}

// NewWriteFileTool 创建文件写入工具。
func NewWriteFileTool() tool.Tool {
	return function.NewFunctionTool(
		WriteFileContent,
		function.WithName("write_file"),
		function.WithDescription("将指定的文本内容写入或覆盖到目标文件路径中"),
	)
}

// NewEditFileTool 创建文件精细编辑修改工具。
func NewEditFileTool() tool.Tool {
	return function.NewFunctionTool(
		EditFileContent,
		function.WithName("edit_file"),
		function.WithDescription("精准、安全地替换文件中唯一匹配的旧代码段（old_str）为新代码段（new_str）"),
	)
}

// NewGlobFilesTool 创建全局通配符文件检索工具。
func NewGlobFilesTool() tool.Tool {
	return function.NewFunctionTool(
		GlobFiles,
		function.WithName("glob_files"),
		function.WithDescription("在工作区内按文件名通配符模式或关键字（如 '*.go' 或 'settings'）递归查找并返回相对路径列表"),
	)
}

// NewGrepSearchTool 创建全局代码文本全文检索匹配工具。
func NewGrepSearchTool() tool.Tool {
	return function.NewFunctionTool(
		GrepSearch,
		function.WithName("grep_search"),
		function.WithDescription("在工作区内递归搜索匹配指定 pattern 关键字的每一行文本，并返回包含文件名、行号及内容的匹配详情（限制最多返回前 50 条）"),
	)
}
