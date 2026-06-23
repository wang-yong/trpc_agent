<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '@/utils/format'

const props = defineProps<{
  role: 'user' | 'ai'
  content: string
  timestamp?: string
  streaming?: boolean
  error?: boolean
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
}>()

const renderedContent = computed(() => {
  if (props.role === 'user') return props.content
  return renderMarkdown(props.content)
})
</script>

<template>
  <div class="msg" :class="[role, { error }]">
    <div class="msg-avatar">
      <span v-if="role === 'user'" class="avatar-user">我</span>
      <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none">
        <defs>
          <linearGradient id="aiGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" style="stop-color:#5b8def"/>
            <stop offset="100%" style="stop-color:#8b5cf6"/>
          </linearGradient>
        </defs>
        <rect x="3" y="4" width="18" height="16" rx="3.5" stroke="url(#aiGrad)" stroke-width="1.6" fill="none"/>
        <circle cx="9" cy="10" r="1.3" fill="#5b8def"/>
        <circle cx="15" cy="10" r="1.3" fill="#5b8def"/>
        <path d="M9 14.5c1.5 .9 4.5 .9 6 0" stroke="#8b5cf6" stroke-width="1.5" stroke-linecap="round"/>
      </svg>
    </div>

    <div class="msg-body">
      <div class="bubble" :class="{ streaming, error }">
        <!-- User: plain text -->
        <template v-if="role === 'user'">
          <p>{{ content }}</p>
        </template>
        <!-- AI: markdown -->
        <template v-else>
          <div
            v-if="content"
            class="md-body"
            v-html="renderedContent"
          />
          <div v-else-if="streaming" class="typing-cursor"></div>
          <div v-else class="empty-content-tip">(未成功返回内容)</div>
        </template>
      </div>

      <!-- Usage & Time Bar -->
      <div v-if="timestamp || (usage && !error)" class="usage-bar">
        <span v-if="timestamp" class="time-stamp">{{ timestamp }}</span>
        <template v-if="usage && !error">
          <span class="usage-item">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="17 11 12 6 7 11"/><line x1="12" y1="6" x2="12" y2="18"/></svg>
            {{ usage.prompt_tokens }}
          </span>
          <span class="usage-item">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="7 13 12 18 17 13"/><line x1="12" y1="18" x2="12" y2="6"/></svg>
            {{ usage.completion_tokens }}
          </span>
          <span class="usage-item total">合计 {{ usage.total_tokens }}</span>
        </template>
      </div>

      <!-- Error hint -->
      <div v-if="error" class="error-hint">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        响应异常
      </div>
    </div>
  </div>
</template>

<style scoped>
.msg {
  display: flex;
  gap: 12px;
}
.msg.user {
  flex-direction: row-reverse;
}

/* Avatar */
.msg-avatar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.avatar-user {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  background: linear-gradient(135deg, #5b8def, #4a7de0);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Body */
.msg-body {
  max-width: calc(100% - 50px);
  min-width: 0;
}

/* Bubble */
.bubble {
  padding: 12px 16px;
  border-radius: 14px;
  font-size: 14.5px;
  line-height: 1.65;
  word-break: break-word;
  position: relative;
}
.msg.user .bubble {
  background: linear-gradient(135deg, rgba(91,141,239,.08), rgba(91,141,239,.04));
  border: 1px solid rgba(91,141,239,.25);
  border-top-right-radius: 5px;
  text-align: left;
}
.msg.ai .bubble {
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-top-left-radius: 5px;
}
.bubble.streaming {
  /* subtle animation for streaming state */
}
.bubble.error {
  border-color: rgba(248,81,73,.3);
  color: var(--error-color);
}

/* Usage */
.usage-bar {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid rgba(128,128,128,.08);
}
.msg.user .usage-bar {
  justify-content: flex-end;
}
.time-stamp {
  font-size: 11px;
  color: var(--text-color-3);
  opacity: .65;
  user-select: none;
}
.usage-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11.5px;
  color: var(--text-color-3);
  opacity: .8;
}
.usage-item svg {
  opacity: .6;
}
.usage-item.total {
  margin-left: auto;
  color: var(--text-color-2);
  font-weight: 500;
}

/* Error */
.error-hint {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--error-color);
  opacity: .85;
}

/* ===== Markdown 样式精细化排版 (Markdown Typography) ===== */
:deep(.md-body) {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-color);
}
:deep(.md-body p) {
  margin: 0 0 10px 0; /* 段落间距 */
}
:deep(.md-body p:last-child) {
  margin-bottom: 0;
}
:deep(.md-body strong) {
  font-weight: 700;
  color: var(--text-color-1);
}
:deep(.md-body em) {
  font-style: italic;
}
:deep(.md-body code) {
  font-family: "JetBrains Mono", monospace;
  background: var(--hover-color);
  color: var(--primary-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12.5px;
}
:deep(.md-body pre) {
  background: #090a10;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 14px;
  margin: 12px 0;
  overflow-x: auto;
}
:deep(.md-body pre code) {
  background: transparent;
  color: #eceef4;
  padding: 0;
  border-radius: 0;
  font-size: 12px;
  line-height: 1.6;
}
:deep(.md-body ul), :deep(.md-body ol) {
  margin: 6px 0 10px 0;
  padding-left: 20px; /* 列表向右缩进 */
}
:deep(.md-body ul) {
  list-style-type: disc !important; /* 强制展现无序圆点 */
}
:deep(.md-body ol) {
  list-style-type: decimal !important; /* 强制展现有序列表数字 */
}
:deep(.md-body li) {
  margin-bottom: 5px; /* 列表项舒适间距 */
}
:deep(.md-body li:last-child) {
  margin-bottom: 0;
}
:deep(.md-body h1), :deep(.md-body h2), :deep(.md-body h3), 
:deep(.md-body h4), :deep(.md-body h5), :deep(.md-body h6) {
  font-weight: 700;
  color: var(--text-color-1);
  margin: 16px 0 8px 0;
  line-height: 1.4;
}
:deep(.md-body h1) { font-size: 18px; }
:deep(.md-body h2) { font-size: 16px; border-bottom: 1px solid var(--divider-color); padding-bottom: 4px; }
:deep(.md-body h3) { font-size: 15px; }
:deep(.md-body h4) { font-size: 14px; }

/* 引用块样式 */
:deep(.md-body blockquote) {
  margin: 12px 0;
  padding: 8px 16px;
  background: rgba(128,128,128,0.04);
  border-left: 4px solid var(--primary-color);
  border-radius: 0 8px 8px 0;
  color: var(--text-color-2);
}
:deep(.md-body blockquote p) {
  margin-bottom: 0;
}

/* 表格样式 */
:deep(.md-body table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 13.5px;
}
:deep(.md-body th), :deep(.md-body td) {
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  text-align: left;
}
:deep(.md-body th) {
  background: var(--hover-color);
  font-weight: 600;
  color: var(--text-color-1);
}
:deep(.md-body tr:nth-child(even)) {
  background: rgba(128,128,128,0.02);
}

/* ===== 数学公式卡片与标签高奢样式 ===== */
:deep(.math-block-card) {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(139, 92, 246, 0.03) !important;
  border: 1px solid rgba(139, 92, 246, 0.15) !important;
  border-left: 3.5px solid #8b5cf6 !important;
  padding: 10px 14px;
  border-radius: 8px;
  margin: 12px 0;
  font-family: "JetBrains Mono", "Fira Code", Consolas, monospace;
}
:deep(.math-badge) {
  font-size: 10px;
  font-weight: 700;
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.12);
  padding: 2px 6px;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: .5px;
  flex-shrink: 0;
  user-select: none;
}
:deep(.math-content) {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text-color-1);
  word-break: break-all;
}

/* 行内公式 */
:deep(.math-inline-tag) {
  background: rgba(139, 92, 246, 0.05);
  border: 1px solid rgba(139, 92, 246, 0.12);
  color: #8b5cf6;
  padding: 1px 5px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  margin: 0 3px;
  font-weight: 600;
}

.empty-content-tip {
  font-size: 13.5px;
  color: var(--text-color-3);
  font-style: italic;
  opacity: .75;
}
</style>