<script setup lang="ts">
import { ref, computed } from 'vue'
import type { ThinkingStep } from '@/stores/chat'

const props = defineProps<{
  steps?: ThinkingStep[]
}>()

const expanded = ref(true)

const activeSteps = computed(() => props.steps || [])
const hasSteps = computed(() => activeSteps.value.length > 0)

// 汇总当前运行状态
const statusSummary = computed(() => {
  if (!hasSteps.value) return '准备中'
  const running = activeSteps.value.some(s => s.status === 'thinking' || s.status === 'running')
  if (running) {
    const activeTool = activeSteps.value.find(s => s.status === 'running')
    if (activeTool) return `正在执行工具: ${activeTool.toolName}`
    return '正在整理思路...'
  }
  const toolCount = activeSteps.value.filter(s => s.type === 'tool').length
  if (toolCount > 0) return `已调度并完成 ${toolCount} 个工具调用`
  return '已完成深度思考'
})

function formatJson(val?: string) {
  if (!val) return ''
  try {
    let cleanVal = val.trim()
    // 修复第三方平台可能返回的双大括号损坏问题
    if (cleanVal.startsWith('{{') && cleanVal.endsWith('}}') && !cleanVal.startsWith('{{{')) {
      cleanVal = cleanVal.slice(1, -1)
    }
    const parsed = JSON.parse(cleanVal)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return val
  }
}
</script>

<template>
  <div v-if="hasSteps" class="thinking-chain" :class="{ open: expanded }">
    <!-- 折叠栏头部 -->
    <header class="chain-header" @click="expanded = !expanded">
      <div class="header-left">
        <!-- 呼吸光环状态 -->
        <span class="status-indicator" :class="{ pulsing: steps?.some(s => s.status === 'thinking' || s.status === 'running') }">
          <svg class="spinner-svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10"/>
          </svg>
        </span>
        <span class="summary-text">{{ statusSummary }}</span>
      </div>
      <div class="header-right">
        <span class="steps-count">共 {{ activeSteps.length }} 步</span>
        <svg class="arrow" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="6 9 12 15 18 9"/></svg>
      </div>
    </header>

    <!-- 步骤详细内容 -->
    <Transition name="expand">
      <div v-show="expanded" class="chain-body">
        <div v-for="(step, idx) in activeSteps" :key="idx" class="step-item" :class="[step.type, step.status]">
          <!-- 时间线连接线 -->
          <div class="step-line" v-if="idx < activeSteps.length - 1"></div>

          <!-- 步骤图标 -->
          <div class="step-icon">
            <template v-if="step.type === 'thought'">
              <!-- 思想灯泡 -->
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A5 5 0 0 0 8 8c0 1 .5 2.5 1.5 3.5.7.8 1.3 1.5 1.5 2.5M9 18h6M10 22h4"/></svg>
            </template>
            <template v-else>
              <!-- 齿轮/工具 -->
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>
            </template>
          </div>

          <!-- 步骤文本 -->
          <div class="step-content">
            <h4 class="step-title">
              <template v-if="step.type === 'thought'">思考</template>
              <template v-else>调用工具: <code>{{ step.toolName }}</code></template>
              <span class="step-badge" :class="step.status">
                {{ step.status === 'thinking' ? '思考中' : step.status === 'running' ? '运行中' : '执行成功' }}
              </span>
            </h4>

            <div class="step-detail">
              <!-- 思考文本 -->
              <p v-if="step.type === 'thought'" class="thought-text">{{ step.content }}</p>

              <!-- 工具调用参数与结果 -->
              <div v-else class="tool-io">
                <!-- 输入参数 -->
                <div v-if="step.args" class="io-block args">
                  <span class="block-label">输入参数 (Arguments)</span>
                  <pre><code>{{ formatJson(step.args) }}</code></pre>
                </div>
                <!-- 物理运行结果 -->
                <div v-if="step.content" class="io-block observation">
                  <span class="block-label">运行结果 (Observation)</span>
                  <pre class="term"><code>{{ formatJson(step.content) }}</code></pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.thinking-chain {
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius, 12px);
  overflow: hidden;
  margin-bottom: 12px;
  transition: all .25s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,.04);
}
.thinking-chain.open {
  box-shadow: 0 4px 16px rgba(0,0,0,.06), 0 0 0 .5px rgba(107,139,245,.05);
  border-color: var(--border-color);
}

/* ===== Header ===== */
.chain-header {
  padding: 10px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  user-select: none;
  background: var(--hover-color);
  transition: background .15s;
}
.chain-header:hover {
  background: rgba(128,128,128,.05);
}
.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.status-indicator {
  display: flex; align-items: center; justify-content: center;
  width: 22px; height: 22px; border-radius: 50%;
  background: rgba(91,141,239,.1); color: var(--primary-color);
}
.status-indicator.pulsing .spinner-svg {
  animation: spin 2.5s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.summary-text {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text-color-2);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.steps-count {
  font-size: 11px;
  color: var(--text-color-3);
  font-weight: 500;
}
.arrow {
  color: var(--text-color-3);
  transition: transform .2s ease;
  opacity: .7;
}
.thinking-chain.open .arrow {
  transform: rotate(180deg);
  color: var(--primary-color);
  opacity: 1;
}

/* ===== Body ===== */
.chain-body {
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 22px;
  border-top: 1px solid var(--divider-color);
  background: var(--card-color);
  position: relative;
}

/* ===== Step Item ===== */
.step-item {
  display: flex;
  gap: 16px;
  position: relative;
}
.step-line {
  position: absolute;
  top: 26px; left: 12px; bottom: -28px;
  width: 1px;
  background: var(--divider-color);
  z-index: 1;
}

/* Icon */
.step-icon {
  width: 25px; height: 25px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  background: var(--body-color);
  border: 1px solid var(--border-color);
  color: var(--text-color-3);
  z-index: 2;
  flex-shrink: 0;
  transition: all .2s;
}
.step-item.thinking .step-icon {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: rgba(91,141,239,.08);
}
.step-item.running .step-icon {
  border-color: #f59e0b;
  color: #f59e0b;
  background: rgba(245,158,11,.08);
}
.step-item.success .step-icon {
  border-color: #10b981;
  color: #10b981;
  background: rgba(16,185,129,.08);
}

/* Content */
.step-content {
  flex: 1;
  min-width: 0;
}
.step-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color-1);
  margin: 0 0 6px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.step-title code {
  font-family: monospace;
  background: var(--body-color);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--primary-color);
}
.step-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 500;
}
.step-badge.thinking { background: rgba(91,141,239,.1); color: var(--primary-color); }
.step-badge.running { background: rgba(245,158,11,.1); color: #f59e0b; }
.step-badge.success { background: rgba(16,185,129,.1); color: #10b981; }

/* Details */
.thought-text {
  font-size: 12.5px;
  color: var(--text-color-2);
  line-height: 1.55;
  margin: 0;
  white-space: pre-wrap;
  background: var(--body-color);
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid var(--divider-color);
}

/* Tool IO Block */
.tool-io {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.io-block {
  background: var(--body-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 10px 14px;
}
.block-label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-color-3);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: .5px;
}
pre {
  margin: 0;
  overflow-x: auto;
}
code {
  font-family: "JetBrains Mono", "Fira Code", Consolas, monospace;
  font-size: 11.5px;
  line-height: 1.5;
  color: var(--text-color-2);
}
.observation pre.term {
  border-left: 2.5px solid #10b981;
  padding-left: 8px;
}

/* Expand Transition */
.expand-enter-active, .expand-leave-active { transition: all .25s ease-out; max-height: 800px; overflow: hidden; }
.expand-enter-from, .expand-leave-to { max-height: 0; opacity: 0; }
</style>
