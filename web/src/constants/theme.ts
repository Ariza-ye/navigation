import type { ThemeValue } from '../api/types'

// 用户手动切换主题后写入本地存储，用来覆盖后台配置的默认主题。
export const themeStorageKey = 'navigation.theme.override'

// 页面支持的主题列表，swatch 用于主题菜单中的颜色预览点。
export const themeOptions: Array<{ name: string; value: ThemeValue; swatch: string }> = [
  { name: '深空', value: 'dark', swatch: 'oklch(85% .11 205)' },
  { name: '晨光', value: 'morning', swatch: 'oklch(68% .12 78)' },
  { name: '森屿', value: 'forest', swatch: 'oklch(82% .13 132)' },
  { name: '梅雾', value: 'plum', swatch: 'oklch(68% .14 325)' }
]

// 对外部输入做白名单校验，防止无效主题值污染 body[data-theme]。
export function hasTheme(theme: unknown): theme is ThemeValue {
  return themeOptions.some(option => option.value === theme)
}

export function normalizedTheme(theme: unknown): ThemeValue {
  return hasTheme(theme) ? theme : 'dark'
}
