import { useEffect, useState } from 'react'

// 将高频输入延迟同步到 debounced，避免搜索框每次敲字都立即请求后端。
export function useDebouncedValue<T>(value: T, delay = 250): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay)
    return () => window.clearTimeout(timer)
  }, [value, delay])

  return debounced
}
