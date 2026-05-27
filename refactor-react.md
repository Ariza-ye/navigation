# 前端 React 改造方案

## 1. 背景与目标

当前项目是一个 Go 1.22 + SQLite 的个人导航站，前端由 `index.html`、`static/css/app.css`、`static/js/app.js` 组成，并通过 `main.go` 中的 `//go:embed index.html static/*` 打包进二进制。

现有前端主要特点：

- 页面结构写在 `index.html`。
- 交互逻辑集中在 `static/js/app.js`，通过 DOM 查询、事件绑定和 `innerHTML` 渲染。
- 样式集中在 `static/css/app.css`，包含主题变量、卡片、弹窗、表单、分类管理、登录遮罩等样式。
- 后端 API 已经比较清晰，React 改造不需要重写业务接口。

改造目标：

- 使用 React 接管前端渲染和状态管理。
- 保持现有 Go 后端 API、SQLite 存储和认证 Cookie 机制不变。
- 保持当前页面视觉风格、功能和移动端适配。
- 引入可维护的前端工程结构，支持本地开发热更新和生产构建。
- 让 Go 二进制继续内嵌前端构建产物，部署方式尽量不变。

## 2. 推荐技术选型

推荐使用 Vite + React + TypeScript。

原因：

- Vite 配置轻量，适合当前小型项目。
- React 负责组件化和状态驱动渲染，能替代当前大量手写 DOM 操作。
- TypeScript 可以约束站点、分类、设置、会话等 API 数据结构，降低后续维护成本。
- 构建产物是普通静态文件，方便继续由 Go `embed.FS` 打包和 `http.FileServerFS` 提供。

建议依赖：

```text
react
react-dom
@vitejs/plugin-react
typescript
vite
```

暂不建议引入大型 UI 框架或全局状态库。当前业务体量用 React 内置状态、自定义 hooks 和少量上下文即可。

## 3. 目标目录结构

建议将前端源码独立放到 `web/`，构建产物输出到 `web/dist/`。

```text
.
├── main.go
├── internal/
├── data/
├── web/
│   ├── index.html
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── api/
│       │   ├── client.ts
│       │   └── types.ts
│       ├── constants/
│       │   ├── emoji.ts
│       │   ├── glow.ts
│       │   └── theme.ts
│       ├── hooks/
│       │   ├── useDebouncedValue.ts
│       │   ├── useSession.ts
│       │   └── useTheme.ts
│       ├── components/
│       │   ├── AccountDialog.tsx
│       │   ├── AuthScreen.tsx
│       │   ├── CategoryDialog.tsx
│       │   ├── CategoryPicker.tsx
│       │   ├── EmojiDialog.tsx
│       │   ├── HeaderUserMenu.tsx
│       │   ├── Hero.tsx
│       │   ├── SettingsDialog.tsx
│       │   ├── SiteDialog.tsx
│       │   ├── SiteGrid.tsx
│       │   ├── Tabs.tsx
│       │   └── ThemeSwitcher.tsx
│       └── styles/
│           └── app.css
├── build.sh
├── Dockerfile
└── README.md
```

迁移完成后可以删除根目录旧的 `index.html`、`static/js/app.js`、`static/css/app.css`，但建议在最终验收通过后再删除。

## 4. Go 后端改造

### 4.1 静态文件嵌入路径

当前：

```go
//go:embed index.html static/*
var staticFiles embed.FS
```

建议改为嵌入 React 生产构建产物：

```go
//go:embed web/dist/*
var staticFiles embed.FS
```

由于 `web/dist` 会成为嵌入路径的一部分，HTTP 层建议使用 `fs.Sub` 得到真正的静态根目录：

```go
dist, err := fs.Sub(staticFiles, "web/dist")
if err != nil {
    log.Fatalf("加载前端资源失败: %v", err)
}
handler := httptransport.NewHandler(svc, authSvc, dist)
```

需要在 `main.go` 增加 `io/fs` import。

### 4.2 静态资源路由

Vite 默认会把资源输出到 `assets/`，例如 `/assets/index-xxx.js`。当前后端只注册了：

```go
mux.Handle("/static/", http.FileServerFS(h.static))
```

建议改为：

```go
mux.Handle("/assets/", http.FileServerFS(h.static))
```

也可以更通用地保留静态文件服务能力：

```go
mux.Handle("/assets/", http.FileServerFS(h.static))
```

如果后续需要 favicon 或 manifest，`serveIndex` 也应允许这些根路径静态文件被直接读取。

### 4.3 SPA 回退策略

如果 React 不引入前端路由，仅保留单页首页，则当前 `serveIndex` 只服务 `/` 和 `/index.html` 也可以。

如果未来引入 React Router，例如 `/sites`、`/settings`，则建议把非 `/api/`、非 `/assets/` 的路径统一回退到 `index.html`：

```go
func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
    if strings.HasPrefix(r.URL.Path, "/api/") {
        http.NotFound(w, r)
        return
    }
    http.ServeFileFS(w, r, h.static, "index.html")
}
```

本次改造建议先不引入复杂前端路由，降低风险。

## 5. React 前端设计

### 5.1 API 层

把当前 `requestJSON` 抽成 `web/src/api/client.ts`。

职责：

- 默认携带 `credentials: 'same-origin'`。
- 默认设置 `Content-Type: application/json`。
- 处理 `204 No Content`。
- 解析后端 `{ error: string }` 错误。
- 对 `401` 抛出可识别错误，由上层决定是否展示登录框。

建议定义接口：

```ts
export async function requestJSON<T>(
  url: string,
  options?: RequestInit & { authPrompt?: boolean }
): Promise<T>
```

在 `web/src/api/types.ts` 定义：

- `Site`
- `CategoryStat`
- `Stats`
- `AppSettings`
- `User`
- `ApiError`

字段应和 README 中 API 文档保持一致。

### 5.2 状态模型

`App.tsx` 作为页面容器，维护这些核心状态：

- `sites: Site[]`
- `categories: string[]`
- `category: string`
- `query: string`
- `stats: Stats | null`
- `settings: AppSettings | null`
- `user: User | null`
- `theme: ThemeValue`
- `activeDialog: null | 'site' | 'category' | 'emoji' | 'account' | 'settings'`
- `editingSite: Site | null`
- `loading: boolean`
- `error: string`

数据加载建议拆成：

- `bootstrap()`：检查 `/api/session`，失败则匿名浏览，然后加载页面数据。
- `loadAll()`：并发请求分类、站点、统计、设置。
- `loadSitesOnly()`：分类或搜索词变化时只刷新站点列表。
- `reloadAfterMutation()`：新增、编辑、删除、分类修改后刷新全部数据。

### 5.3 组件拆分

建议按现有 UI 区域拆分组件，而不是一开始过度抽象。

`Hero`

- 显示徽章、标题、简介。
- 包含搜索框和统计卡片。
- 搜索框通过 `onQueryChange` 更新父组件状态。

`Tabs`

- 渲染分类 tab。
- 当前分类由父组件传入。
- 点击后调用 `onChange(category)`。

`SiteGrid`

- 渲染站点卡片。
- 登录状态下展示编辑、删除按钮。
- 删除按钮调用父组件传入的 `onDelete(site)`。
- 编辑按钮调用 `onEdit(site)`。

`SiteDialog`

- 负责新增/编辑站点表单。
- 内部维护表单临时值。
- 提交时调用 `POST /api/sites` 或 `PUT /api/sites/{id}`。
- 成功后调用 `onSaved()`。

`CategoryDialog`

- 打开时加载 `/api/category-stats`。
- 支持重命名和删除分类。
- 可以先沿用 `prompt`/`confirm`，后续再替换成更完整的二次确认弹窗。

`AuthScreen`

- 登录表单。
- 登录成功后更新 `user` 并刷新数据。
- 保留“暂不登录，继续浏览”能力。

`HeaderUserMenu`

- 登录入口、设置、修改账号密码、退出登录。
- 未登录点击时打开登录遮罩。

`SettingsDialog`

- 修改页面设置和默认主题。
- 保存后更新 `settings`、`document.title` 和主题。

`ThemeSwitcher`

- 管理当前主题。
- 本地主题覆盖继续使用 `localStorage` key：`navigation.theme.override`。
- 切换个人本地主题只写 localStorage，不调用后端。
- 设置里的默认主题保存到 `/api/settings`。

`AccountDialog`

- 修改账号和密码。
- 成功后更新 `user`。

`EmojiDialog`、`CategoryPicker`

- 替代当前表单中的 emoji 网格和分类下拉。
- 分类输入允许新分类，保持现有行为。

### 5.4 样式迁移

建议第一阶段直接复用现有 CSS：

- 把 `static/css/app.css` 移到 `web/src/styles/app.css`。
- 在 `main.tsx` 中 `import './styles/app.css'`。
- 组件 JSX 尽量保留原 className，例如 `wrap`、`hero`、`panel`、`card`、`dialog-backdrop`。

这样可以把视觉回归风险降到最低。React 化稳定后，再考虑 CSS Modules 或拆分样式文件。

### 5.5 主题策略

主题继续使用 `body[data-theme="dark"]` 这套机制。

React 中建议封装 `useTheme`：

- 初始化读取 `localStorage`。
- 如果没有本地覆盖，则使用后端 `settings.theme`。
- 调用 `document.body.dataset.theme = theme`。
- 更新主题按钮文案。

注意：当前 `saveTheme()` 只保存本地覆盖，不更新后端默认主题。这个行为应保持，避免用户误以为普通切换会影响全局默认主题。

## 6. 构建与开发流程

### 6.1 package.json

`web/package.json` 建议：

```json
{
  "scripts": {
    "dev": "vite --host 127.0.0.1 --port 5173",
    "build": "tsc -b && vite build",
    "preview": "vite preview --host 127.0.0.1 --port 4173"
  },
  "dependencies": {
    "@vitejs/plugin-react": "latest",
    "vite": "latest",
    "typescript": "latest",
    "react": "latest",
    "react-dom": "latest"
  },
  "devDependencies": {}
}
```

实际落地时建议使用明确版本号，不要长期保留 `latest`。

### 6.2 Vite 代理

本地开发时，Vite dev server 运行在 `5173`，Go API 运行在 `8080`。

`web/vite.config.ts` 建议配置：

```ts
export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
```

本地开发命令：

```bash
go run . -port 8080 -data data
cd web && npm run dev
```

访问：

```text
http://127.0.0.1:5173
```

### 6.3 build.sh 调整

当前 `build.sh` 只构建 Go 二进制。React 化后建议先构建前端，再构建 Go。

建议流程：

```bash
cd "$ROOT_DIR/web"
npm ci
npm run build

cd "$ROOT_DIR"
GOCACHE="$GOCACHE" CGO_ENABLED="${CGO_ENABLED:-1}" go build -trimpath -ldflags="-s -w" -o "$OUTPUT" .
```

如果希望本地没有 Node 时仍能只构建 Go，可以提供环境变量跳过前端构建：

```bash
SKIP_WEB_BUILD=1 ./build.sh
```

但默认构建应包含前端，避免 Go embed 找不到 `web/dist`。

### 6.4 Dockerfile 调整

当前 Dockerfile 只有 Go builder。React 化后建议增加 Node 构建阶段：

```dockerfile
FROM node:22-alpine AS web-builder
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./web/dist
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/navigation .
```

最终运行镜像不需要额外复制 `index.html`，因为前端已经 embed 到二进制。

可以删除当前这一行：

```dockerfile
COPY index.html /app/index.html
```

## 7. 迁移步骤

建议分阶段提交，避免一次改动过大。

### 阶段一：搭建 React 工程骨架

1. 新增 `web/` 目录和 Vite React TypeScript 配置。
2. 创建最小 `App.tsx`，先渲染当前静态布局骨架。
3. 迁移 `app.css`，确保视觉基础一致。
4. 调整 Go embed 到 `web/dist`。
5. 调整 `build.sh` 和 Dockerfile。
6. 验证 `npm run build`、`go test ./...`、`./build.sh`。

验收标准：

- `go run .` 可以启动并服务 React 构建产物。
- 首页可以打开，CSS 正常加载。
- Docker 镜像能构建成功。

### 阶段二：迁移只读浏览功能

1. 新增 API client 和类型定义。
2. 实现 `bootstrap()`、`loadAll()`、`loadSitesOnly()`。
3. 迁移设置、主题、统计、分类 tab、站点卡片、搜索。
4. 保持匿名用户可以浏览站点。

验收标准：

- 首页数据来自现有 API。
- 分类切换和搜索正常。
- 主题默认值和本地覆盖正常。
- 移动端布局与旧版一致。

### 阶段三：迁移登录和管理功能

1. 实现登录遮罩。
2. 实现用户菜单、退出登录。
3. 实现新增、编辑、删除站点。
4. 实现分类管理、重命名、删除分类。
5. 实现账号密码修改。
6. 实现页面设置修改。

验收标准：

- 未登录时新增、编辑、删除入口隐藏或触发登录。
- 登录成功后管理功能可用。
- `401` 会正确提示登录。
- 所有写操作成功后页面数据刷新。

### 阶段四：清理旧前端

1. 删除根目录旧 `index.html`。
2. 删除旧 `static/js/app.js` 和 `static/css/app.css`。
3. 更新 README 技术栈和开发命令。
4. 更新 `.gitignore`，忽略：

```text
web/node_modules/
web/dist/
```

是否忽略 `web/dist/` 取决于构建策略：

- 如果 CI/本地构建总会先 `npm run build`，建议忽略 `web/dist/`。
- 如果希望纯 Go 构建无需 Node，必须提交 `web/dist/`，但这会带来构建产物噪音。

本项目推荐忽略 `web/dist/`，让 `build.sh` 和 Docker 负责生成产物。

## 8. 测试与验收

### 8.1 后端测试

继续运行：

```bash
go test ./...
```

React 改造理论上不应改变现有 API 行为。如果需要改 `serveIndex` 或静态路由，建议补充 HTTP handler 测试：

- `/` 返回 `index.html`。
- `/assets/...` 返回静态资源。
- `/api/sites` 仍返回 JSON。
- 未登录访问受保护接口仍返回 `401`。

### 8.2 前端构建检查

运行：

```bash
cd web
npm run build
```

确保 TypeScript 无错误，Vite 构建成功。

### 8.3 手动功能验收清单

只读浏览：

- 打开首页。
- 站点列表正常显示。
- 统计数字正常显示。
- 分类 tab 正常切换。
- 搜索可以过滤站点。
- 点击站点卡片在新标签打开。

认证：

- 未登录时显示“登录”入口。
- 登录失败显示错误。
- 登录成功显示用户名。
- 退出登录后管理入口隐藏。
- 会话过期或 401 时展示登录遮罩。

站点管理：

- 新增站点。
- 编辑站点。
- 删除站点。
- 分类输入支持选择已有分类和输入新分类。
- emoji 和光效选择正常。

分类管理：

- 分类统计正常显示。
- 重命名分类后站点分类同步。
- 删除分类后该分类站点保留但分类清空。

设置与主题：

- 修改站点标题、徽章、主标题、简介。
- 修改全局默认主题。
- 左下角主题切换只影响当前浏览器本地覆盖。
- 刷新页面后本地主题覆盖仍生效。

响应式：

- 桌面宽度下四列卡片正常。
- 平板宽度下两列卡片正常。
- 手机宽度下一列卡片、弹窗和表单不溢出。

## 9. 风险与处理

### 9.1 Go embed 找不到 dist

风险：`web/dist` 不存在时，Go 编译会失败。

处理：

- `build.sh` 默认先执行 `npm ci && npm run build`。
- Docker 使用 Node 阶段生成 `web/dist`。
- README 明确本地构建需要 Node。

### 9.2 Cookie 在 Vite 开发环境下不可用

风险：前端 dev server 是 `5173`，后端是 `8080`，如果直接跨源请求，Cookie 行为可能和生产不一致。

处理：

- 使用 Vite proxy，把 `/api` 代理到 `http://127.0.0.1:8080`。
- 前端请求仍使用相对路径 `/api/...`。
- fetch 保持 `credentials: 'same-origin'`。

### 9.3 样式回归

风险：React 组件拆分后 className 不一致导致样式失效。

处理：

- 第一阶段完整复用原 className。
- 不同时做视觉重设计。
- 改造后用桌面和移动端截图对比。

### 9.4 状态刷新不一致

风险：新增、删除、分类重命名后部分状态未同步。

处理：

- 所有写操作成功后先调用 `loadAll()`。
- 如果当前分类被删除，重置为 `全部`。
- 如果当前分类被重命名，同步更新当前分类值。

### 9.5 过早引入复杂架构

风险：小项目引入路由、状态库、UI 框架导致维护成本上升。

处理：

- 首轮只使用 React、TypeScript、Vite。
- 暂不引入 Redux、React Query、Tailwind 或大型组件库。
- 等接口数量和页面数量明显增加后再评估。

## 10. 建议提交拆分

建议按以下提交拆分：

1. `新增 React 前端工程`
2. `调整前端构建与静态资源嵌入`
3. `迁移导航站只读页面`
4. `迁移登录与站点管理`
5. `迁移分类账号设置功能`
6. `清理旧前端资源并更新文档`

每个提交都应至少保证：

- `go test ./...` 通过。
- 相关阶段的前端构建通过。
- 页面能打开，不出现空白屏。

## 11. 推荐最终命令

开发：

```bash
go run . -port 8080 -data data
cd web && npm run dev
```

测试：

```bash
go test ./...
cd web && npm run build
```

构建：

```bash
./build.sh
```

Docker：

```bash
docker build -t navigation .
docker run -p 8080:8080 -v "$PWD/data:/app/data" navigation
```

## 12. 结论

这次 React 改造应以“保持后端 API 不变、保留现有视觉、先完成等价迁移”为原则。最小可行路径是引入 Vite React TypeScript，将现有 DOM 操作拆为组件和状态更新，把 CSS 原样迁移，并让 Go 继续嵌入 `web/dist`。这样可以在不改变部署模型的前提下，提高前端可维护性，并为后续增加页面、组件测试或更复杂交互留出空间。
