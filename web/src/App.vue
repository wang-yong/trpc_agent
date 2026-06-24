<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import {
  NConfigProvider,
  NMessageProvider,
  NDialogProvider,
  NLayout,
  NLayoutSider,
  NLayoutContent,
  darkTheme,
  zhCN,
  dateZhCN,
  type GlobalThemeOverrides,
} from 'naive-ui'
import { useSettingsStore } from '@/stores/settings'
import { useChatStore } from '@/stores/chat'
import TheSidebar from '@/components/TheSidebar.vue'

const settings = useSettingsStore()
const chat = useChatStore()

const theme = computed(() => (settings.theme === 'dark' ? darkTheme : null))

// ====== 极致亮色主题 ======
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#4f46e5', // 科技靛蓝色
    primaryColorHover: '#6366f1',
    primaryColorPressed: '#4338ca',
    borderRadius: '12px',
    borderColor: '#e4e4e7', // 高对比度灰
    dividerColor: '#f4f4f5',
    hoverColor: 'rgba(79, 70, 229, 0.05)',
    cardColor: '#ffffff',
    bodyColor: '#fafafa',
    popoverColor: '#ffffff',
    textColor1: '#18181b',
    textColor2: '#4f4f56',
    textColor3: '#71717a',
  },
  Layout: {
    siderColor: '#ffffff',
    siderBorderColor: '#e4e4e7',
    color: '#fafafa',
  }
}

// ====== 极致暗色主题 (黑曜蓝调) ======
const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#5b8def', // 明亮科技蓝
    primaryColorHover: '#79a6fb',
    primaryColorPressed: '#4472d4',
    borderRadius: '12px',
    borderColor: '#333746', // 清晰深灰蓝边框
    dividerColor: '#252936',
    hoverColor: 'rgba(91, 141, 239, 0.08)',
    cardColor: '#151724', // 深邃蓝黑
    bodyColor: '#0f111a', // 黑曜暗背景
    popoverColor: '#1e2132', // 弹出层明亮一些，浮起来
    textColor1: '#f1f3f9',
    textColor2: '#b2b8cc',
    textColor3: '#6e758a',
  },
  Layout: {
    siderColor: '#121420', // 侧边栏更深色
    siderBorderColor: '#252936',
    color: '#0f111a',
  }
}

const activeThemeOverrides = computed(() =>
  settings.theme === 'dark' ? darkThemeOverrides : themeOverrides
)

// 动态将 Naive UI 的主题变量反射到 documentElement CSS 变量中，使得门户、Markdown、自定义 CSS 都可以获取统一的主题变量！
watch(() => settings.theme, (newTheme) => {
  const overrides = newTheme === 'dark' ? darkThemeOverrides : themeOverrides
  const common = overrides.common || {}
  
  document.documentElement.setAttribute('data-theme', newTheme)
  if (newTheme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  
  const variables = {
    '--primary-color': common.primaryColor,
    '--primary-color-hover': common.primaryColorHover,
    '--primary-color-pressed': common.primaryColorPressed,
    '--border-color': common.borderColor,
    '--divider-color': common.dividerColor,
    '--hover-color': common.hoverColor,
    '--card-color': common.cardColor,
    '--body-color': common.bodyColor,
    '--popover-color': common.popoverColor,
    '--text-color': common.textColor1,
    '--text-color-1': common.textColor1,
    '--text-color-2': common.textColor2,
    '--text-color-3': common.textColor3,
    '--border-radius': common.borderRadius,
  }
  
  for (const [key, val] of Object.entries(variables)) {
    if (val) {
      document.documentElement.style.setProperty(key, val)
    }
  }
}, { immediate: true })

function initLeftResize(e: MouseEvent) {
  e.preventDefault()
  const startX = e.clientX
  const startWidth = settings.sidebarWidth

  document.body.classList.add('is-resizing')

  const handleMouseMove = (moveEvent: MouseEvent) => {
    const diffX = moveEvent.clientX - startX
    const newWidth = startWidth + diffX
    // 彻底解禁拉伸限制！只保留 20px 极简物理安全防负值崩溃兜底
    settings.updateSidebarWidth(Math.max(20, newWidth))
  }

  const handleMouseUp = () => {
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
    document.body.classList.remove('is-resizing')
  }

  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
}

onMounted(async () => {
  try {
    await Promise.all([settings.fetchModels(), settings.fetchSkills()])
    await chat.fetchSessions()
  } catch (err) {
    console.error('初始化失败:', err)
  }
})
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="activeThemeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <NMessageProvider>
      <NDialogProvider>
        <NLayout has-sider class="app-layout">
          <NLayoutSider
            v-if="!chat.sidebarCollapsed"
            bordered
            :width="settings.sidebarWidth"
            :native-scrollbar="false"
            collapse-mode="width"
            class="app-sider"
          >
            <TheSidebar />
          </NLayoutSider>
          <!-- 左侧面板原生拖拽手柄条 -->
          <div v-if="!chat.sidebarCollapsed" class="resize-handle-left" @mousedown="initLeftResize"></div>

          <NLayoutContent class="app-content">
            <RouterView />
          </NLayoutContent>
        </NLayout>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
.app-layout { height: 100vh; background: var(--body-color); }
.app-sider { height: 100vh; background: var(--body-color); border-right: 1px solid var(--border-color); position: relative; }
.app-sider :deep(.n-layout-sider-scroll-container) {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.app-content {
  height: 100vh;
  display: flex; flex-direction: column;
  background: var(--body-color);
}

/* ===== 左侧 Sider 拖拽手柄原生实现 ===== */
.resize-handle-left {
  width: 6px;
  background: transparent;
  cursor: col-resize !important;
  z-index: 100;
  position: relative;
  margin-left: -3px; /* 完美重合 Sider 与 Content 的分界线 */
  margin-right: -3px;
  flex-shrink: 0;
  transition: background 0.2s ease, opacity 0.2s ease;
}
.resize-handle-left:hover,
body.is-resizing .resize-handle-left {
  background: var(--primary-color) !important;
  opacity: 0.5 !important;
}
</style>
