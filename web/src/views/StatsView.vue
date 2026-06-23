<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useChatStore } from '@/stores/chat'
import { api, type TokenStats } from '@/api'
import { fmtNum, fmtCost, fmtDateTime, shortModelName, calcCost } from '@/utils/format'

const chat = useChatStore()

const data = ref<TokenStats | null>(null)
const loading = ref(false)
const currentTab = ref<'daily' | 'product' | 'model'>('daily' as const)
const pageSize = ref(25)
const currentPage = ref(1)
const sortField = ref<string>('timestamp')
const sortDir = ref<'asc' | 'desc'>('desc')

// ====== 按天（每日维度）统计 Token 消耗 ======
const dailyStats = computed(() => {
  const map: Record<string, { date: string, count: number, prompt: number, completion: number, total: number, cost: number }> = {}

  recent.value.forEach(r => {
    // 转换为 YYYY-MM-DD
    const d = new Date(r.timestamp * 1000)
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const dateStr = String(d.getDate()).padStart(2, '0')
    const key = `${year}-${month}-${dateStr}`

    const cost = calcCost(r.model, r.prompt_tokens, r.completion_tokens)

    if (!map[key]) {
      map[key] = {
        date: key,
        count: 0,
        prompt: 0,
        completion: 0,
        total: 0,
        cost: 0
      }
    }

    const item = map[key]
    item.count++
    item.prompt += r.prompt_tokens
    item.completion += r.completion_tokens
    item.total += r.total_tokens
    item.cost += cost
  })

  // 转换成列表并按日期降序
  return Object.values(map).sort((a, b) => b.date.localeCompare(a.date))
})

const summary = computed(() => data.value?.summary || { total_requests: 0, total_prompt: 0, total_completion: 0, total_tokens: 0 })
const byModel = computed(() => data.value?.by_model || [])
const recent = computed(() => data.value?.recent || [])

const totalCost = computed(() => {
  return byModel.value.reduce((sum, m) => sum + calcCost(m.model, m.prompt_tokens, m.completion_tokens), 0)
})

const QUOTA_TOKENS = 1000000
const quotaYuan = computed(() => summary.value.total_tokens / 1000000 * 0.5)
const usedPct = computed(() => Math.min((summary.value.total_tokens / QUOTA_TOKENS) * 100, 100))

const sortedRecords = computed(() => {
  let records = recent.value ? [...recent.value] : []
  records.sort((a, b) => {
    let va: any = a[sortField.value as keyof typeof a]
    let vb: any = b[sortField.value as keyof typeof b]
    if (sortField.value === 'cost') {
      va = calcCost(a.model, a.prompt_tokens, a.completion_tokens)
      vb = calcCost(b.model, b.prompt_tokens, b.completion_tokens)
    }
    if (typeof va === 'string') va = va.toLowerCase()
    if (typeof vb === 'string') vb = vb.toLowerCase()
    if (sortDir.value === 'asc') return va > vb ? 1 : va < vb ? -1 : 0
    return va < vb ? 1 : va > vb ? -1 : 0
  })
  return records
})

const totalPages = computed(() => Math.max(1, Math.ceil(sortedRecords.value.length / pageSize.value)))
const pageRecords = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return sortedRecords.value.slice(start, start + pageSize.value)
})

function sortDetail(field: string) {
  if (sortField.value === field) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = field
    sortDir.value = field === 'timestamp' ? 'desc' : 'asc'
  }
  currentPage.value = 1
}

function switchTab(tab: 'product' | 'model') {
  currentTab.value = tab
}

async function loadStats() {
  loading.value = true
  try {
    data.value = await api.getTokenStats()
  } catch (err) {
    console.error('加载失败:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => loadStats())
</script>

<template>
  <div class="stats-view">
    <!-- Header -->
    <div class="stats-header">
      <button class="icon-btn" @click="chat.toggleSidebar()" title="切换侧栏">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="3" y1="6" x2="21" y2="6"/>
          <line x1="3" y1="12" x2="21" y2="12"/>
          <line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
      </button>
      <span class="title">Token 消耗统计</span>
      <button class="refresh-btn" @click="loadStats()" :disabled="loading">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10"/></svg>
        刷新
      </button>
    </div>

    <div class="stats-body">
      <!-- Quota Card -->
      <div class="quota-card">
        <div class="quota-header">
          <div class="icon-box">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v16"/></svg>
          </div>
          <span class="quota-title">Token 用量概览</span>
        </div>
        <div class="quota-row">
          <span class="quota-used">{{ fmtCost(quotaYuan) }}</span>
          <span class="quota-sep">/</span>
          <span class="quota-total">&yen; 500.00</span>
          <span class="quota-pct-wrap">已用 <b>{{ usedPct.toFixed(1) }}%</b></span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: usedPct + '%' }"></div>
        </div>
      </div>

      <!-- Summary Section -->
      <div class="section-card">
        <div class="section-head">
          <h3 class="section-title">使用汇总</h3>
          <div class="section-tabs">
            <button :class="{ active: currentTab === 'daily' }" @click="switchTab('daily')">每日消耗</button>
            <button :class="{ active: currentTab === 'product' }" @click="switchTab('product')">产品使用</button>
            <button :class="{ active: currentTab === 'model' }" @click="switchTab('model')">模型使用</button>
          </div>
        </div>
        <div class="section-stats">
          <span>请求数 <b>{{ summary.total_requests }}</b></span>
          <span>输入 Tokens <b>{{ fmtNum(summary.total_prompt) }}</b></span>
          <span>输出 Tokens <b>{{ fmtNum(summary.total_completion) }}</b></span>
          <span class="stat-highlight">费用 &yen;{{ fmtCost(totalCost) }}</span>
        </div>

        <div class="table-wrap">
          <!-- 每日消耗统计表格 -->
          <table v-if="currentTab === 'daily'">
            <thead>
              <tr><th>日期</th><th>请求次数</th><th>输入 TOKENS</th><th>输出 TOKENS</th><th>总 TOKENS</th><th>费用</th></tr>
            </thead>
            <tbody>
              <tr v-for="d in dailyStats" :key="d.date">
                <td><span class="date-tag">{{ d.date }}</span></td>
                <td class="num">{{ d.count }} 次</td>
                <td class="num">{{ fmtNum(d.prompt) }}</td>
                <td class="num">{{ fmtNum(d.completion) }}</td>
                <td class="num bold">{{ fmtNum(d.total) }}</td>
                <td class="num price">&yen;{{ d.cost.toFixed(4) }}</td>
              </tr>
              <tr v-if="dailyStats.length === 0"><td colspan="6" class="no-data">暂无数据</td></tr>
            </tbody>
          </table>

          <table v-else-if="currentTab === 'model'">
            <thead>
              <tr><th>模型</th><th>请求次数</th><th>输入 TOKENS</th><th>输出 TOKENS</th><th>总 TOKENS</th><th>费用</th></tr>
            </thead>
            <tbody>
              <tr v-for="m in byModel" :key="m.model">
                <td><span class="model-tag">{{ m.display_name }}</span></td>
                <td class="num">{{ m.request_count }}</td>
                <td class="num">{{ fmtNum(m.prompt_tokens) }}</td>
                <td class="num">{{ fmtNum(m.completion_tokens) }}</td>
                <td class="num bold">{{ fmtNum(m.total_tokens) }}</td>
                <td class="num price">&yen;{{ fmtCost(calcCost(m.model, m.prompt_tokens, m.completion_tokens)) }}</td>
              </tr>
              <tr v-if="byModel.length === 0"><td colspan="6" class="no-data">暂无数据</td></tr>
            </tbody>
          </table>
          <table v-else>
            <thead>
              <tr><th>产品</th><th>请求次数</th><th>输入 TOKENS</th><th>输出 TOKENS</th><th>总 TOKENS</th><th>费用</th></tr>
            </thead>
            <tbody>
              <tr v-if="summary.total_requests > 0">
                <td><span class="model-tag">AI Agent Chat</span></td>
                <td class="num">{{ summary.total_requests }}</td>
                <td class="num">{{ fmtNum(summary.total_prompt) }}</td>
                <td class="num">{{ fmtNum(summary.total_completion) }}</td>
                <td class="num bold">{{ fmtNum(summary.total_tokens) }}</td>
                <td class="num price">&yen;{{ fmtCost(totalCost) }}</td>
              </tr>
              <tr v-else><td colspan="6" class="no-data">暂无数据</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Detail Records -->
      <div class="section-card">
        <div class="section-head">
          <h3 class="section-title">详细使用记录</h3>
          <div class="page-size-wrap">
            <span>每页</span>
            <select v-model="pageSize" @change="currentPage = 1">
              <option :value="25">25</option>
              <option :value="50">50</option>
              <option :value="100">100</option>
            </select>
            <span>条</span>
          </div>
        </div>

        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th class="sortable" @click="sortDetail('timestamp')">
                  <span class="sort-trigger">
                    请求时间
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="16 4 12 8 8 4"/></svg>
                  </span>
                </th>
                <th>模型</th>
                <th class="sortable num" @click="sortDetail('prompt_tokens')">
                  <span class="sort-trigger">
                    输入 TOKENS
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="16 4 12 8 8 4"/></svg>
                  </span>
                </th>
                <th class="sortable num" @click="sortDetail('completion_tokens')">
                  <span class="sort-trigger">
                    输出 TOKENS
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="16 4 12 8 8 4"/></svg>
                  </span>
                </th>
                <th class="sortable num" @click="sortDetail('total_tokens')">
                  <span class="sort-trigger">
                    总 TOKENS
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="16 4 12 8 8 4"/></svg>
                  </span>
                </th>
                <th class="sortable num" @click="sortDetail('cost')">
                  <span class="sort-trigger">
                    费用
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="16 4 12 8 8 4"/></svg>
                  </span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in pageRecords" :key="r.id">
                <td class="time">{{ fmtDateTime(r.timestamp) }}</td>
                <td><span class="model-tag-sm">{{ shortModelName(r.model) }}</span></td>
                <td class="num">{{ r.prompt_tokens }}</td>
                <td class="num">{{ r.completion_tokens }}</td>
                <td class="num bold">{{ r.total_tokens }}</td>
                <td class="num price">&yen;{{ fmtCost(calcCost(r.model, r.prompt_tokens, r.completion_tokens)) }}</td>
              </tr>
              <tr v-if="pageRecords.length === 0"><td colspan="6" class="no-data">暂无数据</td></tr>
            </tbody>
          </table>
        </div>

        <div class="detail-footer">
          <span class="footer-info">
            显示第 {{ pageRecords.length > 0 ? (currentPage - 1) * pageSize + 1 : 0 }}-{{ (currentPage - 1) * pageSize + pageRecords.length }} 条，共 {{ sortedRecords.length }} 条
          </span>
          <div class="pagination">
            <button :disabled="currentPage <= 1" @click="currentPage--">上一页</button>
            <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
            <button :disabled="currentPage >= totalPages" @click="currentPage++">下一页</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.stats-view { height: 100%; display: flex; flex-direction: column; }

/* ===== Header ===== */
.stats-header {
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-color);
}
.icon-btn {
  width: 32px; height: 32px; border-radius: 8px;
  background: transparent; border: none;
  color: var(--text-color-3); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all .15s;
}
.icon-btn:hover { background: var(--hover-color); color: var(--text-color); }

.title { font-size: 15px; font-weight: 600; letter-spacing: -0.2px; flex: 1; }
.refresh-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 5px 12px; border-radius: 8px;
  background: transparent; border: 1px solid var(--border-color);
  color: var(--text-color-3); font-size: 12px; cursor: pointer;
  transition: all .15s; font-family: inherit;
}
.refresh-btn:hover:not(:disabled) { color: var(--text-color); border-color: var(--primary-color-hover); }
.refresh-btn:disabled { opacity: .35; cursor: not-allowed; animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* ===== Body ===== */
.stats-body { flex: 1; overflow-y: auto; padding: 20px 28px; max-width: 1100px; width: 100%; margin: 0 auto; }

/* ===== Quota Card ===== */
.quota-card {
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  padding: 22px 26px;
  margin-bottom: 18px;
}
.quota-header { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
.icon-box {
  width: 36px; height: 36px; border-radius: 10px;
  background: rgba(34,197,94,.08);
  color: #22c55e;
  display: flex; align-items: center; justify-content: center;
}
.quota-title { font-size: 15px; font-weight: 600; letter-spacing: -0.2px; }
.quota-row { display: flex; align-items: baseline; gap: 14px; flex-wrap: wrap; }
.quota-used { font-size: 22px; font-weight: 700; color: #22c55e; }
.quota-sep { color: var(--text-color-3); font-size: 16px; margin: 0 2px; }
.quota-total { font-size: 16px; font-weight: 500; color: var(--text-color-2); }
.quota-pct-wrap { margin-left: auto; font-size: 13px; color: var(--text-color-3); }
.quota-pct-wrap b { color: #22c55e; font-weight: 600; }
.progress-bar {
  width: 100%; height: 8px;
  background: var(--hover-color);
  border-radius: 4px;
  overflow: hidden;
  margin-top: 16px;
}
.progress-fill {
  height: 100%; border-radius: 4px;
  transition: width .6s ease;
  background: linear-gradient(90deg, #5b8def, #7ba5f5, #8b5cf6);
}

/* ===== Section Card ===== */
.section-card {
  background: var(--card-color);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  margin-bottom: 18px;
  overflow: hidden;
}
.section-head {
  display: flex;
  align-items: center;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-color);
  gap: 14px;
  flex-wrap: wrap;
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.2px;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}
.section-title::before {
  content: "";
  width: 3px; height: 16px;
  border-radius: 2px;
  background: var(--primary-color);
}
.section-stats {
  display: flex;
  gap: 20px;
  font-size: 12px;
  color: var(--text-color-3);
  padding: 10px 20px 0;
  flex-wrap: wrap;
}
.section-stats b { color: var(--text-color); font-weight: 600; }
.stat-highlight { margin-left: auto; color: #22c55e !important; font-weight: 600; }

.section-tabs { display: flex; gap: 4px; }
.section-tabs button {
  padding: 5px 13px; border-radius: 7px; font-size: 12.5px; cursor: pointer;
  background: transparent; border: 1px solid transparent;
  color: var(--text-color-3); transition: all .15s;
  white-space: nowrap; font-family: inherit; font-weight: 500;
}
.section-tabs button:hover { color: var(--text-color-2); }
.section-tabs button.active {
  background: rgba(91,141,239,.08);
  border-color: rgba(91,141,239,.2);
  color: var(--primary-color); font-weight: 600;
}

.page-size-wrap {
  margin-left: auto;
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; color: var(--text-color-3);
}
.page-size-wrap select {
  border: 1px solid var(--border-color); border-radius: 6px; padding: 3px 8px;
  font-size: 12px; color: var(--text-color-1);
  background: var(--card-color); cursor: pointer;
  outline: none; font-family: inherit;
}

/* ===== Tables ===== */
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; }
th, td {
  padding: 11px 16px;
  text-align: left;
  font-size: 13px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-color);
}
th {
  background: var(--hover-color);
  color: var(--text-color-3);
  font-weight: 600;
  font-size: 12px;
  letter-spacing: .1px;
}
th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color .15s;
}
th.sortable:hover { color: var(--text-color-2); }
.sort-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
th.num .sort-trigger {
  display: flex;
  justify-content: flex-end;
}
th svg { opacity: .35; flex-shrink: 0; }

td { color: var(--text-color-2); vertical-align: middle; }
tr:hover td { background: var(--hover-color); }
.num { text-align: right; font-variant-numeric: tabular-nums; }
.bold { font-weight: 600; color: var(--text-color) !important; }
.price { color: #22c55e; font-weight: 600; }
.time { font-size: 12px; color: var(--text-color-3); font-variant-numeric: tabular-nums; }
.no-data {
  text-align: center;
  color: var(--text-color-3);
  padding: 36px 16px;
  font-size: 13px;
}

.model-tag {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 4px 11px; border-radius: 7px; font-size: 12px; font-weight: 500;
  background: rgba(91,141,239,.06); border: 1px solid rgba(91,141,239,.15);
  color: var(--primary-color);
}
.model-tag-sm {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 5px;
  font-size: 11px;
  font-weight: 500;
  background: rgba(91,141,239,.06);
  color: var(--primary-color);
  border: 1px solid rgba(91,141,237,.12);
}

/* ===== Footer ===== */
.detail-footer {
  padding: 12px 20px;
  display: flex; align-items: center; justify-content: space-between;
  border-top: 1px solid var(--border-color);
  font-size: 12px; color: var(--text-color-3);
}
.footer-info {}
.pagination { display: flex; align-items: center; gap: 6px; }
.pagination button {
  min-width: 30px; height: 30px; border-radius: 7px; border: 1px solid var(--border-color);
  background: var(--card-color); color: var(--text-color-2); font-size: 12px; cursor: pointer;
  display: flex; align-items: center; justify-content: center; padding: 0 10px;
  transition: all .12s; font-family: inherit; font-weight: 500;
}
.pagination button:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: rgba(91,141,239,.05);
}
.pagination button:disabled { opacity: .35; cursor: not-allowed; }
.page-info { font-size: 12px; color: var(--text-color-3); min-width: 40px; text-align: center; }

@media (max-width: 768px) {
  .stats-body { padding: 14px 16px; }
  .quota-card { padding: 16px 18px; }
  .section-head { padding: 12px 14px; }
  th, td { padding: 9px 12px; }
}

.date-tag {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
  padding: 3px 8px;
  border-radius: 6px;
  font-weight: 600;
  font-size: 11.5px;
}
</style>