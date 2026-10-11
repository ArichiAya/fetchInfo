/**
 * 股票领域的前端计算与展示格式化
 * 后端只返回原始数值（见 docs/api.md），"4.82亿"、"+2.31%" 这类成品文案在前端拼
 */
import type { PositionItem, PositionItemWithValue } from '@/types/stock'

/** 计算持仓市值与盈亏 */
export function calcPositionValue(items: PositionItem[]): PositionItemWithValue[] {
  return items.map((p) => {
    const marketValue = p.shares * p.price
    const profit = (p.price - p.cost) * p.shares
    return {
      ...p,
      marketValue: Number(marketValue.toFixed(2)),
      profit: Number(profit.toFixed(2)),
      profitPercent: Number((((p.price - p.cost) / p.cost) * 100).toFixed(2))
    }
  })
}

/** 成交量/成交额的中文单位，如 482000000 → '4.82亿' */
export function formatAmount(v: number): string {
  const abs = Math.abs(v)
  if (abs >= 1e8) return `${(v / 1e8).toFixed(2)}亿`
  if (abs >= 1e4) return `${(v / 1e4).toFixed(2)}万`
  return v.toFixed(2)
}

/** 金额，如 460220 → '¥460,220' */
export function formatMoney(v: number): string {
  return `¥${v.toLocaleString(undefined, { maximumFractionDigits: 2 })}`
}

/** 带符号的金额，如 26770 → '+¥26,770' */
export function formatSignedMoney(v: number): string {
  return `${v >= 0 ? '+' : '-'}¥${Math.abs(v).toLocaleString(undefined, { maximumFractionDigits: 0 })}`
}

/** 带符号的数值，如 0.29 → '+0.29' */
export function formatSigned(v: number, digits = 2): string {
  return `${v >= 0 ? '+' : ''}${v.toFixed(digits)}`
}

/** 带符号的百分比，如 2.31 → '+2.31%' */
export function formatPercent(v: number, digits = 2): string {
  return `${v >= 0 ? '+' : ''}${v.toFixed(digits)}%`
}

/** 本地日期 YYYY-MM-DD（不用 toISOString，那是 UTC，跨零点会差一天） */
export function todayISO(): string {
  const d = new Date()
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}
