export interface Site {
  id: string
  name: string
  url: string
  category: string
  icon: string
  description: string
  glow: string
  sort: number
  createdAt?: string
  updatedAt?: string
}

export interface AppSettings {
  siteTitle: string
  badge: string
  heroTitle: string
  subtitle: string
  theme: string
}

export interface UserSession {
  username: string
}

export interface Stats {
  siteCount: number
  categoryCount: number
  coverage: string
}

export interface CategoryStat {
  name: string
  count: number
}

export interface Note {
  id: string
  title: string
  filePath: string
  summary: string
  tags: string[]
  status: 'active' | 'archived' | 'deleted'
  pinned: boolean
  createdAt: string
  updatedAt: string
  deletedAt?: string
}

export interface NoteContent extends Note {
  content: string
}

export interface LoginInput {
  username: string
  password: string
}

export interface AccountInput {
  username: string
  currentPassword: string
  newPassword: string
}

export type SiteInput = Pick<Site, 'name' | 'url' | 'category' | 'icon' | 'description' | 'glow' | 'sort'>
export type NoteInput = Pick<NoteContent, 'title' | 'content' | 'tags' | 'status' | 'pinned'>
