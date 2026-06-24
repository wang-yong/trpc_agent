import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, type ModelItem, type Skill } from '@/api'

export const useSettingsStore = defineStore('settings', () => {
  const models = ref<ModelItem[]>([])
  const defaultModel = ref('')
  const currentModel = ref('')
  const skills = ref<Skill[]>([])
  const currentSkill = ref<string | null>(null)
  const theme = ref<'dark' | 'light'>(
    (localStorage.getItem('theme') as 'dark' | 'light') || 'dark'
  )

  // 动态左侧侧边栏宽度管理，默认 268px，支持 localStorage 持久化
  const localSidebarWidth = localStorage.getItem('trpc_agent_sidebar_width')
  const sidebarWidth = ref<number>(localSidebarWidth ? parseInt(localSidebarWidth, 10) : 268)

  // 动态打字流速（毫秒/字），默认 20ms
  const typingSpeed = ref(20)

  function updateSidebarWidth(w: number) {
    sidebarWidth.value = w
    localStorage.setItem('trpc_agent_sidebar_width', String(w))
  }

  function updateTypingSpeed(speed: number) {
    typingSpeed.value = speed
  }

  const currentModelDisplay = computed(() => {
    const m = models.value.find(m => m.name === currentModel.value)
    return m?.display_name || 'Unknown'
  })

  const currentSkillData = computed(() => {
    return skills.value.find(s => s.id === currentSkill.value) || null
  })

  async function fetchModels() {
    const data = await api.getModels()
    models.value = data.models || []
    defaultModel.value = data.default
    currentModel.value = data.default
  }

  async function fetchSkills() {
    skills.value = await api.getSkills()
  }

  function selectModel(name: string) {
    currentModel.value = name
  }

  function selectSkill(id: string) {
    currentSkill.value = currentSkill.value === id ? null : id
  }

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
    localStorage.setItem('theme', theme.value)
  }

  return {
    models,
    defaultModel,
    currentModel,
    skills,
    currentSkill,
    theme,
    sidebarWidth,
    typingSpeed,
    currentModelDisplay,
    currentSkillData,
    fetchModels,
    fetchSkills,
    selectModel,
    selectSkill,
    toggleTheme,
    updateSidebarWidth,
    updateTypingSpeed,
  }
})
