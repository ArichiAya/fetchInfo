/**
 * Mock 数据（写死数据，用于验证前端功能）
 * 全部为确定性生成（正弦函数叠加），刷新页面数据不变
 * 对接后端后删除本文件，改用 API 请求
 */
import type {
  TickPoint,
  KLineDay,
  PositionItem,
  SubscriptionItem,
  KpiItem,
  StockSymbol,
  DataSourceOption
} from '@/types/stock'

// ---------- 当前主股票 ----------
export const CURRENT_STOCK = { code: '000001', name: '平安银行' }
export const BASE_PRICE = 12.86

// ---------- 股票订阅目录（写死，订阅时从这里查找） ----------
export const STOCK_CATALOG: StockSymbol[] = [
  { code: '000001', name: '平安银行' },
  { code: '600519', name: '贵州茅台' },
  { code: '300750', name: '宁德时代' },
  { code: '601318', name: '中国平安' },
  { code: '000858', name: '五粮液' }
]

// ---------- 分时数据：240 个点（上午 09:30-11:30 + 下午 13:00-15:00） ----------
function pad2(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

function buildTickData(): TickPoint[] {
  const points: TickPoint[] = []
  const n = 240
  for (let i = 0; i < n; i++) {
    // 时间：前 120 点为上午，后 120 点为下午
    const isMorning = i < 120
    const minutes = isMorning ? i : i - 120
    const hour = isMorning ? Math.floor((9 * 60 + 30 + minutes) / 60) : Math.floor((13 * 60 + minutes) / 60)
    const minute = isMorning ? (30 + minutes) % 60 : minutes % 60
    // 确定性价格走势：基准价 + 两个正弦波叠加
    const wave =
      0.18 * Math.sin(i * 0.045) +
      0.09 * Math.sin(i * 0.13 + 1.2) +
      0.05 * Math.sin(i * 0.031 + 4.1)
    const price = BASE_PRICE + wave
    points.push({
      time: `${pad2(hour)}:${pad2(minute)}`,
      price: Number(price.toFixed(2)),
      avgPrice: 0, // 下面统一算均价
      volume: Math.round(850 + 620 * Math.abs(Math.sin(i * 0.09)) * (1 + 0.25 * Math.sin(i * 0.37)))
    })
  }
  // 均价线 = VWAP（累计成交额 / 累计成交量），与后端 indicator.CalVWAP 同口径
  let cumAmount = 0
  let cumVolume = 0
  for (let i = 0; i < n; i++) {
    cumAmount += points[i].price * points[i].volume
    cumVolume += points[i].volume
    points[i].avgPrice = Number((cumAmount / cumVolume).toFixed(2))
  }
  return points
}

// ---------- 日K数据：30 个交易日（截止 2026-08-13） ----------
function buildKLineData(): KLineDay[] {
  const list: KLineDay[] = []
  const DAY_MS = 24 * 60 * 60 * 1000
  const end = new Date(2026, 7, 13).getTime()
  const count = 30
  for (let i = count - 1; i >= 0; i--) {
    const d = new Date(end - i * DAY_MS)
    const ts = `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T00:00:00+08:00`
    const idx = count - 1 - i
    // 确定性 OHLC
    const trend = 0.02 * idx + 0.38 * Math.sin(idx * 0.35)
    const open = BASE_PRICE + trend + 0.18 * Math.sin(idx * 1.1)
    const close = open + 0.3 * Math.sin(idx * 0.7 + 2.0)
    const high = Math.max(open, close) + 0.08 + 0.15 * Math.abs(Math.sin(idx * 0.9))
    const low = Math.min(open, close) - 0.08 - 0.15 * Math.abs(Math.cos(idx * 0.6))
    const volume = Math.round(
      920 + 700 * Math.abs(Math.sin(idx * 0.23)) * (1 + 0.2 * Math.sin(idx * 0.51))
    )
    list.push({
      ts,
      open: Number(open.toFixed(2)),
      close: Number(close.toFixed(2)),
      high: Number(high.toFixed(2)),
      low: Number(low.toFixed(2)),
      volume,
      // 成交额 = 成交量(手) × 100 × 收盘价
      amount: Number((volume * close * 100).toFixed(0))
    })
  }
  return list
}

// ---------- 持仓（写死 3 项） ----------
export const POSITIONS: PositionItem[] = [
  { code: '600519', name: '贵州茅台', shares: 200, cost: 1580.0, price: 1665.8 },
  { code: '300750', name: '宁德时代', shares: 500, cost: 210.5, price: 228.4 },
  { code: '000001', name: '平安银行', shares: 1000, cost: 12.2, price: 12.86 }
]

// 市值与盈亏的计算已挪到 @/utils/stock，这里转出去保持向后兼容
export { calcPositionValue } from '@/utils/stock'

// ---------- 已订阅（写死初始 2 条） ----------
export const INITIAL_SUBSCRIPTIONS: SubscriptionItem[] = [
  { code: '000001', name: '平安银行', subscribedAt: '2026-08-01 09:15' },
  { code: '600519', name: '贵州茅台', subscribedAt: '2026-07-28 14:02' }
]

// ---------- KPI 指标（写死，hero 为现价） ----------
export const KPI_ITEMS: KpiItem[] = [
  { label: '现价', value: '12.86', delta: '+0.29 (2.30%)', up: true, hero: true },
  { label: '今日最高', value: '12.92' },
  { label: '成交额', value: '4.82亿', delta: '+8.1% vs 昨日', up: true },
  { label: '持仓市值', value: '¥460,220', delta: '+¥2,860 (0.63%)', up: true }
]

// ---------- 数据来源选项 ----------
export const DATA_SOURCES: DataSourceOption[] = [
  { label: 'API 接口', value: 'api' },
  { label: '数据库', value: 'db' },
  { label: 'CSV 文件', value: 'csv' }
]

// ---------- 导出写死的数据 ----------
export const MOCK_TICK_DATA: TickPoint[] = buildTickData()
export const MOCK_KLINE_DATA: KLineDay[] = buildKLineData()
