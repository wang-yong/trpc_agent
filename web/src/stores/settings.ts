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
    currentModelDisplay,
    currentSkillData,
    fetchModels,
    fetchSkills,
    selectModel,
    selectSkill,
    toggleTheme,
  }
})
