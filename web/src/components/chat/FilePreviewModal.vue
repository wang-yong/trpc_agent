<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { api, type FilePreviewData } from '@/api'
import { renderMarkdown } from '@/utils/format'

const visible = ref(false)
const filePath = ref('')
const loading = ref(false)
const previewData = ref<FilePreviewData | null>(null)

// 缩放级别与全屏
const zoom = ref(1.0)
const isFullscreen = ref(false)

// 判断文件类别
const fileExt = computed(() => {
  if (!previewData.value) return ''
  return previewData.value.extension.toLowerCase()
})

const isImage = computed(() => {
  const imgExts = ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.ico']
  return imgExts.includes(fileExt.value)
})

const isPdf = computed(() => {
  return fileExt.value === '.pdf'
})

const isMarkdown = computed(() => {
  return fileExt.value === '.md'
})

// 格式化文件大小
function formatBytes(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化时间戳
function formatTime(timestamp: number) {
  const d = new Date(timestamp * 1000)
  return d.toLocaleString('zh-CN', { hour12: false })
}

// 直接获取图片的 Raw API 链接
const rawUrl = computed(() => {
  if (!filePath.value) return ''
  return api.getFileRawUrl(filePath.value)
})

// 加载文件预览信息
async function loadPreview(path: string) {
  filePath.value = path
  loading.value = true
  visible.value = true
  previewData.value = null
  zoom.value = 1.0 // 重置缩放
  
  try {
    const data = await api.getFilePreview(path)
    previewData.value = data
  } catch (err) {
    console.error('获取预览失败:', err)
  } finally {
    loading.value = false
  }
}

function close() {
  visible.value = false
  isFullscreen.value = false
}

function handleZoom(amount: number) {
  zoom.value = Math.max(0.5, Math.min(zoom.value + amount, 3.0))
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

// 捕获全局文件预览事件
function handlePreviewEvent(e: Event) {
  const customEvent = e as CustomEvent<string>
  if (customEvent.detail) {
    loadPreview(customEvent.detail)
  }
}

// 支持键盘 Esc 键一键退出
function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && visible.value) {
    close()
  }
}

onMounted(() => {
  window.addEventListener('preview-file', handlePreviewEvent)
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('preview-file', handlePreviewEvent)
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <Transition name="preview-fade">
    <div v-if="visible" class="preview-backdrop" @click.self="close">
      <div 
        class="preview-modal-card" 
        :class="{ 'is-fullscreen': isFullscreen }"
        @click.stop
      >
        <!-- Card Header -->
        <header class="preview-header">
          <div class="header-meta-group">
            <span class="file-icon-badge">
              <template v-if="isImage">🖼️</template>
              <template v-else-if="isPdf">📕</template>
              <template v-else-if="isMarkdown">📝</template>
              <template v-else>📁</template>
            </span>
            <div class="meta-title-col">
              <h3 class="preview-filename" :title="filePath">{{ previewData?.name || filePath }}</h3>
              <p class="preview-filepath">{{ filePath }}</p>
            </div>
          </div>

          <!-- Controls Panel -->
          <div class="header-controls">
            <!-- 缩放控制（如果是图片或可读文本，显示缩放） -->
            <div v-if="previewData && !previewData.is_binary && !isPdf" class="zoom-controls">
              <button class="ctrl-btn" @click="handleZoom(-0.1)" title="缩小">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="5" y1="12" x2="19" y2="12"/></svg>
              </button>
              <span class="zoom-pct">{{ Math.round(zoom * 100) }}%</span>
              <button class="ctrl-btn" @click="handleZoom(0.1)" title="放大">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              </button>
            </div>

            <!-- 全屏按钮 -->
            <button class="ctrl-btn main-action" @click="toggleFullscreen" :title="isFullscreen ? '退出全屏' : '全屏预览'">
              <svg v-if="!isFullscreen" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/></svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M4 14h6v6m10-6h-6v6M4 10h6V4m10 6h-6V4"/></svg>
            </button>

            <!-- 关闭按钮 -->
            <button class="ctrl-btn close-action" @click="close" title="关闭 (Esc)">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
        </header>

        <!-- Card Body -->
        <div class="preview-body">
          <!-- Loading State -->
          <div v-if="loading" class="preview-loading-box">
            <div class="spinner-circle"></div>
            <span class="loading-label">正在读取物理盘数据...</span>
          </div>

          <!-- Content Render Section -->
          <div v-else-if="previewData" class="render-content-wrap">
            
            <!-- 大文件截断安全提示 -->
            <div v-if="previewData.is_truncated" class="trunc-warning-bar">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#d97706" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              <span>⚙️ 该文本文件体积过大，当前已自动开启大文件防卡死防护，仅读取并预览前 150KB 字节。</span>
            </div>

            <!-- Case 1: 图片渲染 -->
            <div v-if="isImage" class="view-viewport image-viewport">
              <img 
                :src="rawUrl" 
                alt="Image Preview" 
                class="preview-img-tag"
                :style="{ transform: `scale(${zoom})` }"
              />
            </div>

            <!-- Case 2: PDF 渲染 -->
            <div v-else-if="isPdf" class="view-viewport pdf-viewport">
              <iframe :src="rawUrl" class="pdf-iframe-tag"></iframe>
            </div>

            <!-- Case 3: Markdown 渲染 -->
            <div v-else-if="isMarkdown" class="view-viewport md-viewport" :style="{ fontSize: `${zoom * 14}px` }">
              <div class="md-body" v-html="renderMarkdown(previewData.content)"></div>
            </div>

            <!-- Case 4: 二进制不可读文件 -->
            <div v-else-if="previewData.is_binary" class="view-viewport binary-viewport">
              <div class="binary-meta-card">
                <div class="binary-icon">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                </div>
                <h4 class="binary-title">不可直读的二进制文件</h4>
                <p class="binary-desc">此文件不包含纯文本编码，为了电脑的物理安全，不推荐强行转译预览。</p>
                <div class="binary-meta-table">
                  <div class="meta-row"><span class="label">文件后缀：</span><span class="val">{{ previewData.extension || '无' }}</span></div>
                  <div class="meta-row"><span class="label">文件大小：</span><span class="val">{{ formatBytes(previewData.size) }}</span></div>
                  <div class="meta-row"><span class="label">最后修改：</span><span class="val">{{ formatTime(previewData.mod_time) }}</span></div>
                </div>
              </div>
            </div>

            <!-- Case 5: 默认普通纯文本代码 -->
            <div v-else class="view-viewport text-viewport" :style="{ fontSize: `${zoom * 13}px` }">
              <pre class="code-terminal-pre"><code>{{ previewData.content }}</code></pre>
            </div>

          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
/* ===== Backdrop Blur ===== */
.preview-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 17, 26, 0.65); /* 暗曜蓝背景融入 */
  backdrop-filter: blur(12px); /* 高级磨砂玻璃 */
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000; /* 置于整站最顶层 */
}

/* ===== Card Layout ===== */
.preview-modal-card {
  width: 820px;
  height: 85%;
  background: var(--popover-color);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  box-shadow: var(--shadow-3);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}
.preview-modal-card.is-fullscreen {
  width: 100% !important;
  height: 100% !important;
  border-radius: 0;
  border: none;
}

/* ===== Header ===== */
.preview-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(128,128,128,0.01);
  flex-shrink: 0;
}
.header-meta-group {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.file-icon-badge {
  font-size: 20px;
  width: 38px; height: 38px;
  border-radius: 10px;
  background: var(--hover-color);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.meta-title-col {
  min-width: 0;
}
.preview-filename {
  font-size: 14.5px;
  font-weight: 700;
  color: var(--text-color-1);
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.preview-filepath {
  font-size: 11px;
  color: var(--text-color-3);
  margin: 0;
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ===== Controls ===== */
.header-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.zoom-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--hover-color);
  padding: 3px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
}
.zoom-pct {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-color-2);
  min-width: 38px;
  text-align: center;
  font-family: monospace;
}
.ctrl-btn {
  width: 26px; height: 26px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: var(--text-color-3);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all .15s;
}
.ctrl-btn:hover {
  background: var(--hover-color);
  color: var(--text-color-1);
}
.ctrl-btn.close-action:hover {
  background: rgba(248, 81, 73, 0.12);
  color: #f85149;
}

/* ===== Body ===== */
.preview-body {
  flex: 1;
  overflow: hidden;
  position: relative;
  background: rgba(128,128,128,0.005);
}
.preview-loading-box {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}
.spinner-circle {
  width: 28px; height: 28px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin-ani 1s infinite linear;
}
.loading-label {
  font-size: 12.5px;
  color: var(--text-color-3);
}

.render-content-wrap {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* ===== Viewport Types ===== */
.view-viewport {
  flex: 1;
  overflow: auto;
  padding: 20px;
}

.image-viewport {
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  background: radial-gradient(circle, rgba(128,128,128,0.03) 0%, transparent 80%);
}
.preview-img-tag {
  max-width: 90%;
  max-height: 90%;
  border-radius: 8px;
  box-shadow: 0 12px 32px rgba(0,0,0,0.15);
  transition: transform 0.1s ease;
  user-select: none;
}

.pdf-viewport {
  padding: 0;
  width: 100%; height: 100%;
  overflow: hidden;
}
.pdf-iframe-tag {
  width: 100%; height: 100%;
  border: none;
}

/* ===== Markdown & Code Viewports ===== */
.md-viewport {
  background: var(--card-color);
  line-height: 1.7;
}

.text-viewport {
  background: #090a10; /* 深曜黑极客编辑器底色 */
  padding: 16px 20px;
  color: #eceef4;
}
.code-terminal-pre {
  margin: 0;
  font-family: "JetBrains Mono", Consolas, monospace;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

/* ===== Binary View ===== */
.binary-viewport {
  display: flex;
  align-items: center;
  justify-content: center;
}
.binary-meta-card {
  width: 360px;
  padding: 24px;
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  text-align: center;
  box-shadow: var(--shadow-1);
}
.binary-icon {
  color: var(--text-color-3);
  opacity: .35;
  margin-bottom: 12px;
}
.binary-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-color-1);
  margin: 0 0 6px 0;
}
.binary-desc {
  font-size: 11.5px;
  color: var(--text-color-3);
  line-height: 1.5;
  margin: 0 0 16px 0;
}
.binary-meta-table {
  border-top: 1px solid var(--border-color);
  padding-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.meta-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}
.meta-row .label { color: var(--text-color-3); }
.meta-row .val { color: var(--text-color-2); font-family: monospace; font-weight: 600; }

/* ===== Warning Bar ===== */
.trunc-warning-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(245, 158, 11, 0.06);
  border-bottom: 1px solid rgba(245, 158, 11, 0.15);
  padding: 8px 16px;
  font-size: 12px;
  color: #d97706;
  font-weight: 600;
  flex-shrink: 0;
}

/* ===== Transitions ===== */
.preview-fade-enter-active, .preview-fade-leave-active {
  transition: all .25s ease;
}
.preview-fade-enter-from, .preview-fade-leave-to {
  opacity: 0;
}
.preview-fade-enter-from .preview-modal-card {
  transform: scale(0.96);
  opacity: 0;
}
.preview-fade-leave-to .preview-modal-card {
  transform: scale(0.96);
  opacity: 0;
}

@keyframes spin-ani {
  to { transform: rotate(360deg); }
}
</style>
