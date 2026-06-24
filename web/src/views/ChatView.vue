<script setup lang="ts">
import { ref, nextTick, watch, computed, onMounted, onUnmounted } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useSettingsStore } from '@/stores/settings'
import MessageBubble from '@/components/chat/MessageBubble.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import ThinkingChain from '@/components/agent/ThinkingChain.vue'
import FileTreeItem from '@/components/chat/FileTreeItem.vue'
import FilePreviewModal from '@/components/chat/FilePreviewModal.vue'
import { api, type FileNode } from '@/api'

const chat = useChatStore()
const settings = useSettingsStore()
const messagesWrap = ref<HTMLElement>()

let lastScrollTime = 0
function scrollToBottom(force = false) {
  const now = Date.now()
  // 50ms 频率限制阀：在流式输出超高频触发时进行物理限流，绝杀浏览器因 smooth 缓动堆叠造成的滚动条严重粘滞、拖不动 Bug
  if (!force && now - lastScrollTime < 50) return
  lastScrollTime = now

  nextTick(() => {
    if (messagesWrap.value) {
      messagesWrap.value.scrollTo({
        top: messagesWrap.value.scrollHeight,
        // 吐字期间使用极速 'auto' 强制紧跟贴底，防漏白；只有新消息到达或用户手动切回合时才使用 'smooth' 带来舒适的滑入感
        behavior: force ? 'smooth' : 'auto',
      })
    }
  })
}

// 监听消息数组长度变动（新回合/新提问消息到达），强制使用 smooth 优雅滑入
watch(() => chat.currentMessages.length, () => scrollToBottom(true))

// 监听最后一个 AI 回复的流式吐字，使用极速 auto 实时贴底追踪，拒绝任何粘滞卡顿
watch(() => {
  const msgs = chat.currentMessages
  return msgs.length ? msgs[msgs.length - 1].content : ''
}, () => scrollToBottom(false))

// 灵感提示示例
const inspirations = [
  { icon: '💡', title: '解释一段代码', desc: '把代码贴给我，我会逐行讲解' },
  { icon: '🐛', title: '排查 Bug', desc: '把错误日志或现象告诉我' },
  { icon: '✨', title: '生成新功能', desc: '描述你的需求，我帮你设计实现' },
  { icon: '🔧', title: '重构优化', desc: '把现有代码交给我优化' },
]

function askInspiration(item: typeof inspirations[number]) {
  const text = `${item.desc}`
  chat.sendMessage(text)
}

// ====== Trae 风格：从当前会话的步骤中，动态、响应式地提取全网搜索和参考资料 ======
const searchReferences = computed(() => {
  const refs: { title: string; url: string; snippet: string; source: string }[] = []
  chat.currentMessages.forEach(msg => {
    if (msg.role === 'ai' && msg.steps) {
      msg.steps.forEach(step => {
        if (step.type === 'tool' && step.toolName === 'web_search' && step.content) {
          try {
            const data = JSON.parse(step.content)
            if (data && Array.isArray(data.results)) {
              data.results.forEach((r: any) => {
                // 跳过无搜索结果的兜底条目
                if (r.title && !r.title.includes("暂无发现") && r.url) {
                  const isBaidu = r.url.includes("baidu.com") || r.title.includes("百度")
                  refs.push({
                    title: r.title,
                    url: r.url,
                    snippet: r.snippet || '',
                    source: isBaidu ? '百度' : '全网',
                  })
                }
              })
            }
          } catch {
            // ignore
          }
        }
      })
    }
  })
  return refs
})

// ====== 上下文窗口利用率计算 ======
const lastUsageInfo = computed(() => {
  const msgs = chat.currentMessages
  if (msgs.length === 0) return { pct: 0, total: 0 }
  // 找最后一个有用量信息的消息
  for (let i = msgs.length - 1; i >= 0; i--) {
    if (msgs[i].usage) {
      const u = msgs[i].usage!
      const limit = 1000000 // 上行 1M (1,000,000) 限制
      const pct = Math.min(Math.round((u.total_tokens / limit) * 100), 100)
      return { pct, total: u.total_tokens }
    }
  }
  return { pct: 0, total: 0 }
})

// ====== 自定义 Workspace 运行根目录与递归文件树 ======
const workspaceRoot = ref('')
const fileTree = ref<FileNode[]>([])
const filesLoading = ref(false)
const editingPath = ref(false)
const newPathInput = ref('')

async function loadWorkspaceAndFiles() {
  filesLoading.value = true
  try {
    const resSettings = await api.getSettings()
    workspaceRoot.value = resSettings.workspace_root
    newPathInput.value = resSettings.workspace_root

    // 毫秒级同步后端 safety.yaml 的最新 typing_speed 极客打字流速配置！
    if (resSettings.typing_speed !== undefined) {
      settings.updateTypingSpeed(resSettings.typing_speed)
    }

    const res = await api.getWorkspaceFiles()
    fileTree.value = res.files
  } catch (err) {
    console.error('加载工作区文件树失败:', err)
  } finally {
    filesLoading.value = false
  }
}

async function saveNewWorkspace() {
  const path = newPathInput.value.trim()
  if (!path) return

  try {
    await api.saveSettings(path)
    editingPath.value = false
    await loadWorkspaceAndFiles()
  } catch (err: any) {
    // 优雅抛出后端返回的 Stat 真实物理路径不存在报错
    alert(err.message || '路径切换失败，请确认您输入的绝对路径在本地物理存在！')
  }
}

// ====== 动态右侧面板宽度拖拽管理 (支持 localStorage 持久化) ======
const localRightWidth = localStorage.getItem('trpc_agent_right_panel_width')
const rightPanelWidth = ref<number>(localRightWidth ? parseInt(localRightWidth, 10) : 280)

function initRightResize(e: MouseEvent) {
  e.preventDefault()
  const startX = e.clientX
  const startWidth = rightPanelWidth.value

  document.body.classList.add('is-resizing')

  const handleMouseMove = (moveEvent: MouseEvent) => {
    const diffX = moveEvent.clientX - startX
    const newWidth = startWidth - diffX
    // 彻底解禁右侧拉伸限制！只保留 20px 极简物理安全防负值崩溃兜底
    rightPanelWidth.value = Math.max(20, newWidth)
  }

  const handleMouseUp = () => {
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
    document.body.classList.remove('is-resizing')
    localStorage.setItem('trpc_agent_right_panel_width', String(rightPanelWidth.value))
  }

  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
}

async function openSelectFolderDialog() {
  try {
    const res = await api.openSelectFolderDialog()
    if (res.ok && !res.canceled && res.path) {
      newPathInput.value = res.path
      await saveNewWorkspace()
    }
  } catch (err: any) {
    console.error('拉起文件夹对话框失败:', err)
  }
}

onMounted(() => {
  loadWorkspaceAndFiles()

  // 智能捕获后端实时推送的工作区文件变动事件，全自动、秒级刷新资源管理器文件树！
  window.addEventListener('workspace-updated', loadWorkspaceAndFiles)
})

onUnmounted(() => {
  window.removeEventListener('workspace-updated', loadWorkspaceAndFiles)
})
</script>

<template>
  <div class="chat-view-container">
    <!-- 三栏布局的主体左侧：对话及输入区域 -->
    <div class="chat-main-column">
      <!-- Header -->
      <header class="chat-header">
        <div class="header-left">
          <button class="icon-btn" @click="chat.toggleSidebar()" title="切换侧栏">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
          </button>
          <span class="title">Agent</span>
          <span v-if="settings.currentSkillData" class="skill-badge">
            {{ settings.currentSkillData.icon }} {{ settings.currentSkillData.name }}
          </span>
        </div>
        <button
          class="clear-btn"
          @click="chat.clearMessages()"
          :disabled="!chat.currentSessionId || chat.busy"
        >清空</button>
      </header>

      <!-- Messages Area -->
      <main class="messages-wrap" ref="messagesWrap">
        <!-- Welcome Screen -->
        <section v-if="chat.currentMessages.length === 0" class="welcome">
          <!-- 背景装饰光晕 -->
          <div class="bg-glow glow-1"></div>
          <div class="bg-glow glow-2"></div>

          <div class="welcome-inner">
            <!-- 头像 + 标题 -->
            <div class="brand">
              <div class="brand-avatar">
                <div class="avatar-ring"></div>
                <div class="avatar-core">
                  <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
                    <defs>
                      <linearGradient id="bgrad" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" stop-color="#7ba2ff"/>
                        <stop offset="100%" stop-color="#b794f6"/>
                      </linearGradient>
                    </defs>
                    <path d="M20 6 L29 11 L29 23 C29 28 25 32 20 34 C15 32 11 28 11 23 L11 11 Z"
                          fill="url(#bgrad)"/>
                    <circle cx="16" cy="20" r="1.8" fill="#fff"/>
                    <circle cx="24" cy="20" r="1.8" fill="#fff"/>
                    <path d="M16 25c1.6 1.2 4.4 1.2 6 0" stroke="#fff" stroke-width="1.6" stroke-linecap="round" fill="none"/>
                  </svg>
                </div>
              </div>
              <h1 class="brand-title">
                <span class="title-text">AI Agent</span>
                <span class="title-tag">Beta</span>
              </h1>
              <p class="brand-sub">你好，今天我能为你做什么？</p>
            </div>

            <!-- 灵感提示卡（Claude / Cursor 风格） -->
            <div class="inspire-grid">
              <button
                v-for="(item, i) in inspirations"
                :key="i"
                class="inspire-card"
                @click="askInspiration(item)"
              >
                <div class="inspire-icon">{{ item.icon }}</div>
                <div class="inspire-text">
                  <div class="inspire-title">{{ item.title }}</div>
                  <div class="inspire-desc">{{ item.desc }}</div>
                </div>
                <svg class="inspire-arrow" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M7 17L17 7M9 7h8v8"/></svg>
              </button>
            </div>

            <!-- 技能选择条 -->
            <div class="skill-strip" v-if="settings.skills.length">
              <span class="strip-label">技能</span>
              <div class="strip-list">
                <button
                  v-for="sk in settings.skills.slice(0, 6)"
                  :key="sk.id"
                  class="strip-chip"
                  @click="settings.selectSkill(sk.id)"
                >
                  <span class="chip-ico">{{ sk.icon }}</span>
                  <span>{{ sk.name }}</span>
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- Message List -->
        <div v-else class="messages">
          <template v-for="(msg, idx) in chat.currentMessages" :key="idx">
            <div v-if="msg.role === 'ai'" class="ai-msg-group">
              <ThinkingChain :steps="msg.steps" />
              <MessageBubble
                :role="msg.role"
                :content="msg.content"
                :streaming="msg.streaming"
                :error="msg.error"
                :usage="msg.usage"
                :timestamp="msg.timestamp"
              />
            </div>
            <MessageBubble
              v-else
              :role="msg.role"
              :content="msg.content"
              :streaming="msg.streaming"
              :error="msg.error"
              :usage="msg.usage"
              :timestamp="msg.timestamp"
            />
          </template>
        </div>
      </main>

      <!-- Input -->
      <ChatInput :busy="chat.busy" @send="chat.sendMessage" />
    </div>

    <!-- 右侧面板拖拽手柄条 -->
    <div class="resize-handle-right" @mousedown="initRightResize"></div>

    <!-- 三栏布局的主体右侧：Trae 风格的 Context & Tasks 侧边面板 -->
    <aside class="chat-right-panel" :style="{ width: rightPanelWidth + 'px' }">
      <!-- 1. 工作区资源管理器区块 -->
      <div class="panel-section flex-shrink-0" style="max-height: 320px; display: flex; flex-direction: column;">
        <div class="section-header-row">
          <h3 class="section-title">工作区</h3>
          <button class="refresh-btn" @click="loadWorkspaceAndFiles" :disabled="filesLoading" style="margin-left: auto; padding: 2px 6px; font-size: 11px; display: inline-flex; align-items: center; gap: 4px;">
            <svg :class="{ pulsing: filesLoading }" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3" stroke-linecap="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10"/></svg>
            刷新
          </button>
        </div>
        
        <!-- 运行根目录修改/展示框 -->
        <div class="workspace-root-bar">
          <template v-if="!editingPath">
            <span class="path-text" :title="workspaceRoot">{{ workspaceRoot || '加载中...' }}</span>
            <button class="edit-path-btn" @click="openSelectFolderDialog" title="弹窗选择本地文件夹" style="margin-right: 4px;">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
            </button>
            <button class="edit-path-btn" @click="editingPath = true" title="手动输入绝对路径">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
            </button>
          </template>
          <template v-else>
            <input
              v-model="newPathInput"
              type="text"
              class="path-input-field"
              placeholder="请输入真实的绝对路径"
              @keyup.enter="saveNewWorkspace"
            />
            <div class="path-actions">
              <button class="path-act-btn cancel" @click="editingPath = false" title="取消">✕</button>
              <button class="path-act-btn save" @click="saveNewWorkspace" title="保存">✓</button>
            </div>
          </template>
        </div>

        <!-- 递归文件树滚动视图 -->
        <div class="file-tree-container scrollable">
          <div v-if="filesLoading && fileTree.length === 0" class="tree-loading">
            <span class="pulse-dot"></span> 正在扫描工作区...
          </div>
          <div v-else-if="fileTree.length > 0" class="tree-list">
            <FileTreeItem
              v-for="node in fileTree"
              :key="node.path"
              :node="node"
            />
          </div>
          <div v-else class="tree-empty">
            工作区无任何文件或未加载
          </div>
        </div>
      </div>

      <div class="panel-divider"></div>

      <!-- 2. 上下文利用率区块 -->
      <div class="panel-section">
        <div class="section-header-row">
          <h3 class="section-title">上下文</h3>
          <button class="compress-btn" :disabled="lastUsageInfo.total === 0">压缩</button>
        </div>
        <div class="context-body">
          <div class="progress-container">
            <div class="progress-bar-bg">
              <div class="progress-fill" :style="{ width: lastUsageInfo.pct + '%' }"></div>
            </div>
            <span class="progress-pct">{{ lastUsageInfo.pct }}%</span>
          </div>
          <div class="context-meta">
            <span class="tokens-count">已使用 {{ lastUsageInfo.total }} Tokens</span>
            <div class="legends">
              <span class="legend-item"><span class="legend-dot search"></span>联网搜索</span>
              <span class="legend-item"><span class="legend-dot other"></span>其他</span>
            </div>
          </div>
        </div>
      </div>

      <div class="panel-divider"></div>

      <!-- 3. 联网搜索及参考资料区块 -->
      <div class="panel-section flex-grow">
        <h3 class="section-title">参考资料</h3>
        <div v-if="searchReferences.length > 0" class="ref-list">
          <a
            v-for="(refItem, i) in searchReferences"
            :key="i"
            :href="refItem.url"
            target="_blank"
            class="ref-item"
            :title="refItem.snippet"
          >
            <span class="ref-badge" :class="refItem.source === '百度' ? 'baidu' : 'web'">
              {{ refItem.source }}
            </span>
            <span class="ref-title">{{ refItem.title }}</span>
          </a>
        </div>
        <div v-else class="empty-state">
          <div class="empty-icon">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </div>
          <div class="empty-text">无参考资料</div>
          <div class="empty-desc">联网搜索到的资讯和引用会排列在下方</div>
        </div>
      </div>
    </aside>

    <!-- 顶奢级文件内容自适应内嵌式预览大弹窗 -->
    <FilePreviewModal />
  </div>
</template>

<style scoped>
.chat-view-container {
  height: 100%;
  display: flex;
  background: var(--body-color);
  position: relative;
  overflow: hidden;
}

/* ===== 左/中栏：聊天及主内容区 ===== */
.chat-main-column {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 0;
  border-right: 1px solid var(--border-color);
}

/* ===== Header ===== */
.chat-header {
  padding: 0 24px;
  height: 52px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  background: var(--body-color);
  position: relative;
  z-index: 2;
}
.header-left {
  display: flex; align-items: center; gap: 11px;
}
.icon-btn {
  width: 34px; height: 34px; border-radius: 9px;
  background: transparent; border: none;
  color: var(--text-color-3); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all .15s;
}
.icon-btn:hover {
  background: var(--hover-color); color: var(--text-color-2);
}
.title {
  font-size: 15px; font-weight: 600; letter-spacing: -0.25px;
}
.skill-badge {
  font-size: 11.5px; padding: 3px 10px; border-radius: 6px;
  background: rgba(107,139,245,.12); color: var(--primary-color);
  font-weight: 500;
}
.clear-btn {
  padding: 6px 14px; border-radius: 8px;
  background: transparent; border: 1px solid var(--divider-color);
  color: var(--text-color-3); font-size: 12.5px; cursor: pointer;
  transition: all .15s; font-family: inherit;
}
.clear-btn:hover:not(:disabled) {
  color: var(--text-color-2);
  border-color: var(--border-color);
  background: var(--hover-color);
}
.clear-btn:disabled { opacity: .35; cursor: not-allowed; }

/* ===== Messages Wrap ===== */
.messages-wrap {
  flex: 1; overflow-y: auto; overflow-x: hidden;
  position: relative;
}

/* ===== Welcome — Claude / Cursor 风格 ===== */
.welcome {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px 60px;
  position: relative;
}

/* 背景装饰光晕 */
.bg-glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
  z-index: 0;
}
.glow-1 {
  width: 380px; height: 380px;
  background: radial-gradient(circle, rgba(107,139,245,.22) 0%, transparent 70%);
  top: 10%; left: 18%;
}
.glow-2 {
  width: 320px; height: 320px;
  background: radial-gradient(circle, rgba(167,139,250,.18) 0%, transparent 70%);
  bottom: 15%; right: 18%;
}
:root.dark .glow-1 { background: radial-gradient(circle, rgba(91,141,239,.18) 0%, transparent 70%); }
:root.dark .glow-2 { background: radial-gradient(circle, rgba(167,139,250,.12) 0%, transparent 70%); }

.welcome-inner {
  position: relative;
  z-index: 1;
  max-width: 640px;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 36px;
}

/* Brand 区域 */
.brand {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}
.brand-avatar {
  position: relative;
  width: 72px; height: 72px;
  display: flex; align-items: center; justify-content: center;
  margin-bottom: 4px;
}
.avatar-ring {
  position: absolute; inset: 0;
  border-radius: 22px;
  background: linear-gradient(135deg, #6b8bf5 0%, #a78bfa 100%);
  opacity: .35;
  filter: blur(14px);
  animation: pulse 3s ease-in-out infinite;
}
.avatar-core {
  position: relative;
  width: 64px; height: 64px;
  border-radius: 18px;
  background: linear-gradient(135deg, #7ba2ff 0%, #b794f6 100%);
  display: flex; align-items: center; justify-content: center;
  box-shadow:
    0 8px 24px rgba(107,139,245,.35),
    inset 0 1px 0 rgba(255,255,255,.2);
}
@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: .35; }
  50% { transform: scale(1.1); opacity: .5; }
}

.brand-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
}
.title-text {
  font-size: 30px;
  font-weight: 700;
  letter-spacing: -0.6px;
  background: linear-gradient(135deg, #6b8bf5 0%, #a78bfa 50%, #c084fc 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.title-tag {
  font-size: 10px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 6px;
  background: linear-gradient(135deg, rgba(107,139,245,.15), rgba(167,139,250,.15));
  color: var(--primary-color);
  letter-spacing: 0.5px;
}
.brand-sub {
  margin: 0;
  font-size: 14.5px;
  color: var(--text-color-3);
  letter-spacing: -0.1px;
}

/* 灵感卡片网格 */
.inspire-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}
.inspire-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 14px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  transition: all .2s cubic-bezier(.4, 0, .2, 1);
  position: relative;
  overflow: hidden;
}
.inspire-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(107,139,245,.06), rgba(167,139,250,.04));
  opacity: 0;
  transition: opacity .2s;
}
.inspire-card:hover {
  border-color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow:
    0 8px 24px rgba(107,139,245,.15),
    0 0 0 .5px rgba(107,139,245,.4);
}
.inspire-card:hover::before { opacity: 1; }
.inspire-card:hover .inspire-arrow {
  opacity: 1;
  transform: translate(0, 0);
}

.inspire-icon {
  font-size: 22px;
  width: 38px; height: 38px;
  display: flex; align-items: center; justify-content: center;
  background: var(--hover-color);
  border-radius: 11px;
  flex-shrink: 0;
  position: relative;
  z-index: 1;
}
.inspire-text {
  flex: 1;
  min-width: 0;
  position: relative;
  z-index: 1;
}
.inspire-title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text-color-1);
  margin-bottom: 2px;
}
.inspire-desc {
  font-size: 12px;
  color: var(--text-color-3);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inspire-arrow {
  color: var(--primary-color);
  opacity: 0;
  transform: translate(-4px, 4px);
  transition: all .2s;
  flex-shrink: 0;
  position: relative;
  z-index: 1;
}

/* 技能条 */
.skill-strip {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 14px;
}
.strip-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color-3);
  flex-shrink: 0;
  padding-left: 4px;
}
.strip-list {
  display: flex;
  gap: 6px;
  flex: 1;
  overflow-x: auto;
  scrollbar-width: none;
}
.strip-list::-webkit-scrollbar { display: none; }
.strip-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 8px;
  background: transparent;
  border: 1px solid var(--divider-color);
  color: var(--text-color-2);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all .15s;
  font-family: inherit;
  white-space: nowrap;
  flex-shrink: 0;
}
.strip-chip:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: rgba(107,139,245,.08);
}
.chip-ico { font-size: 13px; }

/* ===== Message List ===== */
.messages {
  max-width: 820px; width: 100%; margin: 0 auto;
  padding: 24px 24px 12px;
  display: flex; flex-direction: column; gap: 20px;
}
.ai-msg-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* ===== 右侧 Trae 面板 ===== */
.chat-right-panel {
  height: 100%;
  background: var(--body-color);
  display: flex;
  flex-direction: column;
  padding: 20px 16px;
  flex-shrink: 0;
  overflow-y: auto;
  min-width: 0;
}
.panel-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.panel-section.flex-grow {
  flex: 1;
}
.panel-divider {
  height: 1px;
  background: var(--divider-color);
  margin: 18px 0;
}
.section-title {
  font-size: 11.5px;
  font-weight: 700;
  color: var(--text-color-3);
  text-transform: uppercase;
  letter-spacing: .8px;
  margin: 0;
}
.section-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* 待办事项 */
.todo-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.todo-item {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  border-radius: 8px;
}
.todo-bullet {
  width: 6px; height: 6px; border-radius: 50%;
  background: #f59e0b;
}
.todo-bullet.pulsing {
  animation: todo-pulse 1.8s infinite ease-in-out;
}
@keyframes todo-pulse {
  0%, 100% { opacity: .4; transform: scale(.9); }
  50% { opacity: 1; transform: scale(1.15); }
}
.todo-name {
  font-size: 12px;
  color: var(--text-color-2);
  font-weight: 500;
}

/* 压缩按钮 */
.compress-btn {
  padding: 2px 8px;
  font-size: 10.5px;
  border-radius: 5px;
  background: var(--hover-color);
  border: 1px solid var(--border-color);
  color: var(--text-color-3);
  cursor: pointer;
  font-weight: 600;
  transition: all .12s;
}
.compress-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
}
.compress-btn:disabled {
  opacity: .35; cursor: not-allowed;
}

/* 上下文仪表 */
.context-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 12px;
}
.progress-container {
  display: flex;
  align-items: center;
  gap: 10px;
}
.progress-bar-bg {
  flex: 1;
  height: 6px;
  background: var(--hover-color);
  border-radius: 3px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: var(--primary-color);
  border-radius: 3px;
  transition: width .5s ease;
}
.progress-pct {
  font-size: 11px;
  font-weight: 700;
  color: var(--primary-color);
}
.context-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.tokens-count {
  font-size: 11px;
  color: var(--text-color-3);
  font-weight: 500;
}
.legends {
  display: flex;
  gap: 8px;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10.5px;
  color: var(--text-color-3);
  font-weight: 500;
}
.legend-dot {
  width: 5px; height: 5px; border-radius: 50%;
}
.legend-dot.search { background: var(--primary-color); }
.legend-dot.other { background: var(--text-color-3); opacity: .4; }

/* 联网参考资料列表 */
.ref-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 380px;
  overflow-y: auto;
}
.ref-item {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  padding: 10px 12px;
  border-radius: 8px;
  text-decoration: none;
  transition: all .15s;
  min-width: 0;
}
.ref-item:hover {
  border-color: var(--primary-color);
  transform: translateX(1px);
}
.ref-badge {
  font-size: 9px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 4px;
  flex-shrink: 0;
}
.ref-badge.web { background: rgba(91,141,239,.1); color: var(--primary-color); }
.ref-badge.baidu { background: rgba(248,81,73,.08); color: #f85149; }

.ref-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-color-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}
.ref-item:hover .ref-title {
  color: var(--primary-color);
}

/* 侧边面板空状态 */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 24px 12px;
  background: transparent;
  border: 1px dashed var(--divider-color);
  border-radius: 10px;
}
.empty-icon {
  color: var(--text-color-3);
  opacity: .35;
  margin-bottom: 8px;
}
.empty-text {
  font-size: 12px;
  font-weight: 700;
  color: var(--text-color-3);
  margin-bottom: 4px;
}
.empty-desc {
  font-size: 10.5px;
  color: var(--text-color-3);
  opacity: .65;
  line-height: 1.4;
}

@media (max-width: 900px) {
  .chat-right-panel { display: none; }
}
@media (max-width: 600px) {
  .welcome-inner { gap: 24px; }
  .inspire-grid { grid-template-columns: 1fr; }
  .title-text { font-size: 24px; }
  .glow-1, .glow-2 { width: 220px; height: 220px; }
}

/* ===== 5. 侧边栏工作区资源管理器样式 ===== */
.workspace-root-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 6px 10px;
  margin-bottom: 10px;
  min-width: 0;
}
.path-text {
  font-size: 11px;
  color: var(--text-color-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  font-family: monospace;
}
.edit-path-btn {
  background: transparent;
  border: none;
  color: var(--text-color-3);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  width: 20px; height: 20px; border-radius: 4px;
  transition: all .15s;
}
.edit-path-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}
.path-input-field {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text-color);
  font-size: 11px;
  font-family: monospace;
  outline: none;
  min-width: 0;
}
.path-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
.path-act-btn {
  background: transparent;
  border: none;
  width: 18px; height: 18px;
  border-radius: 4px;
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  font-size: 10px;
  transition: all .15s;
}
.path-act-btn.cancel {
  color: #f85149;
}
.path-act-btn.cancel:hover {
  background: rgba(248, 81, 73, 0.1);
}
.path-act-btn.save {
  color: #10b981;
}
.path-act-btn.save:hover {
  background: rgba(16, 185, 129, 0.1);
}

.file-tree-container {
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 10px;
  flex: 1;
  overflow-y: auto;
  min-height: 120px;
  max-height: 240px;
}
.tree-loading {
  font-size: 11.5px;
  color: var(--text-color-3);
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 4px;
}
.pulse-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--primary-color);
  animation: pulse-ani 1.5s infinite ease-in-out;
}
@keyframes pulse-ani {
  0%, 100% { opacity: .4; transform: scale(.85); }
  50% { opacity: 1; transform: scale(1.1); }
}
.tree-empty {
  font-size: 11.5px;
  color: var(--text-color-3);
  text-align: center;
  padding: 24px 12px;
  opacity: .75;
}
.tree-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* 刷新旋转动画 */
.refresh-btn svg.pulsing {
  animation: spin-ani 1.2s infinite linear;
}
@keyframes spin-ani {
  to { transform: rotate(360deg); }
}

/* ===== 4. 高危人机协作审批阻断弹窗 (HITL Glass-Modal) ===== */


/* fade transition */
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* ===== 右侧面板原生拖拽手柄条 ===== */
.resize-handle-right {
  width: 6px;
  background: transparent;
  cursor: col-resize !important;
  z-index: 100;
  position: relative;
  margin-left: -3px; /* 把它微调到正中心，覆盖两栏边框 */
  margin-right: -3px;
  flex-shrink: 0;
  transition: background 0.2s ease, opacity 0.2s ease;
}

/* 鼠标悬停在右侧拉伸条时，呈现精致的主色调光泽 */
.resize-handle-right:hover,
body.is-resizing .resize-handle-right {
  background: var(--primary-color) !important;
  opacity: 0.5 !important;
}
</style>
