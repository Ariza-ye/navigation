import { useCallback, useEffect, useState } from 'react'

import type { ThemeValue } from '../api/types'
import { normalizedTheme, themeStorageKey } from '../constants/theme'

// 读取本地覆盖主题；异常时退回服务端默认主题，避免隐私模式下报错。
function readStoredTheme(): ThemeValue | null {
  try {
    const value = localStorage.getItem(themeStorageKey)
    return value ? normalizedTheme(value) : null
  } catch {
    return null
  }
}

// 主题来源优先级：本地手动选择 > 后端默认设置 > 深色默认值。
export function useTheme(defaultTheme: ThemeValue = 'dark') {
  const [theme, setTheme] = useState<ThemeValue>(() => readStoredTheme() || defaultTheme)

  useEffect(() => {
    setTheme(readStoredTheme() || normalizedTheme(defaultTheme))
  }, [defaultTheme])

  useEffect(() => {
    document.body.dataset.theme = theme
  }, [theme])

  const saveTheme = useCallback((nextTheme: ThemeValue) => {
    const normalized = normalizedTheme(nextTheme)
    setTheme(normalized)
    try {
      localStorage.setItem(themeStorageKey, normalized)
    } catch {
      // localStorage 不可用时只保留当前会话的主题。
    }
  }, [])

  return { theme, setTheme, saveTheme }
}
