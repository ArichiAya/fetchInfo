<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import 'element-plus/theme-chalk/dark/css-vars.css'

import TickChart from './charts/TickChart.vue'
import KLineChart from './charts/KLineChart.vue'
import PositionChart from './charts/PositionChart.vue'
import PositionTable from './PositionTable.vue'
import type {
  KLineDay,
  KpiItem,
  PositionItem,
  Snapshot,
  StockSymbol,
  SubscriptionItem,
  TickPoint
} from '@/types/stock'
import { ApiRequestError } from '@/api/request'
import { logout } from '@/api/auth'
import { fetchKLineData, fetchSnapshot, fetchTickData, searchStocks } from '@/api/market'
import {
  fetchPositions,
  fetchSubscriptions,
  subscribeStock,
  // unsubscribeStock
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

// ---------- 退出登录 ----------
const router = useRouter()

function handleLogout() {
  logout()
  router.replace({ name: 'login' })
}

// ---------- 行情标签页 ----------
const activeTab = ref('tick')

// ---------- 查看日期 / K线周期 ----------
/** 查看的交易日：分时按天取，K线以这天为 ±7 天窗口的中心 */
const viewDate = ref(todayISO())
const isToday = computed(() => viewDate.value === todayISO())

/** K线周期，在图上滚轮切换 */
const KLINE_PERIODS = ['1m', '5m', '15m', '1d']
const klinePeriod = ref('1d')

// ---------- 页面数据 ----------
const loading = ref(true)
const subscriptions = ref<SubscriptionItem[]>([])
/**
 * 正在查看的标的。
 * 与 subscriptions（自选列表）无关 —— 直接输入代码就能看，不需要先订阅。
 */
const current = ref<StockSymbol | null>(null)
const currentCode = computed(() => current.value?.code ?? '')
const snapshot = ref<Snapshot | null>(null)
const tickData = ref<TickPoint[]>([])
const klineData = ref<KLineDay[]>([])
const rawPositions = ref<PositionItem[]>([])

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
async function loadSnapshot(signal?: AbortSignal) {
  const code = currentCode.value
  if (!code) return
  const snap = await fetchSnapshot(code, signal)
  // 请求期间切了股票，这次结果作废，别覆盖新股票的数据
  if (code !== currentCode.value) return
  snapshot.value = snap
}

async function loadTick(signal?: AbortSignal) {
  const code = currentCode.value
  if (!code) return
  const date = viewDate.value
  const ticks = await fetchTickData(code, date, signal)
  if (code !== currentCode.value || date !== viewDate.value) return
  // 后端在没有数据时返回的是 null（nil slice），兜一下免得图表炸掉
  tickData.value = ticks ?? []
}

async function loadKline(signal?: AbortSignal) {
  const code = currentCode.value
  if (!code) return
  const period = klinePeriod.value
  const date = viewDate.value
  // kline 的 date 是 `YYYY-MM-DD HH:MM:SS`，而且是 ±7 天窗口的中心；
  // tick 的 date 是 `YYYY-MM-DD`。两个格式不一样，别直接共用。
  const data = await fetchKLineData(code, period, 30, `${date} 00:00:00`, signal)
  if (code !== currentCode.value || period !== klinePeriod.value || date !== viewDate.value) {
    return
  }
  klineData.value = data ?? []
}

async function loadPositions(signal?: AbortSignal) {
  rawPositions.value = await fetchPositions(signal)
}

// 快照要求实时，日K 与持仓变化慢
const realtimePolling = usePolling(async (signal) => {
  try {
    await loadSnapshot(signal)
    // 只有看当天时才跟着刷分时 —— 历史某天的曲线不会再变
    if (isToday.value) await loadTick(signal)
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

// ---------- 查看 / 订阅 ----------
// 两件事是分开的：查看只切当前标的，订阅只往自选列表里加，互不影响。
const stockCode = ref('')

/** 把输入框里的代码或名称解析成一个标的 */
async function resolveStock(keyword: string): Promise<StockSymbol | null> {
  const items = await searchStocks(keyword)
  if (!items.length) {
    ElMessage.warning(`未找到「${keyword}」`)
    return null
  }
  const target = items[0]
  if (items.length > 1) {
    ElMessage.info(`匹配到 ${items.length} 条，取 ${target.name}（${target.code}）`)
  }
  return target
}

/** 查看：切到这个标的看行情 */
async function handleView() {
  const keyword = stockCode.value.trim()
  if (!keyword) {
    ElMessage.warning('请输入股票代码或名称')
    return
  }
  try {
    const target = await resolveStock(keyword)
    if (target) await selectStock(target)
  } catch (err) {
    if (err instanceof ApiRequestError && err.code === 'SYMBOL_NOT_FOUND') {
      ElMessage.warning(`未找到「${keyword}」`)
      return
    }
    reportError(err, '加载行情失败')
  }
}

/** 订阅：加进自选列表，不改变当前正在看的标的 */
async function handleSubscribe() {
  const keyword = stockCode.value.trim()
  if (!keyword) {
    ElMessage.warning('请输入股票代码或名称')
    return
  }
  try {
    const target = await resolveStock(keyword)
    if (!target) return

    const item = await subscribeStock(target.code)
    subscriptions.value.push(item)
    ElMessage.success(`已订阅 ${item.name}（${item.code}）`)
    errorAnnounced = false
  } catch (err) {
    if (err instanceof ApiRequestError && err.code === 'ALREADY_SUBSCRIBED') {
      ElMessage.info(`${keyword} 已在自选列表中`)
      return
    }
    if (err instanceof ApiRequestError && err.code === 'SYMBOL_NOT_FOUND') {
      ElMessage.warning(`未找到「${keyword}」`)
      return
    }
    reportError(err, '订阅失败')
  }
}


async function handleUnsubscribe(index: number) {
  subscriptions.value.splice(index,1)
}

/** 切换正在查看的标的，并立刻拉一轮，不用等下一次轮询 */
async function selectStock(stock: StockSymbol) {
  if (stock.code === currentCode.value) return
  current.value = stock
  snapshot.value = null
  tickData.value = []
  klineData.value = []
  try {
    await Promise.all([loadSnapshot(), loadTick(), loadKline()])
    errorAnnounced = false
  } catch (err) {
    reportError(err, '加载行情失败')
  }
}

/** 换查看日期：分时和K线都要重拉（快照永远是"此刻"，不受影响） */
async function handleDateChange() {
  tickData.value = []
  klineData.value = []
  try {
    await Promise.all([loadTick(), loadKline()])
    errorAnnounced = false
  } catch (err) {
    reportError(err, '加载失败')
  }
}

/** 滚轮换 K 线周期：-1 更短、+1 更长，到头就不动 */
async function handlePeriodStep(step: number) {
  const i = KLINE_PERIODS.indexOf(klinePeriod.value)
  const next = KLINE_PERIODS[Math.min(Math.max(i + step, 0), KLINE_PERIODS.length - 1)]
  if (next === klinePeriod.value) return

  klinePeriod.value = next
  klineData.value = []
  try {
    await loadKline()
    errorAnnounced = false
  } catch (err) {
    reportError(err, '加载K线失败')
  }
}

// ---------- 启动 ----------
onMounted(async () => {
  try {
    subscriptions.value = await fetchSubscriptions()
    // 有自选就先看第一只；没有就等用户输入代码，别硬塞一个
    const first = subscriptions.value[0]
    if (first) current.value = { code: first.code, name: first.name }
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
          <el-button round text @click="handleLogout">退出</el-button>
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
            <template v-if="current">
              数据 · {{ current.name }}（{{ current.code }}）
            </template>
            <template v-else>数据 · 输入股票代码查看</template>
          </span>
          <div class="subscribed-tags">
            <el-tag
              v-for="(sub, index) in subscriptions"
              :key="sub.code"
              closable
              round
              size="small"
              class="stock-tag"
              :effect="sub.code === currentCode ? 'dark' : 'plain'"
              @click="selectStock(sub)"
              @close="handleUnsubscribe(index)"
            >
              {{ sub.name }}
            </el-tag>
          </div>
        </div>
      </template>

      <!-- 工具条：查看行情 / 加入自选（两件独立的事） -->
      <el-row :gutter="12" align="middle" class="toolbar">
        <el-col :xs="24" :sm="12">
          <el-input
            v-model="stockCode"
            placeholder="输入股票代码或名称（如 000001 / 贵州茅台）"
            :prefix-icon="Search"
            clearable
            @keyup.enter="handleView"
          />
        </el-col>

        <el-col :xs="12" :sm="6">
          <el-button type="primary" round style="width: 100%" @click="handleView">
            查看
          </el-button>
        </el-col>

        <el-col :xs="12" :sm="6">
          <el-button round style="width: 100%" @click="handleSubscribe">订阅</el-button>
        </el-col>
      </el-row>

      <!-- 还没选标的时不渲染图表，给个提示 -->
      <div v-if="!current" class="empty-hint">输入股票代码后点「查看」加载行情</div>

      <!-- 行情标签页 -->
      <el-tabs v-else v-model="activeTab" class="chart-tabs">
        <el-tab-pane label="tick" name="tick">
          <el-row :gutter="12">
            <el-col :xs="24" :lg="12">
              <div class="chart-panel">
                <div class="panel-header">
                  <h4>分时走势</h4>
                  <el-date-picker
                    v-model="viewDate"
                    type="date"
                    placeholder="交易日"
                    value-format="YYYY-MM-DD"
                    :clearable="false"
                    size="small"
                    style="width: 148px"
                    @change="handleDateChange"
                  />
                </div>
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

        <el-tab-pane label="K线" name="daily">
          <el-row :gutter="12">
            <el-col :xs="24" :lg="12">
              <div class="chart-panel">
                <div class="panel-header">
                  <h4>K线走势 · {{ klinePeriod }}</h4>
                  <div class="panel-tools">
                    <span class="panel-hint">滚轮切换周期</span>
                    <!-- 和分时面板绑同一个 viewDate，改哪边都生效 -->
                    <el-date-picker
                      v-model="viewDate"
                      type="date"
                      placeholder="交易日"
                      value-format="YYYY-MM-DD"
                      :clearable="false"
                      size="small"
                      style="width: 148px"
                      @change="handleDateChange"
                    />
                  </div>
                </div>
                <KLineChart
                  :data="klineData"
                  :dark="isDark"
                  :period="klinePeriod"
                  @step-period="handlePeriodStep"
                />
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

.empty-hint {
  padding: 48px 0;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
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

/* 标题右侧可以挂日期选择器 / 提示文字 */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.panel-header h4 {
  margin: 0;
}

.panel-tools {
  display: flex;
  align-items: center;
  gap: 10px;
}

.panel-hint {
  font-size: 12px;
  color: var(--text-muted);
}
</style>
