<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search } from '@element-plus/icons-vue'
import 'element-plus/theme-chalk/dark/css-vars.css'

import TickChart from './charts/TickChart.vue'
import KLineChart from './charts/KLineChart.vue'
import PositionChart from './charts/PositionChart.vue'
import PositionTable from './PositionTable.vue'
import type { SubscriptionItem } from '@/types/stock'
import {
  CURRENT_STOCK,
  STOCK_CATALOG,
  DATA_SOURCES,
  KPI_ITEMS,
  POSITIONS,
  INITIAL_SUBSCRIPTIONS,
  MOCK_TICK_DATA,
  MOCK_KLINE_DATA,
  calcPositionValue
} from '@/mock/stockData'

// ---------- 主题切换 ----------
const isDark = ref(false)
const toggleTheme = () => {
  document.documentElement.classList.toggle('dark', isDark.value)
}

// ---------- 顶部导航 ----------
const activeNav = ref('首页')
const navItems = ['首页', '热点', '其他']

// ---------- 签到日期 ----------
const checkinDate = ref('2026-08-13')

// ---------- 数据来源 ----------
const dataSource = ref('api')

// ---------- 股票订阅（写死目录验证流程，对接后端后替换为 API 调用） ----------
const stockCode = ref('')
const subscriptions = ref<SubscriptionItem[]>([...INITIAL_SUBSCRIPTIONS])

const subscribeStock = () => {
  const keyword = stockCode.value.trim()
  if (!keyword) {
    ElMessage.warning('请输入股票代码或名称')
    return
  }
  const stock = STOCK_CATALOG.find((s) => s.code === keyword || s.name === keyword)
  if (!stock) {
    ElMessage.warning(`未找到「${keyword}」，试试 000001 / 600519 / 300750`)
    return
  }
  if (subscriptions.value.some((s) => s.code === stock.code)) {
    ElMessage.info(`${stock.name} 已在订阅列表中`)
    return
  }
  subscriptions.value.push({
    code: stock.code,
    name: stock.name,
    subscribedAt: new Date().toLocaleString('zh-CN')
  })
  stockCode.value = ''
  ElMessage.success(`已订阅 ${stock.name}（${stock.code}）`)
}

const unsubscribe = (code: string) => {
  subscriptions.value = subscriptions.value.filter((s) => s.code !== code)
}

// ---------- 图表数据（写死，验证前端功能） ----------
const activeTab = ref('tick')
const tickData = MOCK_TICK_DATA
const klineData = MOCK_KLINE_DATA
const positionsWithValue = computed(() => calcPositionValue(POSITIONS))
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
        v-for="kpi in KPI_ITEMS"
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
    <el-card class="data-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">数据 · {{ CURRENT_STOCK.name }}（{{ CURRENT_STOCK.code }}）</span>
          <div class="subscribed-tags">
            <el-tag
              v-for="sub in subscriptions"
              :key="sub.code"
              closable
              round
              size="small"
              @close="unsubscribe(sub.code)"
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
            @keyup.enter="subscribeStock"
          />
        </el-col>

        <el-col :xs="24" :sm="4">
          <el-button type="primary" round style="width: 100%" @click="subscribeStock">
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
