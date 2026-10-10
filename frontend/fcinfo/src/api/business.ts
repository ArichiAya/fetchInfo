/**
 * 订阅与持仓接口
 * 契约见 docs/api.md
 */
import { businessRequest, del, postJson } from './request'
import type { PositionItem, SubscriptionItem } from '@/types/stock'

/** 已订阅列表 */
export const fetchSubscriptions = (signal?: AbortSignal) =>
  businessRequest<SubscriptionItem[]>('/subscriptions', { signal })

/**
 * 订阅股票
 * 只传 code —— 数据来源由后端按代码决定，前端不参与选源
 * 失败：404 SYMBOL_NOT_FOUND（代码不存在）、409 ALREADY_SUBSCRIBED（已在列表中）
 */
export const subscribeStock = (code: string) =>
  postJson<SubscriptionItem>('/subscriptions', { code })

/** 取消订阅，失败 404 SUBSCRIPTION_NOT_FOUND */
export const unsubscribeStock = (code: string) =>
  del<void>(`/subscriptions/${encodeURIComponent(code)}`)

/** 持仓列表（现价由后端 join 好，只返回 shares > 0 的行） */
export const fetchPositions = (signal?: AbortSignal) =>
  businessRequest<PositionItem[]>('/positions', { signal })
