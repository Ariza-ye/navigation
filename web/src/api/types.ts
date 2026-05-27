export type ThemeValue = 'dark' | 'morning' | 'forest' | 'plum'

// 后端站点实体，字段名保持和 Go API 返回的 JSON 一致。
export interface Site {
  id: string
  name: string
  url: string
  category: string
  icon: string
  description: string
  glow: string
  sort: number
  createdAt: string
  updatedAt: string
}

// 新增/编辑站点时不需要提交服务端生成的 ID 和时间戳。
export type SiteInput = Omit<Site, 'id' | 'createdAt' | 'updatedAt'>

export interface CategoryStat {
  name: string
  count: number
}

export interface Stats {
  siteCount: number
  categoryCount: number
  coverage: string
}

export interface AppSettings {
  siteTitle: string
  badge: string
  subtitle: string
  heroTitle: string
  theme: ThemeValue
}

export interface User {
  username: string
}

// 保留 HTTP 状态码，方便页面区分未登录和普通业务错误。
export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}
