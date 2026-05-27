import { ApiError } from './types'

interface ErrorPayload {
  error?: string
}

// 统一封装前端到后端 API 的 JSON 请求，集中处理凭证、响应解析和错误格式。
export async function requestJSON<T>(
  url: string,
  options: RequestInit & { authPrompt?: boolean } = {}
): Promise<T> {
  const { authPrompt = true, headers, ...fetchOptions } = options
  // authPrompt 当前由调用方传入用于表达“是否需要登录提示”的语义，实际弹窗逻辑在 App 中统一处理。
  void authPrompt
  const response = await fetch(url, {
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      ...headers
    },
    ...fetchOptions
  })

  if (response.status === 204) return null as T

  let data: unknown = null
  try {
    data = await response.json()
  } catch {
    // 兼容空响应或非 JSON 错误页，后续会使用默认错误文案。
    data = null
  }

  if (!response.ok) {
    const payload = data as ErrorPayload | null
    throw new ApiError(payload?.error || '请求失败', response.status)
  }

  return data as T
}
