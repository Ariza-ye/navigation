import type { ThemeValue } from '../api/types'

export const themeStorageKey = 'navigation.theme.override'

export const themeOptions: Array<{ name: string; value: ThemeValue; swatch: string }> = [
  { name: '深空', value: 'dark', swatch: 'oklch(85% .11 205)' },
  { name: '晨光', value: 'morning', swatch: 'oklch(68% .12 78)' },
  { name: '森屿', value: 'forest', swatch: 'oklch(82% .13 132)' },
  { name: '梅雾', value: 'plum', swatch: 'oklch(68% .14 325)' }
]

export function hasTheme(theme: unknown): theme is ThemeValue {
  return themeOptions.some(option => option.value === theme)
}

export function normalizedTheme(theme: unknown): ThemeValue {
  return hasTheme(theme) ? theme : 'dark'
}
