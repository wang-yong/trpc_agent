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
  approval?: { id: string; toolName: string; arguments: string }
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
  const localCurrentSessionId = localStorage.getItem('trpc_agent_current_id')

  // 僵尸打字态物理清洗自愈防御：
  // 在初始化载入 local 缓存时，所有的历史会话消息在逻辑上绝对不应该处于流式打字中（streaming: true）。
  // 如果因为之前网络崩溃、服务关停、用户中途重启导致流未正常结束（即没有触发 onDone 闭合），
  // 我们在此行强行将所有历史消息的 streaming 置为 false，优雅绝杀任何残留的跳动光标。
  const parseLocalMessagesMap = (): Record<string, Message[]> => {
    const raw = localStorage.getItem('trpc_agent_messages_map')
    if (!raw) return {}
    try {
      const parsed = JSON.parse(raw) as Record<string, Message[]>
      Object.keys(parsed).forEach(sid => {
        if (parsed[sid]) {
          parsed[sid].forEach(msg => {
            if (msg.streaming) {
              msg.streaming = false
            }
            if (msg.steps) {
              msg.steps.forEach(step => {
                if (step.status === 'thinking' || step.status === 'running') {
                  step.status = 'success'
                }
              })
            }
          })
        }
      })
      return parsed
    } catch {
      return {}
    }
  }

  const sessions = ref<Session[]>(localSessions ? JSON.parse(localSessions) : [])
  const currentSessionId = ref<string | null>(localCurrentSessionId || null)
  const messagesMap = ref<Record<string, Message[]>>(parseLocalMessagesMap())
  const busy = ref(false)
  const sidebarCollapsed = ref(false)
  const currentApproval = ref<{ id: string; tool_name: string; arguments: string } | null>(null)

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
    
    // ====== 【液态打字缓冲器】核心自愈状态定义 ======
    const typingQueue: string[] = []
    let renderedText = ''
    let typingTimer: ReturnType<typeof setInterval> | null = null
    let streamEnded = false

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
            // 将接收到的多字符分片依次推入待打印队列
            for (const char of content) {
              typingQueue.push(char)
            }

            // 启动平滑打字心跳定时器
            if (!typingTimer) {
              const speed = settings.typingSpeed !== undefined ? settings.typingSpeed : 20
              
              // 如果速度设为 0，代表不限速，直接自愈回直出
              if (speed === 0) {
                typingQueue.forEach(char => renderedText += char)
                typingQueue.length = 0
                aiMsg.content = renderedText
                messagesMap.value[sid] = [...messagesMap.value[sid]]
                return
              }

              typingTimer = setInterval(() => {
                if (typingQueue.length > 0) {
                  // 动态积压加速保护机制：
                  // 如果队列中积存的待打字符过多，为避免打字进度延迟过大产生割裂，
                  // 自动以阶梯倍速一次性吐出更多字，确保极高频时的顺畅实时自愈！
                  let charsToTake = 1
                  if (typingQueue.length > 80) charsToTake = 5
                  else if (typingQueue.length > 30) charsToTake = 2

                  for (let i = 0; i < charsToTake; i++) {
                    const char = typingQueue.shift()
                    if (char !== undefined) {
                      renderedText += char
                    }
                  }
                  aiMsg.content = renderedText
                  messagesMap.value[sid] = [...messagesMap.value[sid]] // 强刷 Vue 3 重绘
                } else if (streamEnded) {
                  // 彻底吐完，且后端也已经物理闭环，打字机优雅收工消退！
                  if (typingTimer) {
                    clearInterval(typingTimer)
                    typingTimer = null
                  }
                  aiMsg.streaming = false
                  messagesMap.value[sid] = [...messagesMap.value[sid]]
                }
              }, speed)
            }
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
            messagesMap.value[sid] = [...messagesMap.value[sid]] // 彻底唤醒深层响应式，瞬间触发组件刷新
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
            messagesMap.value[sid] = [...messagesMap.value[sid]] // 彻底唤醒深层响应式，瞬间触发组件刷新
          },
          onApprovalRequest: (data) => {
            console.log('[DEBUG APPROVAL] 收到后端审批事件:', data)
            currentApproval.value = data
            if (!aiMsg.steps) aiMsg.steps = []
            const steps = aiMsg.steps
            const last = steps[steps.length - 1]
            if (last && last.status === 'thinking') {
              last.status = 'success'
            }
            
            // 寻找当前最靠近尾部的 type === 'tool' 的 step (也就是正在申请审批的这个工具)
            const idx = [...steps].reverse().findIndex(s => s.type === 'tool')
            if (idx !== -1) {
              const realIndex = steps.length - 1 - idx
              console.log('[DEBUG APPROVAL] 已成功定位并关联工具步骤, 索引:', realIndex, steps[realIndex].toolName)
              steps[realIndex] = {
                ...steps[realIndex],
                approval: {
                  id: data.id,
                  toolName: data.tool_name,
                  arguments: data.arguments
                }
              }
            } else {
              console.log('[DEBUG APPROVAL] 未找到已有的工具步骤，走兜底 push 新卡片')
              steps.push({
                id: data.id,
                type: 'tool',
                toolName: data.tool_name,
                args: data.arguments,
                status: 'running',
                content: '',
                approval: {
                  id: data.id,
                  toolName: data.tool_name,
                  arguments: data.arguments
                }
              })
            }
            aiMsg.steps = [...steps] // 解构重分配，强制触发 Vue 3 深度响应式重绘
            messagesMap.value[sid] = [...messagesMap.value[sid]] // 彻底唤醒深层响应式，瞬间触发组件刷新
          },
          onWorkspaceUpdated: (data) => {
            window.dispatchEvent(new CustomEvent('workspace-updated', { detail: data }))
          },
          onUsage: (usage) => {
            aiMsg.usage = usage
          },
          onError: (msg) => {
            aiMsg.error = true
            aiMsg.content = '出错了：' + msg
          },
          onDone: () => {
            streamEnded = true
            
            // 如果打字速度为 0 (不限流)，或者本来打字队列里就没积压字符，则立刻秒速完成
            if (settings.typingSpeed === 0 || typingQueue.length === 0) {
              if (typingTimer) {
                clearInterval(typingTimer)
                typingTimer = null
              }
              aiMsg.streaming = false
              if (!renderedText.trim()) {
                aiMsg.content = '(无内容返回)'
              }
              messagesMap.value[sid] = [...messagesMap.value[sid]]
            }

            if (aiMsg.steps) {
              aiMsg.steps.forEach(s => {
                if (s.status === 'thinking' || s.status === 'running') {
                  s.status = 'success'
                }
              })
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

  async function respondApproval(approve: boolean, approvalId?: string) {
    const id = approvalId || (currentApproval.value ? currentApproval.value.id : null)
    if (!id) return

    if (currentApproval.value && currentApproval.value.id === id) {
      currentApproval.value = null // 瞬间重置
    }

    // 遍历所有消息的步骤，把匹配到的 approval 信息清除，并写入温馨的操作提示
    Object.keys(messagesMap.value).forEach(sid => {
      if (messagesMap.value[sid]) {
        messagesMap.value[sid].forEach(msg => {
          if (msg.steps) {
            const steps = [...msg.steps]
            let changed = false
            for (let i = 0; i < steps.length; i++) {
              if (steps[i].approval && steps[i].approval?.id === id) {
                steps[i] = {
                  ...steps[i],
                  content: approve ? '⚡ 已批准，正在执行动作...' : '❌ 已拒绝，取消动作执行',
                  status: approve ? steps[i].status : 'error',
                  approval: undefined
                }
                changed = true
              }
            }
            if (changed) {
              msg.steps = steps // 重新分配数组引用，100% 触发重绘
            }
          }
        })
      }
    })

    try {
      await api.respondApproval(id, approve)
    } catch {
      // 容错
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
    currentApproval,
    respondApproval,
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
