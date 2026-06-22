import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { api, streamChat, type Session } from '@/api'
import { useSettingsStore } from './settings'

export interface ThinkingStep {
  id?: string
  type: 'thought' | 'tool'
  content: string
  toolName?: string
  args?: string
  status?: 'thinking' | 'running' | 'success' | 'error'
}

export interface Message {
  role: 'user' | 'ai'
  content: string
  timestamp?: string
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
  error?: boolean
  streaming?: boolean
  steps?: ThinkingStep[]
}

function getCurrentTime() {
  const d = new Date()
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${hours}:${minutes}`
}

export const useChatStore = defineStore('chat', () => {
  const localSessions = localStorage.getItem('trpc_agent_sessions')
  const localMessagesMap = localStorage.getItem('trpc_agent_messages_map')
  const localCurrentSessionId = localStorage.getItem('trpc_agent_current_id')

  const sessions = ref<Session[]>(localSessions ? JSON.parse(localSessions) : [])
  const currentSessionId = ref<string | null>(localCurrentSessionId || null)
  const messagesMap = ref<Record<string, Message[]>>(localMessagesMap ? JSON.parse(localMessagesMap) : {})
  const busy = ref(false)
  const sidebarCollapsed = ref(false)

  // 深度 watch 自动写入 localStorage，防止重新部署和刷新后清空任务列表与聊天
  watch(sessions, (newVal) => {
    localStorage.setItem('trpc_agent_sessions', JSON.stringify(newVal))
  }, { deep: true })

  watch(messagesMap, (newVal) => {
    localStorage.setItem('trpc_agent_messages_map', JSON.stringify(newVal))
  }, { deep: true })

  watch(currentSessionId, (newVal) => {
    if (newVal) {
      localStorage.setItem('trpc_agent_current_id', newVal)
    } else {
      localStorage.removeItem('trpc_agent_current_id')
    }
  })

  const currentMessages = computed(() => {
    if (!currentSessionId.value) return []
    return messagesMap.value[currentSessionId.value] || []
  })

  const sessionCount = computed(() => sessions.value.length)

  async function fetchSessions() {
    try {
      const remote = await api.getSessions()
      if (remote && remote.length > 0) {
        sessions.value = remote
      }
    } catch {
      // 容错处理：当 API 失败时，我们依然坚守并使用本地 localStorage 的缓存
    }
  }

  async function createSession(title: string, model: string): Promise<Session> {
    const sess = await api.createSession(title, model)
    sessions.value.unshift(sess)
    messagesMap.value[sess.id] = []
    return sess
  }

  async function deleteSession(id: string) {
    await api.deleteSession(id)
    sessions.value = sessions.value.filter(s => s.id !== id)
    delete messagesMap.value[id]
    if (currentSessionId.value === id) {
      currentSessionId.value = null
    }
  }

  function selectSession(id: string) {
    currentSessionId.value = id
  }

  function newTask() {
    currentSessionId.value = null
  }

  function clearMessages() {
    if (currentSessionId.value && messagesMap.value[currentSessionId.value]) {
      messagesMap.value[currentSessionId.value] = []
    }
  }

  async function sendMessage(text: string) {
    if (!text.trim() || busy.value) return

    const settings = useSettingsStore()

    // 确保有会话
    if (!currentSessionId.value) {
      const sess = await createSession(text.slice(0, 30), settings.currentModel)
      currentSessionId.value = sess.id
    }

    const sid = currentSessionId.value
    if (!messagesMap.value[sid]) messagesMap.value[sid] = []

    // 添加用户消息
    messagesMap.value[sid].push({ role: 'user', content: text, timestamp: getCurrentTime() })

    // 添加 AI 消息占位
    const aiMsg: Message = { role: 'ai', content: '', streaming: true, steps: [], timestamp: getCurrentTime() }
    messagesMap.value[sid].push(aiMsg)

    busy.value = true
    let answer = ''

    try {
      await streamChat(
        {
          message: text,
          session_id: sid,
          model: settings.currentModel,
          skill_id: settings.currentSkill,
        },
        {
          onDelta: (content) => {
            answer += content
            aiMsg.content = answer
          },
          onThought: (content) => {
            if (!aiMsg.steps) aiMsg.steps = []
            const steps = aiMsg.steps
            const last = steps[steps.length - 1]
            if (!last || last.type !== 'thought') {
              if (last && last.type === 'thought' && last.status === 'thinking') {
                last.status = 'success'
              }
              steps.push({ type: 'thought', content: content, status: 'thinking' })
            } else {
              last.content += content
            }
            aiMsg.steps = [...steps] // 解构重分配，强制触发 Vue 3 深度响应式重绘
          },
          onToolCall: (data) => {
            if (!aiMsg.steps) aiMsg.steps = []
            const steps = aiMsg.steps
            const last = steps[steps.length - 1]
            if (last && last.type === 'thought' && last.status === 'thinking') {
              last.status = 'success'
            }
            const existing = steps.find(s => s.id === data.id)
            if (existing) {
              existing.args = (existing.args || '') + data.arguments
            } else {
              steps.push({
                id: data.id,
                type: 'tool',
                toolName: data.name,
                args: data.arguments,
                status: 'running',
                content: '',
              })
            }
            aiMsg.steps = [...steps] // 解构重分配，强制触发 Vue 3 深度响应式重绘
          },
          onObservation: (data) => {
            if (!aiMsg.steps) aiMsg.steps = []
            const steps = aiMsg.steps
            const existing = steps.find(s => s.id === data.id)
            if (existing) {
              existing.status = 'success'
              existing.content = data.content
            } else {
              steps.push({
                id: data.id,
                type: 'tool',
                toolName: data.name,
                status: 'success',
                content: data.content,
              })
            }
            aiMsg.steps = [...steps] // 解构重分配，强制触发 Vue 3 深度响应式重绘
          },
          onUsage: (usage) => {
            aiMsg.usage = usage
          },
          onError: (msg) => {
            aiMsg.error = true
            aiMsg.content = '出错了：' + msg
          },
          onDone: () => {
            aiMsg.streaming = false
            if (aiMsg.steps) {
              aiMsg.steps.forEach(s => {
                if (s.status === 'thinking' || s.status === 'running') {
                  s.status = 'success'
                }
              })
            }
            if (!answer.trim()) {
              aiMsg.content = '(无内容返回)'
            }
          },
        }
      )

      // 更新会话标题
      const sess = sessions.value.find(s => s.id === sid)
      if (sess && sess.title === '新对话') {
        sess.title = text.slice(0, 30) + (text.length > 30 ? '...' : '')
      }
    } catch (err: any) {
      aiMsg.error = true
      aiMsg.content = '出错了：' + err.message
      aiMsg.streaming = false
    } finally {
      busy.value = false
    }
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  return {
    sessions,
    currentSessionId,
    messagesMap,
    busy,
    sidebarCollapsed,
    currentMessages,
    sessionCount,
    fetchSessions,
    createSession,
    deleteSession,
    selectSession,
    newTask,
    clearMessages,
    sendMessage,
    toggleSidebar,
  }
})
