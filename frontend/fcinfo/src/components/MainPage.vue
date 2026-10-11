<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import 'element-plus/theme-chalk/dark/css-vars.css'

import TickChart from './charts/TickChart.vue'
import KLineChart from './charts/KLineChart.vue'
import PositionChart from './charts/PositionChart.vue'
import PositionTable from './PositionTable.vue'
import type {
  DataSourceOption,
  KLineDay,
  KpiItem,
  PositionItem,
  Snapshot,
  SubscriptionItem,
  TickPoint
} from '@/types/stock'
import { ApiRequestError } from '@/api/request'
import { fetchKLineData, fetchSnapshot, fetchTickData, searchStocks } from '@/api/market'
import {
  fetchPositions,
  fetchSubscriptions,
  subscribeStock,
  unsubscribeStock
} from '@/api/business'
import { usePolling } from '@/composables/usePolling'
import {
  calcPositionValue,
  formatAmount,
  formatMoney,
  formatPercent,
  formatSigned,
  formatSignedMoney,
  todayISO
} from '@/utils/stock'

// ---------- 主题切换 ----------
const isDark = ref(false)
const toggleTheme = () => {
  document.documentElement.classList.toggle('dark', isDark.value)
}

// ---------- 顶部导航 ----------
const activeNav = ref('首页')
const navItems = ['首页', '热点', '其他']

// ---------- 签到日期 ----------
const checkinDate = ref(todayISO())

// ---------- 数据来源（纯前端状态，不参与后端取数） ----------
const DATA_SOURCES: DataSourceOption[] = [
  { label: 'API 接口', value: 'api' },
  { label: '数据库', value: 'db' },
  { label: 'CSV 文件', value: 'csv' }
]
const dataSource = ref('api')

// ---------- 行情标签页 ----------
const activeTab = ref('tick')

// ---------- 页面数据 ----------
const loading = ref(true)
const subscriptions = ref<SubscriptionItem[]>([])
const currentCode = ref('')
const snapshot = ref<Snapshot | null>(null)
const tickData = ref<TickPoint[]>([])
const klineData = ref<KLineDay[]>([])
const rawPositions = ref<PositionItem[]>([])

const currentStock = computed(() =>
  subscriptions.value.find((s) => s.code === currentCode.value)
)
const positionsWithValue = computed(() => calcPositionValue(rawPositions.value))

// KPI 卡片：后端只给数值，文案在这里拼
const kpiItems = computed<KpiItem[]>(() => {
  const s = snapshot.value
  if (!s) return []
  const items: KpiItem[] = [
    {
      label: '现价',
      value: s.price.toFixed(2),
      // 没有昨收（库里只有今天的数据）就不显示涨跌幅，别拿 0 冒充
      delta:
        s.change !== null && s.changePercent !== null
          ? `${formatSigned(s.change)} (${formatPercent(s.changePercent)})`
          : undefined,
      up: (s.change ?? 0) >= 0,
      hero: true
    },
    { label: '今日最高', value: s.high.toFixed(2) },
    { label: '成交额', value: formatAmount(s.cumAmount) }
  ]

  const list = positionsWithValue.value
  if (list.length) {
    const total = list.reduce((sum, p) => sum + p.marketValue, 0)
    const cost = list.reduce((sum, p) => sum + p.cost * p.shares, 0)
    const profit = total - cost
    items.push({
      label: '持仓市值',
      value: formatMoney(total),
      delta: `${formatSignedMoney(profit)} (${formatPercent(cost ? (profit / cost) * 100 : 0)})`,
      up: profit >= 0
    })
  }
  return items
})

// ---------- 错误提示：后端不通时每 5s 一次会刷屏，同样的话只说一遍 ----------
let errorAnnounced = false
function reportError(err: unknown, fallback: string) {
  if (errorAnnounced) return
  errorAnnounced = true
  ElMessage.error(err instanceof ApiRequestError ? err.message : fallback)
}

// ---------- 拉数 ----------
async function loadRealtime(signal?: AbortSignal) {
  const code = currentCode.value
  if (!code) return
  const [snap, ticks] = await Promise.all([
    fetchSnapshot(code, signal),
    fetchTickData(code, undefined, signal)
  ])
  // 请求期间切了股票，这次结果作废，别覆盖新股票的数据
  if (code !== currentCode.value) return
  snapshot.value = snap
  tickData.value = ticks
}

async function loadKline(signal?: AbortSignal) {
  const code = currentCode.value
  if (!code) return
  const data = await fetchKLineData(code, '1d', 30, signal)
  if (code !== currentCode.value) return
  klineData.value = data
}

async function loadPositions(signal?: AbortSignal) {
  rawPositions.value = await fetchPositions(signal)
}

// 分时与快照要求实时，日K 与持仓变化慢
const realtimePolling = usePolling(async (signal) => {
  try {
    await loadRealtime(signal)
    errorAnnounced = false
  } catch (err) {
    if (!signal.aborted) reportError(err, '行情刷新失败')
  }
}, 5000)

const slowPolling = usePolling(async (signal) => {
  try {
    await Promise.all([loadKline(signal), loadPositions(signal)])
    errorAnnounced = false
  } catch (err) {
    if (!signal.aborted) reportError(err, '数据刷新失败')
  }
}, 60000)

// ---------- 订阅 ----------
const stockCode = ref('')

async function handleSubscribe() {
  const keyword = stockCode.value.trim()
  if (!keyword) {
    ElMessage.warning('请输入股票代码或名称')
    return
  }
  try {
    // 订阅体只收 code，所以先用搜索把"代码或名称"解析成代码
    const items = await searchStocks(keyword)
    if (!items.length) {
      ElMessage.warning(`未找到「${keyword}」`)
      return
    }
    const target = items[0]
    if (items.length > 1) {
      ElMessage.info(`匹配到 ${items.length} 条，已订阅 ${target.name}（${target.code}）`)
    }

    const item = await subscribeStock(target.code)
    subscriptions.value.push(item)
    stockCode.value = ''
    if (!currentCode.value) currentCode.value = item.code
    ElMessage.success(`已订阅 ${item.name}（${item.code}）`)
    errorAnnounced = false
  } catch (err) {
    if (err instanceof ApiRequestError && err.code === 'ALREADY_SUBSCRIBED') {
      ElMessage.info(`${keyword} 已在订阅列表中`)
      return
    }
    if (err instanceof ApiRequestError && err.code === 'SYMBOL_NOT_FOUND') {
      ElMessage.warning(`未找到「${keyword}」`)
      return
    }
    reportError(err, '订阅失败')
  }
}

async function handleUnsubscribe(code: string) {
  try {
    await unsubscribeStock(code)
    subscriptions.value = subscriptions.value.filter((s) => s.code !== code)
    if (currentCode.value === code) {
      currentCode.value = subscriptions.value[0]?.code ?? ''
      snapshot.value = null
      tickData.value = []
      klineData.value = []
    }
  } catch (err) {
    if (err instanceof ApiRequestError && err.code === 'SUBSCRIPTION_NOT_FOUND') {
      subscriptions.value = subscriptions.value.filter((s) => s.code !== code)
      return
    }
    reportError(err, '取消订阅失败')
  }
}

async function selectStock(code: string) {
  if (code === currentCode.value) return
  currentCode.value = code
  snapshot.value = null
  tickData.value = []
  klineData.value = []
  try {
    await Promise.all([loadRealtime(), loadKline()])
    errorAnnounced = false
  } catch (err) {
    reportError(err, '加载行情失败')
  }
}

// ---------- 启动 ----------
onMounted(async () => {
  try {
    subscriptions.value = await fetchSubscriptions()
    currentCode.value = subscriptions.value[0]?.code ?? ''
    errorAnnounced = false
  } catch (err) {
    reportError(err, '加载订阅列表失败')
  } finally {
    loading.value = false
  }
  // start() 会立刻跑一轮，首屏的行情、日K、持仓由这里带出来
  realtimePolling.start()
  slowPolling.start()
})
</script>

<template>
  <div class="main-page" :class="{ dark: isDark }">

    <!-- ========== 顶部导航栏 ========== -->
    <header class="top-bar">
      <el-row :gutter="14" align="middle">
        <el-col v-for="item in navItems" :key="item" :span="3">
          <el-button
            :type="activeNav === item ? 'primary' : 'default'"
            :plain="activeNav !== item"
            round
            text
            @click="activeNav = item"
          >
            {{ item }}
          </el-button>
        </el-col>

        <el-col :span="3" :offset="6">
          <el-date-picker
            v-model="checkinDate"
            type="date"
            placeholder="签到"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-col>

        <el-col :span="3">
          <el-button round text>我的</el-button>
        </el-col>

        <el-col :span="3">
          <el-switch v-model="isDark" active-text="暗" inactive-text="亮" @change="toggleTheme" />
        </el-col>
      </el-row>
    </header>

    <!-- ========== KPI 指标行 ========== -->
    <section class="kpi-row">
      <div
        v-for="kpi in kpiItems"
        :key="kpi.label"
        class="kpi-tile"
        :class="{ hero: kpi.hero }"
      >
        <div class="kpi-label">{{ kpi.label }}</div>
        <div class="kpi-value">{{ kpi.value }}</div>
        <div v-if="kpi.delta" class="kpi-delta" :class="kpi.up ? 'up' : 'down'">
          {{ kpi.delta }}
        </div>
      </div>
    </section>

    <!-- ========== 数据卡片 ========== -->
    <el-card v-loading="loading" class="data-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <template v-if="currentStock">
              数据 · {{ currentStock.name }}（{{ currentStock.code }}）
            </template>
            <template v-else>数据 · 暂无订阅</template>
          </span>
          <div class="subscribed-tags">
            <el-tag
              v-for="sub in subscriptions"
              :key="sub.code"
              closable
              round
              size="small"
              class="stock-tag"
              :effect="sub.code === currentCode ? 'dark' : 'plain'"
              @click="selectStock(sub.code)"
              @close="handleUnsubscribe(sub.code)"
            >
              {{ sub.name }}
            </el-tag>
          </div>
        </div>
      </template>

      <!-- 工具条：数据来源 + 订阅 -->
      <el-row :gutter="12" align="middle" class="toolbar">
        <el-col :xs="24" :sm="8">
          <el-select v-model="dataSource" placeholder="数据来源" style="width: 100%">
            <el-option
              v-for="opt in DATA_SOURCES"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-col>

        <el-col :xs="24" :sm="12">
          <el-input
            v-model="stockCode"
            placeholder="输入股票代码或名称（如 000001 / 贵州茅台）"
            :prefix-icon="Search"
            clearable
            @keyup.enter="handleSubscribe"
          />
        </el-col>

        <el-col :xs="24" :sm="4">
          <el-button type="primary" round style="width: 100%" @click="handleSubscribe">
            订阅
          </el-button>
        </el-col>
      </el-row>

      <!-- 行情标签页 -->
      <el-tabs v-model="activeTab" class="chart-tabs">
        <el-tab-pane label="tick" name="tick">
          <el-row :gutter="12">
            <el-col :xs="24" :lg="12">
              <div class="chart-panel">
                <h4>分时走势</h4>
                <TickChart :data="tickData" :dark="isDark" />
              </div>
            </el-col>
            <el-col :xs="24" :lg="12">
              <div class="chart-panel">
                <h4>持仓分布</h4>
                <PositionChart :positions="positionsWithValue" :dark="isDark" />
                <PositionTable :positions="positionsWithValue" />
              </div>
            </el-col>
          </el-row>
        </el-tab-pane>

        <el-tab-pane label="日K" name="daily">
          <el-row :gutter="12">
            <el-col :xs="24" :lg="12">
              <div class="chart-panel">
                <h4>日K走势</h4>
                <KLineChart :data="klineData" :dark="isDark" />
              </div>
            </el-col>
            <el-col :xs="24" :lg="12">
              <div class="chart-panel">
                <h4>持仓分布</h4>
                <PositionChart :positions="positionsWithValue" :dark="isDark" />
                <PositionTable :positions="positionsWithValue" />
              </div>
            </el-col>
          </el-row>
        </el-tab-pane>
      </el-tabs>
    </el-card>

  </div>
</template>

<style scoped>
/* ======== 全局 ======== */
.main-page {
  min-height: 100vh;
  background: var(--bg-page);
  color: var(--text-secondary);
  transition: background 0.3s, color 0.3s;
}

/* ======== 顶部导航 ======== */
.top-bar {
  padding: 16px 24px;
  margin: 24px;
  background: var(--bg-card);
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

/* ======== KPI 指标行 ======== */
.kpi-row {
  display: flex;
  gap: 16px;
  margin: 0 24px 16px;
  flex-wrap: wrap;
}

.kpi-tile {
  flex: 1 1 160px;
  min-width: 160px;
  background: var(--bg-card);
  border-radius: 8px;
  padding: 14px 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.kpi-label {
  font-size: 12px;
  color: var(--text-muted);
}

.kpi-value {
  font-size: 22px;
  font-weight: 600;
  margin: 4px 0;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.kpi-tile.hero .kpi-value {
  font-size: 48px;
  line-height: 1.1;
}

.kpi-delta {
  font-size: 12px;
}

.kpi-delta.up {
  color: var(--rise-color);
}

.kpi-delta.down {
  color: var(--fall-color);
}

/* ======== 数据卡片 ======== */
.data-card {
  margin: 0 24px 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.card-title {
  font-weight: 600;
  font-size: 15px;
}

.subscribed-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.stock-tag {
  cursor: pointer;
}

.toolbar {
  margin-bottom: 16px;
}

/* ======== 图表面板 ======== */
.chart-tabs {
  margin-top: 8px;
}

.chart-panel {
  background: var(--bg-card);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
}

.chart-panel h4 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}
</style>
