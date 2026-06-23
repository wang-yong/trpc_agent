/** API 类型定义和请求封装 */

const BASE = ''

export interface ModelItem {
  name: string
  display_name: string
}

export interface ModelsResponse {
  default: string
  models: ModelItem[]
}

export interface Skill {
  id: string
  name: string
  description: string
  icon: string
  prompt: string
}

export interface Session {
  id: string
  title: string
  model: string
  created_at: number
  updated_at: number
}

export interface TokenRecord {
  id: number
  session_id: string
  model: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  timestamp: number
}

export interface ModelTokenStat {
  model: string
  display_name: string
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface TokenStats {
  summary: {
    total_requests: number
    total_prompt: number
    total_completion: number
    total_tokens: number
  }
  by_model: ModelTokenStat[]
  recent: TokenRecord[]
}

function getUserId(): string {
  let uid = localStorage.getItem('trpc_agent_user_id')
  if (!uid) {
    uid = 'user-' + Math.random().toString(36).substring(2, 11) + '-' + Date.now().toString(36)
    localStorage.setItem('trpc_agent_user_id', uid)
  }
  return uid
}

async function request<T>(url: string, opts: RequestInit = {}): Promise<T> {
  const resp = await fetch(BASE + url, {
    headers: { 
      'Content-Type': 'application/json', 
      'X-User-Id': getUserId(),
      ...opts.headers 
    },
    ...opts,
  })
  if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
  return resp.json()
}

export const api = {
  getModels: () => request<ModelsResponse>('/api/models'),
  getSkills: () => request<Skill[]>('/api/skills'),
  getSessions: () => request<Session[]>('/api/sessions'),
  createSession: (title: string, model: string) =>
    request<Session>('/api/sessions', { method: 'POST', body: JSON.stringify({ title, model }) }),
  deleteSession: (id: string) =>
    request<{ ok: boolean }>(`/api/sessions?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),
  getTokenStats: () => request<TokenStats>('/api/token-stats'),
  respondApproval: (id: string, approve: boolean) =>
    request<{ ok: boolean }>('/api/approvals/respond', { method: 'POST', body: JSON.stringify({ id, approve }) }),
}

/** SSE 流式聊天 */
export interface ChatParams {
  message: string
  session_id: string
  model: string
  skill_id: string | null
}

export interface SSEHandlers {
  onDelta: (content: string) => void
  onThought?: (content: string) => void
  onToolCall?: (toolCall: { id: string; name: string; arguments: string }) => void
  onObservation?: (observation: { id: string; name: string; content: string }) => void
  onApprovalRequest?: (approval: { id: string; tool_name: string; arguments: string }) => void
  onUsage: (usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number }) => void
  onError: (msg: string) => void
  onDone: () => void
}

export async function streamChat(params: ChatParams, handlers: SSEHandlers): Promise<void> {
  const resp = await fetch('/api/chat', {
    method: 'POST',
    headers: { 
      'Content-Type': 'application/json',
      'X-User-Id': getUserId()
    },
    body: JSON.stringify(params),
  })

  if (!resp.ok || !resp.body) throw new Error(`请求失败：HTTP ${resp.status}`)

  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    let idx: number
    while ((idx = buffer.indexOf('\n\n')) !== -1) {
      const chunk = buffer.slice(0, idx)
      buffer = buffer.slice(idx + 2)
      handleSSEEvent(chunk, handlers)
    }
  }
}

function handleSSEEvent(chunk: string, handlers: SSEHandlers) {
  let event = 'message'
  let dataLines: string[] = []
  for (const line of chunk.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  }
  const dataRaw = dataLines.join('\n')
  let data: any = {}
  try { data = JSON.parse(dataRaw) } catch { /* ignore */ }

  if (event === 'delta' && data.content) {
    handlers.onDelta(data.content)
  } else if (event === 'thought' && data.content) {
    handlers.onThought?.(data.content)
  } else if (event === 'tool_call') {
    handlers.onToolCall?.(data)
  } else if (event === 'observation') {
    handlers.onObservation?.(data)
  } else if (event === 'approval_request') {
    handlers.onApprovalRequest?.(data)
  } else if (event === 'usage') {
    handlers.onUsage(data)
  } else if (event === 'error') {
    handlers.onError(data.message || '未知错误')
  } else if (event === 'done') {
    handlers.onDone()
  }
}
