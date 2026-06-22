<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'

const router = useRouter()
const route = useRoute()
const settings = useSettingsStore()
const open = ref(false)

function toggle() { open.value = !open.value }
function close() { open.value = false }

function goStats() {
  close()
  router.push('/stats')
}

function goChat() {
  close()
  router.push('/')
}

function toggleTheme() {
  settings.toggleTheme()
  close()
}

const handleOutsideClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.closest('.user-area')) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', handleOutsideClick)
})

onUnmounted(() => {
  document.removeEventListener('click', handleOutsideClick)
})
</script>

<template>
  <div class="user-area" @click.stop="toggle">
    <div class="avatar">U</div>
    <div class="user-info">
      <div class="username">User</div>
      <div class="plan-label">免费版</div>
    </div>
    <svg class="chevron" :class="{ open }" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"/></svg>

    <!-- Popup -->
    <Transition name="popup">
      <div v-if="open" class="popup" @click.stop>
        <!-- 动态切换按钮：如果在统计页，显示“返回对话”；如果在对话页，显示“Token统计” -->
        <button v-if="route.path === '/stats'" class="popup-item" @click="goChat">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <span>返回对话</span>
        </button>
        <button v-else class="popup-item" @click="goStats">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/>
          </svg>
          <span>Token 统计</span>
        </button>
        <div class="popup-divider"></div>
        <button class="popup-item" @click="toggleTheme">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="12" cy="12" r="5"/>
            <line x1="12" y1="1" x2="12" y2="3"/>
            <line x1="12" y1="21" x2="12" y2="23"/>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
            <line x1="1" y1="12" x2="3" y2="12"/>
            <line x1="21" y1="12" x2="23" y2="12"/>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
          </svg>
          <span>{{ settings.theme === 'dark' ? '明亮模式' : '暗色模式' }}</span>
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.user-area {
  margin-top: auto;
  padding: 11px 16px;
  border-top: 1px solid var(--divider-color);
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  position: relative;
  transition: background .12s;
}
.user-area:hover {
  background: var(--hover-color);
}

.avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: linear-gradient(135deg, #5b8def, #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}
.user-info {
  flex: 1;
  min-width: 0;
}
.username {
  font-size: 13px;
  font-weight: 600;
}
.plan-label {
  font-size: 11px;
  color: var(--text-color-3);
}
.chevron {
  flex-shrink: 0;
  color: var(--text-color-3);
  transition: transform .2s ease;
}
.chevron.open {
  transform: rotate(180deg);
}

/* Popup — 必须从背景中明显浮起 */
.popup {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 8px;
  right: 8px;
  background: var(--popover-color);
  border: 1px solid var(--border-color);
  border-radius: 13px;
  padding: 6px;
  /* 强阴影确保弹出层浮起 */
  box-shadow:
    0 12px 40px rgba(0,0,0,.4),
    0 4px 12px rgba(0,0,0,.25),
    0 0 1px rgba(255,255,255,.08) inset;
  z-index: 200;
}
.popup-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 9px;
  font-size: 13px;
  color: var(--text-color-2);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all .12s;
  font-family: inherit;
}
.popup-item:hover {
  background: var(--hover-color);
  color: var(--text-color);
}
.popup-divider {
  height: 1px;
  background: var(--divider-color);
  margin: 4px 8px;
}

/* Transition */
.popup-enter-active,
.popup-leave-active {
  transition: all .18s ease;
}
.popup-enter-from,
.popup-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(.97);
}
</style>