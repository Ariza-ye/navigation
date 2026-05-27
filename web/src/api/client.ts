import { ApiError } from './types'

interface ErrorPayload {
  error?: string
}

export async function requestJSON<T>(
  url: string,
  options: RequestInit & { authPrompt?: boolean } = {}
): Promise<T> {
  const { authPrompt = true, headers, ...fetchOptions } = options
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
    data = null
  }

  if (!response.ok) {
    const payload = data as ErrorPayload | null
    throw new ApiError(payload?.error || '请求失败', response.status)
  }

  return data as T
}
