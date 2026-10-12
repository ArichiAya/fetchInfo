/**
 * 统一请求层：注入服务前缀与 JWT、解析后端错误体、把失败统一抛成 ApiRequestError
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

/**
 * token 失效时的处理由外层注册。
 * 不在这里直接 import router —— 那会让请求层和路由互相依赖。
 */
let onUnauthorized: (() => void) | null = null

export const setUnauthorizedHandler = (fn: () => void) => {
  onUnauthorized = fn
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

interface RequestOptions extends RequestInit {
  /**
   * 登录/注册返回的 401 是"密码错误"，不是"登录态失效"，
   * 不该触发全局的跳登录页（用户本来就在登录页上）
   */
  skipUnauthorizedHandler?: boolean
}

/**
 * 后端目前有两套错误体，这里都认：
 * - 项目约定 `{ "error": { "code": "...", "message": "..." } }`
 * - gin-jwt 默认 `{ "code": 401, "message": "..." }`（登录/注册）
 */
function parseError(body: unknown): { code?: string | number; message?: string } {
  const wrapped = (body as ApiErrorBody | null)?.error
  if (wrapped) return wrapped
  const flat = body as { code?: string | number; message?: string } | null
  return { code: flat?.code, message: flat?.message }
}

async function request<T>(base: string, url: string, opts?: RequestOptions): Promise<T> {
  const { skipUnauthorizedHandler, ...init } = opts ?? {}

  const headers = new Headers(init.headers)
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(`${base}${url}`, { ...init, headers })

  // 204 No Content（取消订阅）没有响应体
  if (res.status === 204) return undefined as T

  const body = await res.json().catch(() => null)

  if (!res.ok) {
    // token 失效就丢掉，避免拿着废 token 反复 401
    if (res.status === 401) {
      setToken(null)
      if (!skipUnauthorizedHandler) onUnauthorized?.()
    }

    const err = parseError(body)
    throw new ApiRequestError(
      res.status,
      err.code === undefined ? 'UNKNOWN' : String(err.code),
      err.message ?? `请求失败（HTTP ${res.status}）`
    )
  }

  return body as T
}

/** 查询行情（GET 为主，只读） */
export const marketRequest = <T>(url: string, opts?: RequestOptions) =>
  request<T>(MARKET_BASE, url, opts)

/** 订阅与持仓（含写操作） */
export const businessRequest = <T>(url: string, opts?: RequestOptions) =>
  request<T>(BUSINESS_BASE, url, opts)

/**
 * 登录/注册：这两个接口挂在根路径下（不在 /api 前缀里），vite 按 /login、/register 分别代理。
 * 它们的 401 是凭据错误而非登录态失效，所以跳过全局的跳登录处理。
 */
export const authRequest = <T>(url: string, opts?: RequestOptions) =>
  request<T>('', url, { ...opts, skipUnauthorizedHandler: true })

/** POST JSON，自动带上 Content-Type */
export const postJson = <T>(url: string, data: unknown, opts?: RequestOptions) =>
  businessRequest<T>(url, {
    ...opts,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    body: JSON.stringify(data)
  })

/** DELETE，用 business 前缀 */
export const del = <T>(url: string, opts?: RequestOptions) =>
  businessRequest<T>(url, { ...opts, method: 'DELETE' })
