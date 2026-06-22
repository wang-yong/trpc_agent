/** 格式化工具函数 */

export function formatTime(ts: number): string {
  const d = new Date(ts * 1000)
  const now = new Date()
  const diff = (now.getTime() - d.getTime()) / 1000
  if (diff < 60) return '刚刚'
  if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
  if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
  return `${d.getMonth() + 1}/${d.getDate()}`
}

export function fmtDateTime(ts: number): string {
  const d = new Date(ts * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export function fmtNum(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return String(n)
}

export function fmtCost(v: number): string {
  return '¥' + v.toFixed(4)
}

export function shortModelName(name: string): string {
  const parts = name.split('/')
  return parts[parts.length - 1]
}

/** 硅基流动参考价格 (每百万 tokens) */
const PRICING: Record<string, { input: number; output: number }> = {
  default: { input: 0.001, output: 0.002 },
  'deepseek-ai/DeepSeek-V3': { input: 0.0001, output: 0.0001 },
  'Qwen/Qwen2.5-72B-Instruct': { input: 0.002, output: 0.01 },
  'Qwen/Qwen2.5-Coder-32B-Instruct': { input: 0.002, output: 0.006 },
}

export function calcCost(model: string, promptTok: number, completionTok: number): number {
  const p = PRICING[model] || PRICING.default
  return (promptTok / 1000000 * p.input + completionTok / 1000000 * p.output)
}

/** Markdown 渲染 */
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  highlight(str: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre><code class="hljs">${hljs.highlight(str, { language: lang }).value}</code></pre>`
      } catch { /* fall through */ }
    }
    return `<pre><code class="hljs">${md.utils.escapeHtml(str)}</code></pre>`
  },
})

export function renderMarkdown(text: string): string {
  return md.render(text)
}
