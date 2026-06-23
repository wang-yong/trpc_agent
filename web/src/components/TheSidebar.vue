<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'
import { useChatStore } from '@/stores/chat'
import UserMenu from './UserMenu.vue'

const router = useRouter()
const settings = useSettingsStore()
const chat = useChatStore()

const skillCollapsed = ref(false)
const sessionCollapsed = ref(false)

function handleNewTask() {
  chat.newTask()
  settings.currentSkill = null
  router.push('/')
}
</script>

<template>
  <aside class="sidebar">
    <!-- Logo -->
    <div class="sidebar-top">
      <div class="brand-row" @click="router.push('/')" style="cursor: pointer;">
        <div class="logo-box">
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="4" width="18" height="16" rx="3"/>
            <circle cx="9" cy="10" r="1.2"/>
            <circle cx="15" cy="10" r="1.2"/>
            <path d="M9 15c1.5 1 4.5 1 6 0"/>
          </svg>
        </div>
        <span class="brand">AI Agent</span>
        <span class="beta">Beta</span>
      </div>

      <button class="new-task" @click="handleNewTask">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3" stroke-linecap="round">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        新建任务
      </button>
    </div>

    <!-- Skills -->
    <div class="sec">
      <div class="sec-hd" @click="skillCollapsed = !skillCollapsed">
        <svg :class="{ rot: skillCollapsed }" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="6 9 12 15 18 9"/></svg>
        <span>技能</span>
      </div>
      <Transition name="slide">
        <div v-show="!skillCollapsed" class="item-list">
          <div
            v-for="sk in settings.skills"
            :key="sk.id"
            class="item"
            :class="{ on: settings.currentSkill === sk.id }"
            @click="settings.selectSkill(sk.id)"
          >
            <span class="ico">{{ sk.icon }}</span>
            <span class="label">{{ sk.name }}</span>
          </div>
          <div v-if="settings.skills.length === 0" class="empty">暂无技能</div>
        </div>
      </Transition>
    </div>

    <!-- Sessions -->
    <div class="sec grow">
      <div class="sec-hd" @click="sessionCollapsed = !sessionCollapsed">
        <svg :class="{ rot: sessionCollapsed }" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="6 9 12 15 18 9"/></svg>
        <span>任务列表</span>
        <span v-if="chat.sessionCount > 0" class="cnt">{{ chat.sessionCount }}</span>
      </div>
      <Transition name="slide">
        <div v-show="!sessionCollapsed" class="item-list scrollable">
          <div
            v-for="sess in chat.sessions"
            :key="sess.id"
            class="item"
            :class="{ on: chat.currentSessionId === sess.id }"
            @click="chat.selectSession(sess.id)"
          >
            <span class="ico faint">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 11.5a8.38 8.38 0 01-.9 3.8 8.5 8.5 0 01-7.6 4.7 8.38 8.38 0 01-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 01-.9-3.8 8.5 8.5 0 014.7-7.6 8.38 8.38 0 013.8-.9h.5a8.48 8.48 0 018 8v.5z"/></svg>
            </span>
            <span class="label ellipsis" :title="sess.title">{{ sess.title }}</span>
            <button class="x-btn" @click.stop="chat.deleteSession(sess.id)" title="删除">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
          <div v-if="chat.sessions.length === 0" class="empty">暂无对话</div>
        </div>
      </Transition>
    </div>

    <!-- User Menu -->
    <UserMenu />
  </aside>
</template>

<style scoped>
.sidebar {
  height: 100vh; /* 强力、100% 顶天立地贴合屏幕最底端，避免任何父级高度塌陷 */
  display: flex; flex-direction: column;
  /* 侧边栏比主内容略深，形成层次 */
  background: var(--body-color);
}

/* ===== Top Brand ===== */
.sidebar-top { padding: 16px 16px 10px; }
.brand-row {
  display: flex; align-items: center; gap: 9px; padding-bottom: 14px;
}
.logo-box {
  width: 30px; height: 30px; border-radius: 9px;
  background: linear-gradient(135deg, #6b8bf5, #a78bfa);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.brand {
  font-size: 14.5px; font-weight: 700; letter-spacing: -.3px;
}
.beta {
  font-size: 10px; padding: 2px 7px; border-radius: 5px;
  background: rgba(107,139,245,.12); color: var(--primary-color);
  font-weight: 600; letter-spacing: .3px;
}

/* New Task Button */
.new-task {
  display: flex; align-items: center; gap: 8px;
  width: 100%; padding: 10px 14px;
  border-radius: 11px;
  background: linear-gradient(135deg, rgba(107,139,245,.09), rgba(167,139,250,.07));
  border: 1px dashed rgba(107,139,245,.35);
  color: var(--primary-color); font-size: 13px; font-weight: 500;
  cursor: pointer; transition: all .2s; font-family: inherit;
}
.new-task:hover {
  border-style: solid;
  border-color: var(--primary-color);
  background: rgba(107,139,245,.14);
  transform: translateY(-1px);
}

/* ===== Sections ===== */
.sec { margin-top: 6px; }
.sec.grow { flex: 1; min-height: 0; display: flex; flex-direction: column; }

.sec-hd {
  padding: 10px 16px 6px;
  font-size: 11.5px; color: var(--text-color-3); font-weight: 600;
  text-transform: uppercase; letter-spacing: .8px;
  cursor: pointer; user-select: none;
  display: flex; align-items: center; gap: 6px;
  transition: color .15s;
}
.sec-hd:hover { color: var(--text-color-2); }
.sec-hd svg { transition: transform .2s ease; opacity: .6; }
.sec-hd svg.rot { transform: rotate(-90deg); }
.cnt {
  margin-left: auto;
  font-size: 10.5px; font-weight: 600;
  padding: 1px 7px; border-radius: 10px;
  background: var(--hover-color); color: var(--text-color-3);
}

/* Item List */
.item-list { padding: 2px 8px 4px; }
.item-list.scrollable { flex: 1; overflow-y: auto; padding: 2px 8px 8px; }

.item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px; border-radius: 9px;
  font-size: 13px; color: var(--text-color-2);
  cursor: pointer; transition: all .12s; margin-bottom: 1px;
}
.item:hover { background: var(--hover-color); color: var(--text-color); }
.item:hover .x-btn { opacity: 1; }
.item.on { background: rgba(107,139,245,.12); color: var(--primary-color); font-weight: 500; }

.ico { font-size: 14.5px; width: 20px; text-align: center; flex-shrink: 0; }
.ico.faint { opacity: .45; }
.label { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; flex: 1; }
.label.ellipsis { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.x-btn {
  opacity: 0; flex-shrink: 0;
  width: 20px; height: 20px; border-radius: 5px;
  border: none; background: transparent;
  color: var(--text-color-3); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all .12s;
}
.x-btn:hover { background: rgba(248,81,73,.1); color: #f85149; }

.empty { text-align: center; padding: 16px 0 8px; font-size: 12px; color: var(--text-color-3); opacity: .65; }

/* Transition */
.slide-enter-active, .slide-leave-active { transition: all .2s ease; overflow: hidden; }
.slide-enter-from, .slide-leave-to { opacity: 0; max-height: 0; padding-top: 0; padding-bottom: 0; }
</style>
