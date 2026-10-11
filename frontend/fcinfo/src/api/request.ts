/**
 * 统一请求层：注入服务前缀与 JWT、解析后端统一错误体、把失败统一抛成 ApiRequestError
 * 契约见 docs/api.md
 */
import type { ApiErrorBody } from '@/types/stock'

/** 行情查询服务（analysis/cmd/server） */
export const MARKET_BASE = '/api/market'
/** 订阅与持仓服务（business/cmd/server，尚未实现） */
export const BUSINESS_BASE = '/api/business'

const TOKEN_KEY = 'fcinfo_token'

/** 后端鉴权走 `Authorization: Bearer <token>`（common/auth 的 TokenLookup 配置） */
export const getToken = () => localStorage.getItem(TOKEN_KEY)

export const setToken = (token: string | null) => {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

/** 后端返回的错误，携带 HTTP 状态码与业务错误码，便于调用侧分流提示 */
export class ApiRequestError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = code
  }
}

async function request<T>(base: string, url: string, opts?: RequestInit): Promise<T> {
  const headers = new Headers(opts?.headers)
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(`${base}${url}`, { ...opts, headers })

  // 204 No Content（取消订阅）没有响应体
  if (res.status === 204) return undefined as T

  const body = await res.json().catch(() => null)

  if (!res.ok) {
    // token 失效就丢掉，避免拿着废 token 反复 401
    if (res.status === 401) setToken(null)

    const err = (body as ApiErrorBody | null)?.error
    throw new ApiRequestError(
      res.status,
      err?.code ?? 'UNKNOWN',
      err?.message ?? `请求失败（HTTP ${res.status}）`
    )
  }

  return body as T
}

/** 查询行情（GET 为主，只读） */
export const marketRequest = <T>(url: string, opts?: RequestInit) =>
  request<T>(MARKET_BASE, url, opts)

/** 订阅与持仓（含写操作） */
export const businessRequest = <T>(url: string, opts?: RequestInit) =>
  request<T>(BUSINESS_BASE, url, opts)

/** POST JSON，自动带上 Content-Type */
export const postJson = <T>(url: string, data: unknown, opts?: RequestInit) =>
  businessRequest<T>(url, {
    ...opts,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    body: JSON.stringify(data)
  })

/** DELETE，用 business 前缀 */
export const del = <T>(url: string, opts?: RequestInit) =>
  businessRequest<T>(url, { ...opts, method: 'DELETE' })
