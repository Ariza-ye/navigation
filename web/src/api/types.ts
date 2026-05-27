export type ThemeValue = 'dark' | 'morning' | 'forest' | 'plum'

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

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}
