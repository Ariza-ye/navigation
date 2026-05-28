# 前端说明书

`frontend` 是导航站的独立 Vue 3 前端项目，负责站点展示、搜索筛选、分类管理、登录会话、账号设置、页面设置和主题切换。项目通过 Vite 开发和构建，开发环境下把 `/api` 请求代理到后端 Go 服务。

## 技术栈

- Vue 3：页面和组件开发，使用 `<script setup lang="ts">`。
- TypeScript：接口类型、组合式函数和 API 调用类型约束。
- Vite：本地开发服务器、生产构建和预览。
- Tailwind CSS：工具类样式和少量组件层样式。
- lucide-vue-next：按钮、菜单和状态图标。
- class-variance-authority、clsx、tailwind-merge：UI 组件 class 组合。

## 目录结构

```text
frontend/
├── index.html              # Vite HTML 入口
├── package.json            # npm 脚本和依赖
├── vite.config.ts          # Vite 配置，包含 /api 代理和构建输出目录
├── tailwind.config.ts      # Tailwind 主题扩展
├── postcss.config.js       # PostCSS 配置
├── tsconfig.json           # TypeScript 配置
└── src/
    ├── main.ts             # Vue 挂载入口
    ├── App.vue             # 应用主编排组件
    ├── assets/main.css     # 全局样式、主题变量和组件层样式
    ├── components/         # 业务组件和基础 UI 组件
    ├── composables/        # 认证、站点、设置、主题等状态逻辑
    ├── lib/                # API 封装、选项配置、工具函数
    └── types/api.ts        # 前后端共享的数据结构类型
```

## 快速开始

先启动后端服务：

```bash
go run . -port 8080 -data data
```

再启动前端开发服务器：

```bash
cd frontend
npm install
npm run dev
```

Vite 默认会启动在 `http://localhost:5173`。前端访问 `/api/*` 时会通过 `vite.config.ts` 代理到 `http://localhost:8080`。

## 常用命令

```bash
npm run dev
```

启动本地开发服务器，监听 `0.0.0.0`，便于局域网设备访问。

```bash
npm run build
```

先运行 `vue-tsc --noEmit` 做类型检查，再执行 `vite build`。构建产物输出到仓库根目录的 `web/dist`。

```bash
npm run preview
```

预览生产构建结果。

```bash
npm run lint
```

当前 lint 脚本实际执行 TypeScript 类型检查：`vue-tsc --noEmit`。

## 应用功能

### 公开访问

- 展示导航站标题、徽章、简介和统计数据。
- 按分类切换站点列表。
- 在搜索框中按关键字筛选站点。
- 点击站点卡片在新标签页打开目标地址。
- 切换个人主题偏好，主题偏好保存在 `localStorage`。

### 登录后可用

- 新增站点。
- 编辑站点。
- 删除站点。
- 管理分类，包括重命名和删除分类。
- 修改账号信息。
- 修改页面设置，包括页面标题、徽章、主标题、简介和全局默认主题。

登录状态来自后端会话接口。未登录时执行管理操作会打开登录弹窗；遇到 `401` 会清空前端会话并重新要求登录。

## 数据流

`src/App.vue` 是应用主编排层，负责组合认证、站点、设置和主题逻辑：

- 启动时调用 `auth.refreshSession()` 读取当前会话。
- 调用 `settingsStore.loadSettings()` 读取页面设置，并通过 `useTheme()` 应用默认主题。
- 调用 `sites.loadAll()` 并行读取分类、站点和统计数据。
- 搜索关键字变化时，使用 `debounce` 延迟刷新站点列表。
- 保存站点、删除站点、分类变更后重新加载相关数据。

状态逻辑主要放在 `src/composables`：

- `useAuth.ts`：会话读取、登录、登出、账号更新、登录拦截和 `401` 处理。
- `useSites.ts`：站点列表、分类列表、分类统计、站点增删改、分类重命名和删除。
- `useSettings.ts`：页面设置读取与保存，同时同步 `document.title`。
- `useTheme.ts`：主题选项、主题规范化、默认主题应用和本地偏好持久化。

## API 依赖

所有请求统一由 `src/lib/api.ts` 封装，默认携带：

```ts
credentials: 'same-origin'
Content-Type: 'application/json'
```

当前前端依赖以下后端接口：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/session` | 获取当前登录会话 |
| `POST` | `/api/login` | 登录 |
| `POST` | `/api/logout` | 登出 |
| `PUT` | `/api/account` | 更新账号 |
| `GET` | `/api/settings` | 获取页面设置 |
| `PUT` | `/api/settings` | 更新页面设置 |
| `GET` | `/api/sites` | 获取站点列表，支持 `category` 和 `q` 查询参数 |
| `POST` | `/api/sites` | 新增站点 |
| `PUT` | `/api/sites/:id` | 更新站点 |
| `DELETE` | `/api/sites/:id` | 删除站点 |
| `GET` | `/api/categories` | 获取分类列表 |
| `PUT` | `/api/categories/:name` | 重命名分类 |
| `DELETE` | `/api/categories/:name` | 删除分类 |
| `GET` | `/api/stats` | 获取首页统计 |
| `GET` | `/api/category-stats` | 获取分类统计 |

请求失败时，`requestJSON` 会抛出 `APIError`。错误消息优先使用后端返回的 `error` 字段，否则显示 `请求失败`。

## 数据类型

核心类型定义在 `src/types/api.ts`。

站点数据：

```ts
interface Site {
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
```

页面设置：

```ts
interface AppSettings {
  siteTitle: string
  badge: string
  heroTitle: string
  subtitle: string
  theme: string
}
```

## 组件说明

主要业务组件：

- `AppShell.vue`：页面外壳，包含右上角用户菜单和全局主题切换器。
- `HeroSection.vue`：首页标题区、搜索框和统计信息。
- `CategoryTabs.vue`：分类切换。
- `SiteGrid.vue`：站点卡片网格和空状态。
- `SiteCard.vue`：单个站点入口卡片，登录后显示编辑和删除按钮。
- `SiteDialog.vue`：新增和编辑站点表单。
- `CategoryDialog.vue`：分类统计、分类重命名和删除入口。
- `SettingsDialog.vue`：页面设置表单。
- `AccountDialog.vue`：账号修改表单。
- `LoginDialog.vue`：登录弹窗。
- `ThemeSwitcher.vue`：主题切换控件。
- `EmojiDialog.vue`：站点 emoji 图标选择器。

基础 UI 组件放在 `src/components/ui`：

- `Button.vue`
- `Dialog.vue`
- `SelectField.vue`
- `TextArea.vue`
- `TextField.vue`

新增交互时优先复用这些基础组件，保持表单、按钮、弹窗样式一致。

## 主题与样式

全局样式集中在 `src/assets/main.css`。主题通过 CSS 变量实现，根节点 `html[data-theme="..."]` 控制当前主题。

当前内置主题：

- `dark`：深空
- `morning`：晨光
- `forest`：森屿
- `plum`：梅雾

主题选项定义在 `src/composables/useTheme.ts`。如果要新增主题，需要同时：

1. 在 `themeOptions` 中增加选项。
2. 在 `main.css` 中增加对应的 `html[data-theme="新主题值"]` 变量定义。
3. 确认 `SettingsDialog.vue` 的默认主题选择和 `ThemeSwitcher.vue` 的切换逻辑正常显示。

站点卡片的光效颜色定义在 `src/lib/options.ts` 的 `glowOptions`，站点图标候选定义在同文件的 `emojiOptions`。

## 构建产物

`vite.config.ts` 中设置：

```ts
build: {
  outDir: '../web/dist',
  emptyOutDir: true,
}
```

因此在 `frontend` 目录执行 `npm run build` 后，会把静态资源输出到仓库根目录下的 `web/dist`，供后端或部署流程使用。

## 开发约定

- 保持业务逻辑在 `composables`，组件主要负责展示和事件派发。
- 新增后端字段时，先更新 `src/types/api.ts`，再更新 `src/lib/api.ts` 和相关组件。
- 新增 API 调用时统一放入 `src/lib/api.ts`，避免组件直接拼装请求。
- 用户可见文案保持中文。
- 修改样式时优先复用 `main.css` 中的 CSS 变量，避免在组件中硬编码颜色。
- 修改涉及前端行为后，至少运行 `npm run lint`；发布前运行 `npm run build`。

## 常见问题

### 前端请求接口失败

确认后端已经运行在 `http://localhost:8080`，并检查 `vite.config.ts` 中的代理配置：

```ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
  },
}
```

### 页面设置保存后标题没有变化

标题由 `useSettings.ts` 同步到 `document.title`。如果后端返回的 `siteTitle` 不符合预期，优先检查 `/api/settings` 的响应内容。

### 主题切换后刷新仍保留个人主题

这是预期行为。个人主题偏好会写入 `localStorage` 的 `navigation.theme.override`。全局默认主题只在没有本地偏好时生效。
