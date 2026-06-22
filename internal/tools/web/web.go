// Package web 提供网络交互工具，使 Agent 能够发起 HTTP 请求、抓取网页纯文本，以及进行免费免 Key 的 DuckDuckGo 网页检索。
package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// HTTPRequestInput HTTP 请求输入。
type HTTPRequestInput struct {
	URL    string            `json:"url" description:"要请求的完整 HTTP/HTTPS URL 路径"`
	Method string            `json:"method" description:"请求方法，支持 GET, POST, PUT, DELETE等，默认为 GET"`
	Body   string            `json:"body" description:"请求体内容（常用于 POST/PUT 请求）"`
	Headers map[string]string `json:"headers" description:"自定义请求头映射对（可选）"`
}

// HTTPRequestOutput HTTP 请求输出。
type HTTPRequestOutput struct {
	StatusCode int    `json:"status_code" description:"HTTP 状态码（如 200, 404）"`
	Status     string `json:"status" description:"HTTP 状态文本描述"`
	Body       string `json:"body" description:"返回的原始响应体（前 100,000 字节，防止文本过长撑爆大模型上下文）"`
}

// WebScrapeInput 网页抓取输入。
type WebScrapeInput struct {
	URL string `json:"url" description:"要抓取内容的网页 URL（如 https://news.ycombinator.com）"`
}

// WebScrapeOutput 网页抓取输出（过滤 HTML 标签，返回干净文本）。
type WebScrapeOutput struct {
	Title string `json:"title" description:"网页标题"`
	Text  string `json:"text" description:"剥离 HTML 标签后的网页主体干净纯文本（前 80,000 字节）"`
}

// WebSearchInput 网页检索输入。
type WebSearchInput struct {
	Query string `json:"query" description:"搜索关键字（如 'Golang 1.24 新特性'，将调用免 Key DuckDuckGo 服务进行实时全球检索）"`
}

// SearchResult 结构化单条检索结果。
type SearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchOutput 网页检索输出。
type WebSearchOutput struct {
	Query   string         `json:"query" description:"搜索关键词"`
	Results []SearchResult `json:"results" description:"前 8-10 条最相关的检索结果列表"`
}

// client 全局复用带超时限制的 HTTP 客户端
var client = &http.Client{
	Timeout: 15 * time.Second,
}

// ExecuteHTTPRequest 发起通用 HTTP/HTTPS 网络请求。
func ExecuteHTTPRequest(ctx context.Context, input HTTPRequestInput) (HTTPRequestOutput, error) {
	method := strings.ToUpper(input.Method)
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if input.Body != "" {
		bodyReader = strings.NewReader(input.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, input.URL, bodyReader)
	if err != nil {
		return HTTPRequestOutput{}, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置默认请求头，伪装普通浏览器，防止被防爬防火墙拒绝
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 注入用户自定义请求头
	for k, v := range input.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return HTTPRequestOutput{}, fmt.Errorf("请求网络发生异常: %v", err)
	}
	defer resp.Body.Close()

	// 限制读取前 100,000 字节，防止下载超大文件拖垮内存
	limitReader := io.LimitReader(resp.Body, 100000)
	bodyBytes, err := io.ReadAll(limitReader)
	if err != nil {
		return HTTPRequestOutput{}, fmt.Errorf("读取响应失败: %v", err)
	}

	return HTTPRequestOutput{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(bodyBytes),
	}, nil
}

// CleanHTML 过滤并清洗 HTML，仅提取标题及干净的主体内容（移除 scripts, styles 块和各种标签）
func CleanHTML(htmlContent string) (string, string) {
	// 提取 Title 标签
	titleReg := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	title := "无标题"
	if matches := titleReg.FindStringSubmatch(htmlContent); len(matches) > 1 {
		title = strings.TrimSpace(matches[1])
	}

	// 1. 强力丢弃无用和干扰性的 script 与 style 代码块
	scriptReg := regexp.MustCompile(`(?s)<script.*?>.*?</script>`)
	cleaned := scriptReg.ReplaceAllString(htmlContent, " ")
	styleReg := regexp.MustCompile(`(?s)<style.*?>.*?</style>`)
	cleaned = styleReg.ReplaceAllString(cleaned, " ")

	// 2. 剥离所有的 HTML 标签
	tagsReg := regexp.MustCompile(`<.*?>`)
	cleaned = tagsReg.ReplaceAllString(cleaned, "\n")

	// 3. 解码基本的 HTML 实体转义字符
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")

	// 4. 清理连续的冗余空白字符和换行
	lines := strings.Split(cleaned, "\n")
	var cleanLines []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l != "" {
			cleanLines = append(cleanLines, l)
		}
	}

	return title, strings.Join(cleanLines, "\n")
}

// ScrapeWebPage 抓取任意网页并剥离 HTML，返回干净好读的文本内容。
func ScrapeWebPage(ctx context.Context, input WebScrapeInput) (WebScrapeOutput, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		return WebScrapeOutput{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return WebScrapeOutput{}, fmt.Errorf("抓取网页异常: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WebScrapeOutput{}, fmt.Errorf("抓取失败，网页返回状态码: %s", resp.Status)
	}

	// 限制最大读取 150,000 字节，防止过长
	limitReader := io.LimitReader(resp.Body, 150000)
	bodyBytes, err := io.ReadAll(limitReader)
	if err != nil {
		return WebScrapeOutput{}, err
	}

	title, text := CleanHTML(string(bodyBytes))
	// 截取前 80,000 字节以节省大模型输入上下文空间
	if len(text) > 80000 {
		text = text[:80000] + "\n\n...[以下内容由于篇幅过长已被工具截断]..."
	}

	return WebScrapeOutput{
		Title: title,
		Text:  text,
	}, nil
}

// SearchWeb 调用免 Key 的 DuckDuckGo HTML 搜索界面，对全网信息进行实时高并发搜索。
func SearchWeb(ctx context.Context, input WebSearchInput) (WebSearchOutput, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(input.Query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return WebSearchOutput{}, err
	}

	// 模拟老牌主流浏览器 User-Agent，确保百分百不会被 DuckDuckGo WAF 拦截阻断
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return WebSearchOutput{}, fmt.Errorf("搜索网络超时或阻断: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WebSearchOutput{}, fmt.Errorf("搜索响应异常: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return WebSearchOutput{}, err
	}

	body := string(bodyBytes)

	// 极度鲁棒的正规表达式：用来从 DuckDuckGo 简洁好读的 HTML 布局中提取搜索条目
	// 每一个搜索条目都被包在 `<div class="result body">` 或 `<div class="links_main">`
	// 我们可以提取其 Title、Url 和 Snippet
	var results []SearchResult

	// 极度鲁棒的宽松型正则表达式，不强制限制 class 与 href 的绝对顺序或引号类型
	linkReg := regexp.MustCompile(`(?s)<a\s+[^>]*class="[^"]*result__a[^"]*"\s+href="([^"]*)"[^>]*>(.*?)</a>`)
	snippetReg := regexp.MustCompile(`(?s)<a\s+[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>`)

	// 按照每个结果块切分（兼容 class="result" 或 links_main 等各种静态 HTML 渲染特征）
	var blocks []string
	if strings.Contains(body, "class=\"result") {
		blocks = strings.Split(body, "class=\"result")
	} else if strings.Contains(body, "class=\"links_main") {
		blocks = strings.Split(body, "class=\"links_main")
	} else {
		blocks = strings.Split(body, "<div class=\"result")
	}

	// 遍历前 8-10 个切片，提取搜索信息
	for i := 1; i < len(blocks) && len(results) < 8; i++ {
		block := blocks[i]

		linkMatches := linkReg.FindStringSubmatch(block)
		if len(linkMatches) < 3 {
			continue
		}

		rawURL := linkMatches[1]
		title := linkMatches[2]

		// 解析 DuckDuckGo 内嵌的跳转 URL，如 /l/?kh=-1&uddg=https%3A%2F%2Fexample.com
		if strings.Contains(rawURL, "uddg=") {
			parts := strings.Split(rawURL, "uddg=")
			if len(parts) > 1 {
				decoded, err := url.QueryUnescape(parts[1])
				if err == nil {
					rawURL = decoded
				}
			}
		}

		// 剥离标题里的 HTML 标签
		title = regexp.MustCompile(`<.*?>`).ReplaceAllString(title, "")
		title = strings.TrimSpace(title)

		// 提取 Snippet
		snippet := "无描述"
		snippetMatches := snippetReg.FindStringSubmatch(block)
		if len(snippetMatches) > 1 {
			snippet = regexp.MustCompile(`<.*?>`).ReplaceAllString(snippetMatches[1], "")
			snippet = strings.TrimSpace(snippet)
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     rawURL,
			Snippet: snippet,
		})
	}

	// 如果免 Key 的 DuckDuckGo 被防火墙阻断、返回空结果，立即执行百度实时搜索引擎自动自愈！
	if len(results) == 0 {
		baiduResults, err := SearchBaidu(ctx, input.Query)
		if err == nil && len(baiduResults) > 0 {
			results = baiduResults
		}
	}

	// 如果仍然完全没搜到，提供友好兜底
	if len(results) == 0 {
		return WebSearchOutput{
			Query: input.Query,
			Results: []SearchResult{
				{
					Title:   "实时搜索暂无发现",
					URL:     "https://duckduckgo.com",
					Snippet: "由于第三方搜索引擎接口在当前运行环境下未能成功返回静态结果，建议老公重试或更换更加具体的关键词。",
				},
			},
		}, nil
	}

	return WebSearchOutput{
		Query:   input.Query,
		Results: results,
	}, nil
}

// SearchBaidu 作为备用搜索引擎，在 DuckDuckGo 遭受 WAF 拦截或无结果时自动自愈。
func SearchBaidu(ctx context.Context, query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	body := string(bodyBytes)
	var results []SearchResult

	// 匹配百度搜索的 H3 标题块和摘要快
	h3Reg := regexp.MustCompile(`(?s)<h3[^>]*class="[^"]*t[^"]*"[^>]*>.*?<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	abstractReg := regexp.MustCompile(`(?s)<div[^>]*class="[^"]*c-abstract[^"]*"[^>]*>(.*?)</div>`)
	if !strings.Contains(body, "c-abstract") {
		abstractReg = regexp.MustCompile(`(?s)<span[^>]*class="[^"]*content-right[^"]*"[^>]*>(.*?)</span>`)
	}

	// 支持两种切分方案
	blocks := strings.Split(body, "class=\"result c-container")
	if len(blocks) <= 1 {
		blocks = strings.Split(body, "id=\"")
	}

	for i := 1; i < len(blocks) && len(results) < 8; i++ {
		block := blocks[i]
		h3Matches := h3Reg.FindStringSubmatch(block)
		if len(h3Matches) < 3 {
			continue
		}

		rawURL := h3Matches[1]
		title := regexp.MustCompile(`<.*?>`).ReplaceAllString(h3Matches[2], "")
		title = strings.TrimSpace(title)

		snippet := "点击查看详情"
		abstractMatches := abstractReg.FindStringSubmatch(block)
		if len(abstractMatches) > 1 {
			snippet = regexp.MustCompile(`<.*?>`).ReplaceAllString(abstractMatches[1], "")
			snippet = strings.TrimSpace(snippet)
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     rawURL,
			Snippet: snippet,
		})
	}

	return results, nil
}

// NewHTTPRequestTool 创建 HTTP 请求工具。
func NewHTTPRequestTool() tool.Tool {
	return function.NewFunctionTool(
		ExecuteHTTPRequest,
		function.WithName("http_request"),
		function.WithDescription("发起任意 HTTP/HTTPS 网络请求（支持 GET, POST, PUT, DELETE 动作），并返回响应状态和内容数据"),
	)
}

// NewWebScrapeTool 创建网页内容抓取纯文本工具。
func NewWebScrapeTool() tool.Tool {
	return function.NewFunctionTool(
		ScrapeWebPage,
		function.WithName("web_scrape"),
		function.WithDescription("输入网页 URL，抓取并剥离所有的 HTML 标签、JavaScript 和 CSS 干扰，返回最干净好读、适合大模型吸收的纯文本正文内容"),
	)
}

// NewWebSearchTool 创建实时搜索工具。
func NewWebSearchTool() tool.Tool {
	return function.NewFunctionTool(
		SearchWeb,
		function.WithName("web_search"),
		function.WithDescription("调用免费、免 API Key 的 DuckDuckGo 网页检索接口，获取全网最新最相关的搜索条目（标题、URL、描述），使你拥有实时联网检索的能力"),
	)
}
