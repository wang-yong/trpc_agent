<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useSettingsStore } from '@/stores/settings'

const props = defineProps<{ busy: boolean }>()
const emit = defineEmits<{ send: [text: string] }>()

const settings = useSettingsStore()
const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement>()
const modelDropdownOpen = ref(false)

function handleSend() {
  const text = inputText.value.trim()
  if (!text || props.busy) return
  emit('send', text)
  inputText.value = ''
  nextTick(() => autoResize())
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 180) + 'px'
}

function toggleModelDropdown(e: Event) {
  e.stopPropagation()
  modelDropdownOpen.value = !modelDropdownOpen.value
}

document.addEventListener('click', () => { modelDropdownOpen.value = false })
</script>

<template>
  <footer class="input-area">
    <div class="input-inner">
      <!-- Input Container — 有明显边框和阴影的卡片 -->
      <div class="input-card">
        <textarea
          ref="textareaRef"
          v-model="inputText"
          rows="1"
          placeholder="输入消息… Enter 发送 · Shift+Enter 换行"
          @keydown="handleKeydown"
          @input="autoResize"
        />
        <div class="toolbar">
          <!-- Model Selector -->
          <div class="model-picker" :class="{ open: modelDropdownOpen }" @click="toggleModelDropdown">
            <span class="dot"></span>
            <span class="m-name">{{ settings.currentModelDisplay }}</span>
            <span class="arr">▾</span>
            <Transition name="drop">
              <div v-if="modelDropdownOpen" class="dd-menu" @click.stop>
                <div
                  v-for="m in settings.models"
                  :key="m.name"
                  class="dd-item"
                  :class="{ on: m.name === settings.currentModel }"
                  @click.stop="settings.selectModel(m.name); modelDropdownOpen = false"
                >
                  <span class="dot"></span>
                  <span>{{ m.display_name }}</span>
                  <span v-if="m.name === settings.currentModel" class="chk">✓</span>
                </div>
              </div>
            </Transition>
          </div>

          <!-- Send Button -->
          <button class="send" @click="handleSend" :disabled="busy || !inputText.trim()">
            <svg width="17" height="17" viewBox="0 0 24 24" fill="currentColor">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Skill Pills -->
      <div v-if="settings.skills.length > 0" class="pills">
        <button
          v-for="sk in settings.skills"
          :key="sk.id"
          class="pill"
          :class="{ on: settings.currentSkill === sk.id }"
          @click="settings.selectSkill(sk.id)"
        >
          <span>{{ sk.icon }} {{ sk.name }}</span>
        </button>
      </div>

      <p class="note">AI 生成内容仅供参考 · 当前模型: {{ settings.currentModelDisplay }}</p>
    </div>
  </footer>
</template>

<style scoped>
.input-area {
  flex-shrink: 0;
  padding: 10px 24px 18px;
}
.input-inner { max-width: 800px; margin: 0 auto; }

/* ===== Input Card — 核心视觉焦点 ===== */
.input-card {
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  padding: 12px 16px;
  transition: all .2s ease;
  box-shadow: 0 2px 12px rgba(0,0,0,.06), 0 0 0 .5px rgba(128,130,145,.08);
}
.input-card:focus-within {
  border-color: var(--primary-color);
  box-shadow:
    0 4px 24px rgba(0,0,0,.1),
    0 0 0 3px rgba(107,139,245,.12),
    0 0 0 .5px rgba(107,139,245,.25);
}

textarea {
  width: 100%; background: transparent; border: none; outline: none;
  color: var(--text-color); font-size: 14.5px; line-height: 1.58;
  resize: none; max-height: 180px; padding: 4px 0 8px;
  font-family: inherit; min-height: 26px;
}
textarea::placeholder { color: var(--text-color-3); }

/* Toolbar */
.toolbar {
  display: flex; align-items: center; justify-content: space-between; gap: 6px; padding-top: 4px;
}

/* Model Picker */
.model-picker {
  display: flex; align-items: center; gap: 5px;
  padding: 5px 10px; border-radius: 8px;
  background: transparent; cursor: pointer; position: relative;
  transition: background .15s; font-size: 12px; color: var(--text-color-3);
}
.model-picker:hover { background: var(--hover-color); color: var(--text-color-2); }
.dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: #22c55e; box-shadow: 0 0 5px rgba(34,197,94,.45); flex-shrink: 0;
}
.m-name { white-space: nowrap; max-width: 110px; overflow: hidden; text-overflow: ellipsis; }
.arr { font-size: 11px; transition: transform .2s; }
.model-picker.open .arr { transform: rotate(180deg); }

/* Dropdown — 必须从背景中明显浮起 */
.dd-menu {
  position: absolute; bottom: calc(100% + 6px); left: -4px;
  min-width: 210px;
  background: var(--popover-color);
  border: 1px solid var(--border-color);
  border-radius: 12px; padding: 5px;
  /* 强阴影确保弹出层浮起 */
  box-shadow:
    0 12px 40px rgba(0,0,0,.4),
    0 4px 12px rgba(0,0,0,.25),
    0 0 1px rgba(255,255,255,.08) inset;
  z-index: 100;
}
.dd-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 11px; border-radius: 8px;
  font-size: 13px; color: var(--text-color-2);
  cursor: pointer; transition: all .12s;
}
.dd-item:hover { background: var(--hover-color); color: var(--text-color); }
.dd-item.on { color: var(--primary-color); background: rgba(107,139,245,.08); }
.chk { margin-left: auto; font-size: 14px; font-weight: 600; }

/* Send Button */
.send {
  width: 34px; height: 34px; border-radius: 10px;
  background: linear-gradient(135deg, #6b8bf5, #7b98f6); border: none;
  color: #fff; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all .18s; flex-shrink: 0;
}
.send:hover:not(:disabled) {
  background: linear-gradient(135deg, #7b98f6, #8aa8f8);
  transform: scale(1.06);
  box-shadow: 0 3px 14px rgba(107,139,245,.35);
}
.send:disabled { opacity: .35; cursor: not-allowed; transform: none; }

/* Skill Pills */
.pills { display: flex; gap: 7px; margin-top: 10px; flex-wrap: wrap; padding: 0 2px; }
.pill {
  padding: 5px 13px; border-radius: 20px; font-size: 12px;
  background: transparent; border: 1px solid var(--divider-color);
  color: var(--text-color-3); cursor: pointer; transition: all .15s; font-family: inherit;
}
.pill:hover { border-color: var(--primary-color-hover); color: var(--text-color-2); }
.pill.on { border-color: var(--primary-color); color: var(--primary-color); background: rgba(107,139,245,.08); }

/* Disclaimer */
.note {
  text-align: center; font-size: 11px; color: var(--text-color-3);
  opacity: .6; margin-top: 10px; letter-spacing: .2px;
}

/* Transitions */
.drop-enter-active, .drop-leave-active { transition: all .15s ease; }
.drop-enter-from, .drop-leave-to { opacity: 0; transform: translateY(4px) scale(.97); }

@media (max-width: 768px) {
  .input-area { padding-left: 14px; padding-right: 14px; }
  .m-name { max-width: 80px; }
}
</style>
