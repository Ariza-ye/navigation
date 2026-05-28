import type { AccountInput, AppSettings, CategoryStat, LoginInput, Note, NoteContent, NoteInput, Site, SiteInput, Stats, UserSession } from '@/types/api'

export class APIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'APIError'
    this.status = status
  }
}

type RequestOptions = RequestInit & {
  authPrompt?: boolean
}

async function requestJSON<T>(url: string, options: RequestOptions = {}): Promise<T> {
  const { authPrompt: _authPrompt, headers, ...fetchOptions } = options
  const response = await fetch(url, {
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
    ...fetchOptions,
  })

  if (response.status === 204) return null as T

  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    throw new APIError(data.error || '请求失败', response.status)
  }
  return data as T
}

export function getSession() {
  return requestJSON<UserSession>('/api/session')
}

export function login(input: LoginInput) {
  return requestJSON<UserSession>('/api/login', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function logout() {
  return requestJSON<null>('/api/logout', { method: 'POST' })
}

export function updateAccount(input: AccountInput) {
  return requestJSON<UserSession>('/api/account', {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function getSettings() {
  return requestJSON<AppSettings>('/api/settings')
}

export function updateSettings(input: AppSettings) {
  return requestJSON<AppSettings>('/api/settings', {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function listSites(params: { category?: string; q?: string } = {}) {
  const search = new URLSearchParams()
  if (params.category && params.category !== '全部') search.set('category', params.category)
  if (params.q) search.set('q', params.q)
  const suffix = search.toString()
  return requestJSON<Site[]>(suffix ? `/api/sites?${suffix}` : '/api/sites')
}

export function createSite(input: SiteInput) {
  return requestJSON<Site>('/api/sites', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateSite(id: string, input: SiteInput) {
  return requestJSON<Site>(`/api/sites/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deleteSite(id: string) {
  return requestJSON<null>(`/api/sites/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export function listCategories() {
  return requestJSON<string[]>('/api/categories')
}

export function renameCategory(name: string, nextName: string) {
  return requestJSON<{ name: string; renamedSites: number }>(`/api/categories/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify({ name: nextName }),
  })
}

export function deleteCategory(name: string) {
  return requestJSON<{ uncategorizedSites: number }>(`/api/categories/${encodeURIComponent(name)}`, { method: 'DELETE' })
}

export function getStats() {
  return requestJSON<Stats>('/api/stats')
}

export function getCategoryStats() {
  return requestJSON<CategoryStat[]>('/api/category-stats')
}

export function listNotes(params: { q?: string; status?: string } = {}) {
  const search = new URLSearchParams()
  if (params.q) search.set('q', params.q)
  if (params.status) search.set('status', params.status)
  const suffix = search.toString()
  return requestJSON<Note[]>(suffix ? `/api/notes?${suffix}` : '/api/notes')
}

export function getNote(id: string) {
  return requestJSON<NoteContent>(`/api/notes/${encodeURIComponent(id)}`)
}

export function createNote(input: NoteInput) {
  return requestJSON<NoteContent>('/api/notes', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateNote(id: string, input: NoteInput) {
  return requestJSON<NoteContent>(`/api/notes/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deleteNote(id: string) {
  return requestJSON<null>(`/api/notes/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
