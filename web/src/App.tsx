import { FormEvent, MouseEvent, ReactNode, CSSProperties, useCallback, useEffect, useMemo, useState } from 'react'

import { requestJSON } from './api/client'
import { ApiError, AppSettings, CategoryStat, Site, SiteInput, Stats, User } from './api/types'
import { emojiOptions } from './constants/emoji'
import { glowOptions } from './constants/glow'
import { normalizedTheme, themeOptions } from './constants/theme'
import { useDebouncedValue } from './hooks/useDebouncedValue'
import { useTheme } from './hooks/useTheme'

type DialogName = null | 'site' | 'category' | 'emoji' | 'account' | 'settings'

interface SiteFormState {
  id: string
  name: string
  category: string
  url: string
  icon: string
  sort: string
  glow: string
  description: string
}

const emptySettings: AppSettings = {
  siteTitle: '导航站',
  badge: 'DEV PORTAL / 个人导航站',
  subtitle: '聚合了常用网站',
  heroTitle: '常用站点导航',
  theme: 'dark'
}

const emptySiteForm: SiteFormState = {
  id: '',
  name: '',
  category: '',
  url: '',
  icon: emojiOptions[0],
  sort: '',
  glow: glowOptions[0].value,
  description: ''
}

function styleVars(vars: Record<string, string>) {
  return vars as CSSProperties
}

function isAuthError(error: unknown) {
  return error instanceof ApiError && error.status === 401
}

export default function App() {
  const [sites, setSites] = useState<Site[]>([])
  const [categories, setCategories] = useState<string[]>([])
  const [category, setCategory] = useState('全部')
  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query)
  const [stats, setStats] = useState<Stats>({ siteCount: 0, categoryCount: 0, coverage: '99%' })
  const [settings, setSettings] = useState<AppSettings>(emptySettings)
  const [user, setUser] = useState<User | null>(null)
  const [activeDialog, setActiveDialog] = useState<DialogName>(null)
  const [editingSite, setEditingSite] = useState<Site | null>(null)
  const [siteForm, setSiteForm] = useState<SiteFormState>(emptySiteForm)
  const [categoryStats, setCategoryStats] = useState<CategoryStat[]>([])
  const [categoryLoading, setCategoryLoading] = useState(false)
  const [categoryMenuOpen, setCategoryMenuOpen] = useState(false)
  const [themeMenuOpen, setThemeMenuOpen] = useState(false)
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const [loginVisible, setLoginVisible] = useState(false)
  const [loginError, setLoginError] = useState('')
  const [formError, setFormError] = useState('')
  const [accountError, setAccountError] = useState('')
  const [settingsError, setSettingsError] = useState('')
  const [loadError, setLoadError] = useState('')
  const { theme, saveTheme } = useTheme(settings.theme)

  const currentTheme = useMemo(
    () => themeOptions.find(option => option.value === theme) || themeOptions[0],
    [theme]
  )

  const showLogin = useCallback(() => {
    setUser(null)
    setUserMenuOpen(false)
    setActiveDialog(null)
    setLoginError('')
    setLoginVisible(true)
    document.body.dataset.theme = theme
  }, [theme])

  const siteQueryURL = useCallback((nextCategory: string, nextQuery: string) => {
    const params = new URLSearchParams()
    if (nextCategory && nextCategory !== '全部') params.set('category', nextCategory)
    if (nextQuery.trim()) params.set('q', nextQuery.trim())
    const suffix = params.toString()
    return suffix ? `/api/sites?${suffix}` : '/api/sites'
  }, [])

  const handleRequestError = useCallback((error: unknown) => {
    if (isAuthError(error)) {
      showLogin()
    }
    throw error
  }, [showLogin])

  const loadSitesOnly = useCallback(async (nextCategory = category, nextQuery = debouncedQuery) => {
    try {
      const data = await requestJSON<Site[]>(siteQueryURL(nextCategory, nextQuery))
      setSites(data)
      setLoadError('')
    } catch (error) {
      handleRequestError(error)
    }
  }, [category, debouncedQuery, handleRequestError, siteQueryURL])

  const loadAll = useCallback(async (nextCategory = category, nextQuery = debouncedQuery) => {
    try {
      const [nextCategories, nextSites, nextStats, nextSettings] = await Promise.all([
        requestJSON<string[]>('/api/categories'),
        requestJSON<Site[]>(siteQueryURL(nextCategory, nextQuery)),
        requestJSON<Stats>('/api/stats'),
        requestJSON<AppSettings>('/api/settings')
      ])
      setCategories(nextCategories)
      setSites(nextSites)
      setStats(nextStats)
      setSettings(nextSettings)
      setLoadError('')
    } catch (error) {
      handleRequestError(error)
    }
  }, [category, debouncedQuery, handleRequestError, siteQueryURL])

  useEffect(() => {
    document.title = settings.siteTitle
  }, [settings.siteTitle])

  useEffect(() => {
    requestJSON<User>('/api/session', { authPrompt: false })
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => {
        loadAll().catch(error => setLoadError(error instanceof Error ? error.message : '加载失败'))
      })
  }, [])

  useEffect(() => {
    loadSitesOnly(category, debouncedQuery).catch(error => {
      setLoadError(error instanceof Error ? error.message : '加载失败')
    })
  }, [category, debouncedQuery])

  function closeDialog() {
    setActiveDialog(null)
    setFormError('')
    setAccountError('')
    setSettingsError('')
    setCategoryMenuOpen(false)
  }

  function openSiteDialog(site: Site | null = null) {
    if (!user) {
      setActiveDialog(null)
      showLogin()
      return
    }
    setEditingSite(site)
    setSiteForm(site ? {
      id: site.id,
      name: site.name,
      category: site.category,
      url: site.url,
      icon: emojiOptions.includes(site.icon) ? site.icon : emojiOptions[0],
      sort: site.sort ? String(site.sort) : '',
      glow: glowOptions.some(option => option.value === site.glow) ? site.glow : glowOptions[0].value,
      description: site.description
    } : emptySiteForm)
    setFormError('')
    setActiveDialog('site')
  }

  async function openCategoryDialog() {
    if (!user) {
      showLogin()
      return
    }
    setCategoryLoading(true)
    setActiveDialog('category')
    try {
      setCategoryStats(await requestJSON<CategoryStat[]>('/api/category-stats'))
    } catch (error) {
      if (isAuthError(error)) showLogin()
      setCategoryStats([])
    } finally {
      setCategoryLoading(false)
    }
  }

  async function login(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoginError('')
    const form = new FormData(event.currentTarget)
    try {
      const nextUser = await requestJSON<User>('/api/login', {
        method: 'POST',
        body: JSON.stringify({
          username: String(form.get('username') || ''),
          password: String(form.get('password') || '')
        })
      })
      setUser(nextUser)
      setLoginVisible(false)
      setActiveDialog(null)
      await loadAll()
    } catch (error) {
      setLoginError(error instanceof Error ? error.message : '登录失败')
    }
  }

  async function logout() {
    await requestJSON<null>('/api/logout', { method: 'POST' }).catch(() => null)
    setUser(null)
    setUserMenuOpen(false)
  }

  async function saveSite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!user) {
      showLogin()
      return
    }
    setFormError('')
    const payload: SiteInput = {
      name: siteForm.name,
      category: siteForm.category,
      url: siteForm.url,
      icon: siteForm.icon,
      sort: Number(siteForm.sort || 0),
      glow: siteForm.glow,
      description: siteForm.description
    }
    try {
      await requestJSON<Site>(siteForm.id ? `/api/sites/${siteForm.id}` : '/api/sites', {
        method: siteForm.id ? 'PUT' : 'POST',
        body: JSON.stringify(payload)
      })
      closeDialog()
      await loadAll()
    } catch (error) {
      if (isAuthError(error)) showLogin()
      setFormError(error instanceof Error ? error.message : '保存失败')
    }
  }

  async function deleteSite(site: Site) {
    if (!user) {
      showLogin()
      return
    }
    if (!confirm(`确定删除「${site.name}」吗？`)) return
    try {
      await requestJSON<null>(`/api/sites/${site.id}`, { method: 'DELETE' })
      await loadAll()
    } catch (error) {
      if (isAuthError(error)) showLogin()
      else alert(error instanceof Error ? error.message : '删除失败')
    }
  }

  async function renameCategory(stat: CategoryStat) {
    const nextName = prompt('请输入新的分类名称', stat.name)
    if (nextName === null) return
    const normalizedName = nextName.trim()
    if (!normalizedName || normalizedName === stat.name) return
    try {
      const result = await requestJSON<{ name: string; renamedSites: number }>(
        `/api/categories/${encodeURIComponent(stat.name)}`,
        { method: 'PUT', body: JSON.stringify({ name: normalizedName }) }
      )
      const nextCategory = category === stat.name ? result.name : category
      if (category === stat.name) setCategory(result.name)
      await loadAll(nextCategory)
      await openCategoryDialog()
    } catch (error) {
      if (isAuthError(error)) showLogin()
      else alert(error instanceof Error ? error.message : '保存失败')
    }
  }

  async function deleteCategory(stat: CategoryStat) {
    const message = `确定删除「${stat.name}」分类吗？该分类下的 ${stat.count} 个站点会保留，但分类会被清空。`
    if (!confirm(message)) return
    try {
      await requestJSON<{ uncategorizedSites: number }>(`/api/categories/${encodeURIComponent(stat.name)}`, {
        method: 'DELETE'
      })
      if (category === stat.name) setCategory('全部')
      await loadAll(category === stat.name ? '全部' : category)
      await openCategoryDialog()
    } catch (error) {
      if (isAuthError(error)) showLogin()
      else alert(error instanceof Error ? error.message : '删除失败')
    }
  }

  async function saveAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAccountError('')
    const form = new FormData(event.currentTarget)
    try {
      const nextUser = await requestJSON<User>('/api/account', {
        method: 'PUT',
        body: JSON.stringify({
          username: String(form.get('username') || ''),
          currentPassword: String(form.get('currentPassword') || ''),
          newPassword: String(form.get('newPassword') || '')
        })
      })
      setUser(nextUser)
      closeDialog()
    } catch (error) {
      if (isAuthError(error)) showLogin()
      setAccountError(error instanceof Error ? error.message : '保存失败')
    }
  }

  async function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSettingsError('')
    const form = new FormData(event.currentTarget)
    try {
      const nextSettings = await requestJSON<AppSettings>('/api/settings', {
        method: 'PUT',
        body: JSON.stringify({
          siteTitle: String(form.get('siteTitle') || ''),
          badge: String(form.get('badge') || ''),
          heroTitle: String(form.get('heroTitle') || ''),
          subtitle: String(form.get('subtitle') || ''),
          theme: normalizedTheme(form.get('theme'))
        })
      })
      setSettings(nextSettings)
      closeDialog()
    } catch (error) {
      if (isAuthError(error)) showLogin()
      setSettingsError(error instanceof Error ? error.message : '保存失败')
    }
  }

  function updateSiteForm(value: Partial<SiteFormState>) {
    setSiteForm(previous => ({ ...previous, ...value }))
  }

  function stopCardAction(event: MouseEvent<HTMLButtonElement>) {
    event.preventDefault()
    event.stopPropagation()
  }

  return (
    <>
      <main className="wrap">
        <div className="topbar">
          <div className="user-menu" id="userMenu">
            <button
              className="user-btn"
              type="button"
              onClick={() => user ? setUserMenuOpen(open => !open) : showLogin()}
            >
              <span>{user ? user.username : '登录'}</span> ▾
            </button>
            <div className={`user-dropdown${userMenuOpen ? ' open' : ''}`}>
              <button className="menu-item" type="button" onClick={() => { setUserMenuOpen(false); setActiveDialog('settings') }}>设置</button>
              <button className="menu-item" type="button" onClick={() => { setUserMenuOpen(false); setActiveDialog('account') }}>修改账号密码</button>
              <button className="menu-item" type="button" onClick={logout}>退出登录</button>
            </div>
          </div>
        </div>

        <section className="hero">
          <div>
            <div className="badge"><span className="badge-dot"></span> <span>{settings.badge}</span></div>
            <h1><strong>{settings.heroTitle}</strong></h1>
            <p className="sub">{settings.subtitle}</p>
          </div>
          <div className="panel">
            <div className="search">🔎 <input value={query} onChange={event => setQuery(event.target.value)} placeholder="搜索工具、文档、服务..." /></div>
            <div className="stats">
              <div className="stat"><b>{stats.siteCount}</b><small>常用站点</small></div>
              <div className="stat"><b>{stats.categoryCount}</b><small>分类分组</small></div>
              <div className="stat"><b>{stats.coverage}</b><small>日常覆盖</small></div>
            </div>
          </div>
        </section>

        <nav className="tabs">
          {categories.map(item => (
            <button
              key={item}
              className={`tab${item === category ? ' active' : ''}`}
              type="button"
              onClick={() => setCategory(item)}
            >
              {item}
            </button>
          ))}
        </nav>

        <div className="section-title">
          <h2>常用入口</h2>
          <div className="title-actions">
            <span>点击卡片快速访问</span>
            {user && <button className="ghost-btn" type="button" onClick={openCategoryDialog}>分类管理</button>}
            {user && <button className="primary-btn" type="button" onClick={() => openSiteDialog()}>+ 新增站点</button>}
          </div>
        </div>
        <section className="grid">
          {loadError ? <div className="empty">{loadError}</div> : null}
          {!loadError && sites.length === 0 ? <div className="empty">没有找到匹配的站点</div> : null}
          {!loadError && sites.map(site => (
            <a
              key={site.id}
              className="card"
              href={site.url}
              target="_blank"
              rel="noopener noreferrer"
              style={styleVars({ '--glow': site.glow || glowOptions[0].value })}
            >
              {user && (
                <div className="card-actions">
                  <button className="icon-btn" type="button" title="编辑" onClick={event => { stopCardAction(event); openSiteDialog(site) }}>✎</button>
                  <button className="icon-btn" type="button" title="删除" onClick={event => { stopCardAction(event); deleteSite(site) }}>×</button>
                </div>
              )}
              <div className="icon">{site.icon || '🔗'}</div>
              <h3>{site.name}</h3>
              <p>{site.description || site.category}</p>
              <span className="arrow">→</span>
            </a>
          ))}
        </section>
      </main>

      <div className="theme-switcher" id="themeSwitcher">
        <div className={`theme-menu${themeMenuOpen ? ' open' : ''}`}>
          {themeOptions.map(option => (
            <button
              key={option.value}
              className={`theme-option${option.value === theme ? ' active' : ''}`}
              type="button"
              style={styleVars({ '--theme-swatch': option.swatch })}
              onClick={() => { setThemeMenuOpen(false); saveTheme(option.value) }}
            >
              <span className="theme-option-label"><span className="theme-swatch"></span><span>{option.name}</span></span>
              <span className="theme-check">✓</span>
            </button>
          ))}
        </div>
        <button className="theme-toggle" type="button" aria-haspopup="true" aria-expanded={themeMenuOpen} onClick={() => setThemeMenuOpen(open => !open)}>
          <span className="theme-toggle-dot"></span>
          <span>主题：{currentTheme.name}</span>
          <span>▴</span>
        </button>
      </div>

      <div className={`auth-screen${loginVisible ? '' : ' hidden'}`}>
        <div className="auth-card">
          <h2>登录导航站</h2>
          <p>请输入管理员账号密码</p>
          <form className="auth-form" onSubmit={login}>
            <div className="field">
              <label htmlFor="loginUsername">账号</label>
              <input id="loginUsername" name="username" autoComplete="username" required />
            </div>
            <div className="field">
              <label htmlFor="loginPassword">密码</label>
              <input id="loginPassword" name="password" type="password" autoComplete="current-password" required />
            </div>
            <div className="error">{loginError}</div>
            <button className="primary-btn" type="submit">登录</button>
            <button className="ghost-btn" type="button" onClick={() => setLoginVisible(false)}>暂不登录，继续浏览</button>
          </form>
        </div>
      </div>

      <Dialog open={activeDialog === 'site'} title={editingSite ? '编辑站点' : '新增站点'} onClose={closeDialog}>
        <form onSubmit={saveSite}>
          <div className="form-grid">
            <Field label="名称" htmlFor="name"><input id="name" value={siteForm.name} onChange={event => updateSiteForm({ name: event.target.value })} required /></Field>
            <Field label="分类" htmlFor="category">
              <div className="category-picker" id="categoryPicker">
                <div className="category-input-wrap">
                  <input
                    id="category"
                    value={siteForm.category}
                    onFocus={() => setCategoryMenuOpen(true)}
                    onChange={event => { updateSiteForm({ category: event.target.value }); setCategoryMenuOpen(true) }}
                    autoComplete="off"
                    required
                  />
                  <button className="category-toggle" type="button" aria-label="展开分类" onClick={() => setCategoryMenuOpen(open => !open)}>🔽</button>
                </div>
                <div className={`category-menu${categoryMenuOpen ? ' open' : ''}`}>
                  {categories.filter(item => item !== '全部').filter(item => !siteForm.category || item.toLowerCase().includes(siteForm.category.toLowerCase())).map(item => (
                    <button key={item} className={`category-option${item === siteForm.category ? ' active' : ''}`} type="button" onMouseDown={event => event.preventDefault()} onClick={() => { updateSiteForm({ category: item }); setCategoryMenuOpen(false) }}>
                      <span>{item}</span><small>已有</small>
                    </button>
                  ))}
                  {!categories.filter(item => item !== '全部').filter(item => !siteForm.category || item.toLowerCase().includes(siteForm.category.toLowerCase())).length && (
                    <div className="category-empty">{siteForm.category ? '没有匹配分类，保存后会创建为新分类。' : '暂无可选分类，可以直接输入新分类。'}</div>
                  )}
                </div>
              </div>
            </Field>
            <Field label="地址" htmlFor="url" full><input id="url" type="url" value={siteForm.url} onChange={event => updateSiteForm({ url: event.target.value })} placeholder="https://example.com" required /></Field>
            <Field label="图标" htmlFor="icon">
              <button className="emoji-select" type="button" onClick={() => setActiveDialog('emoji')}>
                <span className="emoji-preview">{siteForm.icon}</span>
                <span className="emoji-select-text">选择 emoji 图标</span>
                <span>›</span>
              </button>
            </Field>
            <Field label="排序" htmlFor="sort"><input id="sort" type="number" min="1" value={siteForm.sort} onChange={event => updateSiteForm({ sort: event.target.value })} /></Field>
            <Field label="光效颜色" htmlFor="glow" full>
              <div className="glow-picker">
                {glowOptions.map(option => (
                  <button key={option.value} className={`glow-option${option.value === siteForm.glow ? ' active' : ''}`} type="button" style={styleVars({ '--swatch': option.value })} onClick={() => updateSiteForm({ glow: option.value })}>
                    {option.name}
                  </button>
                ))}
              </div>
            </Field>
            <Field label="描述" htmlFor="description" full><textarea id="description" value={siteForm.description} onChange={event => updateSiteForm({ description: event.target.value })}></textarea></Field>
          </div>
          <div className="form-actions">
            <div className="error">{formError}</div>
            <div>
              <button className="ghost-btn" type="button" onClick={closeDialog}>取消</button>
              <button className="primary-btn" type="submit">保存</button>
            </div>
          </div>
        </form>
      </Dialog>

      <Dialog open={activeDialog === 'emoji'} title="选择图标" onClose={() => setActiveDialog('site')}>
        <div className="emoji-grid">
          {emojiOptions.map(icon => (
            <button key={icon} className={`emoji-option${icon === siteForm.icon ? ' active' : ''}`} type="button" title={icon} onClick={() => { updateSiteForm({ icon }); setActiveDialog('site') }}>
              {icon}
            </button>
          ))}
        </div>
      </Dialog>

      <Dialog open={activeDialog === 'category'} title="分类管理" onClose={closeDialog}>
        <div className="category-list">
          {categoryLoading ? <div className="empty">正在加载分类...</div> : null}
          {!categoryLoading && !categoryStats.length ? <div className="empty">暂无分类</div> : null}
          {!categoryLoading && categoryStats.map(stat => (
            <div className="category-row" key={stat.name}>
              <div className="category-info"><b>{stat.name}</b><small>{stat.count} 个站点</small></div>
              <div className="category-actions">
                <button className="ghost-btn" type="button" onClick={() => renameCategory(stat)}>编辑</button>
                <button className="danger-btn" type="button" onClick={() => deleteCategory(stat)}>删除</button>
              </div>
            </div>
          ))}
        </div>
      </Dialog>

      <Dialog open={activeDialog === 'account'} title="修改账号密码" onClose={closeDialog}>
        <form key={user?.username || 'anonymous'} onSubmit={saveAccount}>
          <div className="form-grid">
            <Field label="账号" htmlFor="accountUsername" full><input id="accountUsername" name="username" defaultValue={user?.username || ''} required /></Field>
            <Field label="当前密码" htmlFor="currentPassword" full><input id="currentPassword" name="currentPassword" type="password" autoComplete="current-password" required /></Field>
            <Field label="新密码" htmlFor="newPassword" full><input id="newPassword" name="newPassword" type="password" autoComplete="new-password" placeholder="留空则只修改账号" /></Field>
          </div>
          <div className="form-actions">
            <div className="error">{accountError}</div>
            <button className="primary-btn" type="submit">保存</button>
          </div>
        </form>
      </Dialog>

      <Dialog open={activeDialog === 'settings'} title="页面设置" onClose={closeDialog}>
        <form key={`${settings.siteTitle}-${settings.badge}-${settings.heroTitle}-${settings.subtitle}-${settings.theme}`} onSubmit={saveSettings}>
          <div className="form-grid">
            <Field label="Title" htmlFor="siteTitleInput" full><input id="siteTitleInput" name="siteTitle" defaultValue={settings.siteTitle} required /></Field>
            <Field label="徽章" htmlFor="badgeInput" full><input id="badgeInput" name="badge" defaultValue={settings.badge} required /></Field>
            <Field label="主标题" htmlFor="heroTitleInput" full><input id="heroTitleInput" name="heroTitle" defaultValue={settings.heroTitle} required /></Field>
            <Field label="简介" htmlFor="subtitleInput" full><textarea id="subtitleInput" name="subtitle" defaultValue={settings.subtitle} required></textarea></Field>
            <Field label="全局默认主题" htmlFor="defaultThemeInput" full>
              <select id="defaultThemeInput" name="theme" defaultValue={normalizedTheme(settings.theme)} required>
                {themeOptions.map(option => <option key={option.value} value={option.value}>{option.name}</option>)}
              </select>
            </Field>
          </div>
          <div className="form-actions">
            <div className="error">{settingsError}</div>
            <button className="primary-btn" type="submit">保存</button>
          </div>
        </form>
      </Dialog>
    </>
  )
}

function Dialog({ open, title, onClose, children }: { open: boolean; title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className={`dialog-backdrop${open ? ' open' : ''}`} aria-hidden={!open} onClick={event => {
      if (event.target === event.currentTarget) onClose()
    }}>
      <div className="dialog" role="dialog" aria-modal="true" aria-labelledby={`${title}-title`}>
        <div className="dialog-head">
          <h2 id={`${title}-title`}>{title}</h2>
          <button className="close-btn" type="button" onClick={onClose}>×</button>
        </div>
        {children}
      </div>
    </div>
  )
}

function Field({ label, htmlFor, full = false, children }: { label: string; htmlFor: string; full?: boolean; children: ReactNode }) {
  return (
    <div className={`field${full ? ' full' : ''}`}>
      <label htmlFor={htmlFor}>{label}</label>
      {children}
    </div>
  )
}
