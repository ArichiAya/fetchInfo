/**
 * 行情查询接口（只读）
 * 契约见 docs/api.md
 *
 * 所有查询都接受 AbortSignal：轮询与快速切换股票时需要取消上一个未完成的请求
 */
import { marketRequest } from './request'
import type { KLineDay, Snapshot, StockSymbol, TickPoint } from '@/types/stock'

/** 按代码或名称模糊搜索标的（订阅输入框用），V1 不返回现价 */
export const searchStocks = async (
  keyword: string,
  limit = 10,
  signal?: AbortSignal
): Promise<StockSymbol[]> => {
  const query = `keyword=${encodeURIComponent(keyword)}&limit=${limit}`
  const { items } = await marketRequest<{ items: StockSymbol[] }>(`/stocks/search?${query}`, {
    signal
  })
  return items
}

/** 行情快照（KPI 卡片用），建议 5s 轮询 */
export const fetchSnapshot = (code: string, signal?: AbortSignal) =>
  marketRequest<Snapshot>(`/stocks/${encodeURIComponent(code)}/snapshot`, { signal })

/**
 * 分时数据，一个交易日 240 个点
 * @param date 交易日 YYYY-MM-DD，缺省为当日
 */
export const fetchTickData = (code: string, date?: string, signal?: AbortSignal) => {
  const query = date ? `?date=${date}` : ''
  return marketRequest<TickPoint[]>(`/stocks/${encodeURIComponent(code)}/tick${query}`, { signal })
}

/**
 * K 线数据
 * @param period 周期，1m / 5m / 15m / 1d
 * @param limit  返回根数
 */
export const fetchKLineData = (
  code: string,
  period = '1d',
  limit = 30,
  signal?: AbortSignal
) => {
  const query = `period=${period}&limit=${limit}`
  return marketRequest<KLineDay[]>(`/stocks/${encodeURIComponent(code)}/kline?${query}`, {
    signal
  })
}
