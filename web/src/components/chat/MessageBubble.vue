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
          <div v-else class="typing-cursor"></div>
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
</style>