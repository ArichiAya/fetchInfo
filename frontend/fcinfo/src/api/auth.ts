/**
 * 登录 / 注册
 * 契约见 docs/api.md
 *
 * 注意这两个接口的响应体是 gin-jwt 的默认结构，与项目其余接口的
 * `{ error: { code, message } }` 不一致 —— request.ts 里已做兼容。
 */
import { authRequest, setToken } from './request'

export interface LoginResult {
  token: string
  /** 过期时间，RFC3339 */
  expire: string
}

interface LoginResponse {
  code: number
  token: string
  expire: string
}

interface RegisterResponse {
  code: number
}

const jsonPost = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body)
})

/** 登录成功即写入 token，后续请求自动带上 Authorization */
export async function login(username: string, password: string): Promise<LoginResult> {
  const res = await authRequest<LoginResponse>('/login', jsonPost({ username, password }))
  setToken(res.token)
  return { token: res.token, expire: res.expire }
}

/**
 * 注册
 * 不自动登录 —— 后端未约定注册返回 token，成功后让用户再走一次登录。
 * 响应体只有 `{ code: 201 }`，没有可用的载荷。
 */
export async function register(username: string, password: string): Promise<void> {
  await authRequest<RegisterResponse>('/register', jsonPost({ username, password }))
}

/** 退出登录：后端没有 logout 接口，本地清掉 token 即可 */
export function logout() {
  setToken(null)
}
