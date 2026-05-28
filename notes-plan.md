# 笔记能力改造方案

本文档是一套面向 AI 编程代理或开发者的可执行改造方案。目标是在当前导航项目中增加“笔记”能力：笔记正文以 Markdown 文件形式存放在 SQLite 数据库文件旁边的 `notes/` 文件夹中，SQLite 只保存索引、元数据、搜索辅助信息和未来扩展关系。

## 目标

1. 在不破坏现有导航站功能的前提下，新增笔记模块。
2. 笔记正文保存为 Markdown 文件，目录固定为 `<dataDir>/notes/`。
3. SQLite 保存笔记元数据，便于列表、排序、搜索、软删除和未来关联。
4. 后端保持现有分层风格：`domain` 定义模型，`service` 处理业务规则，`storage` 处理 SQLite 和文件读写，`transport/http` 处理 HTTP。
5. 为长期演进成“大而全的个人管理平台”预留标签、关联、全文搜索和附件能力。

## 当前项目观察

当前仓库是一个 Go 1.22 Web 服务：

- 入口：`main.go`
- 配置：`internal/config/config.go`
- 领域模型：`internal/domain`
- 业务服务：`internal/service`
- 持久化：`internal/storage/sqlite_site_store.go`
- HTTP：`internal/transport/http/handler.go`
- 静态前端：`web/dist/*`，通过 `embed.FS` 嵌入
- 运行数据：`data/sites.db`、`data/sites.json`

现有 SQLite store 同时负责站点、用户、设置，并在 `ensureDatabase` 中执行表初始化。笔记模块可以先复用同一个 SQLite 连接，但代码上应避免把所有逻辑继续堆进现有站点方法里。

## 设计原则

1. Markdown 文件是正文源。
2. SQLite 是元数据和索引源。
3. 服务层负责保证 SQLite 和 Markdown 文件之间的一致性。
4. 所有磁盘路径都必须限定在 `<dataDir>/notes/` 内，禁止路径穿越。
5. 初期不做复杂抽象，但所有实体都使用稳定 ID，便于后续统一标签、关联、任务和项目模块。
6. 删除默认软删除，不直接物理删除 Markdown 文件。

## 数据目录约定

基于现有 `-data` 参数推导：

```text
<dataDir>/
  sites.db
  sites.json
  notes/
    2026/
      05/
        note_<id>.md
    attachments/
      note_<id>/
        image.png
    .trash/
      note_<id>.md
```

第一阶段只需要实现：

```text
<dataDir>/notes/YYYY/MM/note_<id>.md
```

附件目录和回收站目录可以先保留规范，不一定在第一阶段实现完整上传能力。

## 数据库设计

在 `internal/storage/sqlite_site_store.go` 的初始化 SQL 中新增 `notes` 表。

建议第一阶段表结构：

```sql
CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL UNIQUE,
    summary TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active',
    pinned INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_notes_status_updated_at
ON notes(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_notes_pinned_updated_at
ON notes(pinned DESC, updated_at DESC);
```

字段说明：

- `id`：稳定 ID，例如复用现有服务层 ID 生成策略或新增 `note_` 前缀 ID。
- `title`：展示标题。
- `file_path`：相对 `dataDir` 的路径，例如 `notes/2026/05/note_abcd1234.md`。
- `summary`：列表摘要，第一阶段可由正文前 120 个字符生成。
- `tags`：JSON 字符串数组，第一阶段先不拆表。
- `status`：`active`、`archived`、`deleted`。
- `pinned`：0 或 1。
- `created_at`、`updated_at`：RFC3339 字符串，保持和现有站点模型一致。
- `deleted_at`：软删除时间，未删除为空字符串。

第二阶段可增加全文搜索：

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    note_id UNINDEXED,
    title,
    content
);
```

FTS 不在第一阶段强制实现，避免初始改造过大。

## 后端文件改造清单

### 1. 新增领域模型

新增文件：`internal/domain/note.go`

建议定义：

```go
package domain

type Note struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	FilePath  string   `json:"filePath"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Status    string   `json:"status"`
	Pinned    bool     `json:"pinned"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	DeletedAt string   `json:"deletedAt,omitempty"`
}

type NoteContent struct {
	Note
	Content string `json:"content"`
}
```

可选常量：

```go
const (
	NoteStatusActive   = "active"
	NoteStatusArchived = "archived"
	NoteStatusDeleted  = "deleted"
)
```

### 2. 新增存储接口和文件读写实现

新增文件：`internal/storage/note_file_store.go`

职责：

- 创建 `<dataDir>/notes/`。
- 根据当前时间和笔记 ID 生成 Markdown 文件路径。
- 写入 Markdown 文件。
- 读取 Markdown 文件。
- 移动到 `.trash/`，或者第一阶段仅更新数据库状态，不移动文件。
- 校验所有目标路径都在 notes 根目录下。

建议方法：

```go
type NoteFileStore struct {
	dataDir  string
	notesDir string
}

func NewNoteFileStore(dataDir string) (*NoteFileStore, error)
func (s *NoteFileStore) NewRelativePath(id string, now time.Time) string
func (s *NoteFileStore) Write(relativePath, content string) error
func (s *NoteFileStore) Read(relativePath string) (string, error)
func (s *NoteFileStore) MoveToTrash(relativePath, id string) (string, error)
```

路径安全要求：

- `relativePath` 必须使用 `filepath.Clean`。
- 不允许绝对路径。
- 不允许 `..` 路径段逃逸。
- 最终绝对路径必须以 `notesDir` 为前缀。

可以实现一个私有方法：

```go
func (s *NoteFileStore) resolve(relativePath string) (string, error)
```

### 3. 扩展 SQLite store 的笔记元数据能力

可以继续在 `internal/storage/sqlite_site_store.go` 中添加方法，也可以拆出 `internal/storage/sqlite_note_store.go`，但接收者仍然是 `*SQLiteSiteStore`。

建议新增文件：`internal/storage/sqlite_note_store.go`

新增方法：

```go
func (s *SQLiteSiteStore) ListNotes(status, query string) ([]domain.Note, error)
func (s *SQLiteSiteStore) GetNote(id string) (domain.Note, error)
func (s *SQLiteSiteStore) CreateNote(note domain.Note) error
func (s *SQLiteSiteStore) UpdateNote(note domain.Note) error
func (s *SQLiteSiteStore) SoftDeleteNote(id, deletedAt string) error
```

实现细节：

- `tags` 在 SQLite 中存 JSON 字符串，读写时用 `encoding/json` 转换。
- `ListNotes` 默认只返回 `status = active` 的笔记。
- `query` 第一阶段只匹配 `title` 和 `summary`，使用 `LIKE` 即可。
- 排序为 `pinned DESC, updated_at DESC`。
- 如果查不到记录，返回 `sql.ErrNoRows`，由 service 映射为 `service.ErrNotFound`。

### 4. 新增业务服务

新增文件：`internal/service/note_service.go`

建议接口：

```go
type NoteMetaStore interface {
	ListNotes(status, query string) ([]domain.Note, error)
	GetNote(id string) (domain.Note, error)
	CreateNote(domain.Note) error
	UpdateNote(domain.Note) error
	SoftDeleteNote(id, deletedAt string) error
}

type NoteContentStore interface {
	NewRelativePath(id string, now time.Time) string
	Write(relativePath, content string) error
	Read(relativePath string) (string, error)
	MoveToTrash(relativePath, id string) (string, error)
}

type NoteService struct {
	mu      sync.Mutex
	meta    NoteMetaStore
	content NoteContentStore
}
```

建议方法：

```go
func NewNoteService(meta NoteMetaStore, content NoteContentStore) *NoteService
func (s *NoteService) ListNotes(status, query string) ([]domain.Note, error)
func (s *NoteService) GetNote(id string) (domain.NoteContent, error)
func (s *NoteService) CreateNote(input domain.NoteContent) (domain.NoteContent, error)
func (s *NoteService) UpdateNote(id string, input domain.NoteContent) (domain.NoteContent, error)
func (s *NoteService) DeleteNote(id string) error
```

校验规则：

- 标题不能为空，最长建议 120 字符。
- 正文最大长度第一阶段可限制为 1MB，避免误传大文件。
- 标签最多 20 个，每个标签最长 32 字符，去重、去空格。
- `status` 只允许 `active` 或 `archived`，删除只能走 `DeleteNote`。
- `pinned` 是布尔值。

标题和摘要处理：

- 如果 `input.Title` 为空，但正文第一行是 `# 标题`，可以使用该标题。
- 如果仍然没有标题，返回 `ValidationError{Message: "笔记标题不能为空"}`。
- `summary` 从去掉 Markdown 标记后的正文中截取，第一阶段也可简单截取原文前 120 字符。

一致性策略：

- 创建时先生成 ID 和文件路径，再写 Markdown 文件，最后写 SQLite 元数据。
- 如果 SQLite 创建失败，尝试删除刚写入的 Markdown 文件；如果不实现删除，也要返回清晰错误。
- 更新时先读取元数据，再写文件，最后更新 SQLite。
- 删除第一阶段可只更新 SQLite 状态为 `deleted`，不移动文件；如果实现移动文件，需要同时更新 `file_path`。

### 5. 接入 main.go

修改 `main.go`：

1. 初始化 SQLite store 后，新增 `NoteFileStore`。
2. 创建 `NoteService`。
3. 把 `NoteService` 传给 HTTP handler。

示意：

```go
noteFiles, err := storage.NewNoteFileStore(cfg.DataDir)
if err != nil {
	log.Fatalf("初始化笔记目录失败: %v", err)
}
noteSvc := service.NewNoteService(store, noteFiles)
handler := httptransport.NewHandler(svc, authSvc, noteSvc, staticFiles)
```

### 6. 修改 HTTP handler

修改文件：`internal/transport/http/handler.go`

Handler 增加字段：

```go
notes *service.NoteService
```

`NewHandler` 签名改为：

```go
func NewHandler(service *service.SiteService, auth *service.AuthService, notes *service.NoteService, static fs.FS) *Handler
```

新增路由：

```go
mux.HandleFunc("/api/notes", h.handleNotes)
mux.HandleFunc("/api/notes/", h.handleNoteByID)
```

认证建议：

- `GET /api/notes` 和 `GET /api/notes/{id}` 第一阶段建议需要登录，因为笔记更偏私人数据。
- `POST /api/notes`、`PUT /api/notes/{id}`、`DELETE /api/notes/{id}` 必须登录。
- 如果希望公开只读，可后续通过设置项控制，不要第一阶段默认公开。

接口定义：

```text
GET /api/notes?q=&status=
POST /api/notes
GET /api/notes/{id}
PUT /api/notes/{id}
DELETE /api/notes/{id}
```

请求和响应示例：

```json
{
  "title": "新的笔记",
  "content": "# 新的笔记\n\n正文",
  "tags": ["idea"],
  "pinned": false,
  "status": "active"
}
```

错误映射：

- `service.ValidationError` -> 400
- `service.ErrNotFound` -> 404
- 存储读取失败 -> 500
- 存储保存失败 -> 500
- 未登录 -> 401

可以复用现有 `writeServiceError`，必要时让它支持 `NoteService` 的错误。

## 前端改造建议

当前运行时使用 `web/dist`。如果仓库中有真实前端源码，应优先改源码再构建；如果没有源码，避免手改压缩后的 dist 文件，先补后端和文档。

第一阶段前端目标：

1. 增加“笔记”入口。
2. 笔记页采用三栏或双栏工作台，不做营销落地页。
3. 支持列表、搜索、新建、编辑、保存、删除。
4. 编辑器先使用 `<textarea>`。
5. 预览可以第二阶段再做；如果做预览，必须处理 Markdown XSS。

推荐交互：

```text
左侧：模块导航（导航 / 笔记 / 设置）
中间：笔记列表、搜索框、新建按钮
右侧：标题输入、标签输入、Markdown textarea、保存/删除按钮
```

第一阶段可以不引入 Markdown 编辑器依赖。后续如果需要更好体验，再考虑 CodeMirror 或 Milkdown。

## 测试计划

### 服务层测试

新增文件：`internal/service/note_service_test.go`

覆盖：

1. 创建笔记时自动生成 ID、时间、路径和摘要。
2. 标题为空时返回校验错误。
3. 标签会去重、去空格。
4. 获取不存在笔记返回 `ErrNotFound`。
5. 更新笔记会保留 ID 和创建时间，更新时间变化。
6. 删除笔记会调用软删除。

### 文件存储测试

新增文件：`internal/storage/note_file_store_test.go`

覆盖：

1. 能在临时目录下创建 `notes/`。
2. 能写入并读取 Markdown。
3. 拒绝绝对路径。
4. 拒绝 `../` 路径穿越。
5. 生成路径格式为 `notes/YYYY/MM/note_<id>.md`。

### SQLite 存储测试

新增文件：`internal/storage/sqlite_note_store_test.go`

使用 `t.TempDir()`，避免污染 `data/sites.db`。

覆盖：

1. 初始化后存在 `notes` 表。
2. 能创建、列表、读取、更新、软删除笔记。
3. `tags` JSON 能正确读写。
4. `ListNotes` 默认不返回 `deleted`。

### HTTP 测试

扩展或新增：`internal/transport/http/handler_test.go`

覆盖：

1. 未登录访问笔记接口返回 401。
2. 登录后可创建笔记。
3. 登录后可读取笔记详情。
4. 创建非法笔记返回 400。
5. 删除不存在笔记返回 404。

## 验收标准

完成第一阶段后，必须满足：

1. `go test ./...` 通过。
2. `go run . -port 8080 -data data` 可以启动。
3. 启动后自动创建 `data/notes/`。
4. 登录后可以通过 API 创建笔记。
5. 创建笔记后，Markdown 文件出现在 `data/notes/YYYY/MM/` 下。
6. SQLite 的 `notes` 表中出现对应元数据。
7. `GET /api/notes` 返回笔记列表，但不返回全文。
8. `GET /api/notes/{id}` 返回笔记元数据和 Markdown 正文。
9. `PUT /api/notes/{id}` 会更新 Markdown 文件和元数据。
10. `DELETE /api/notes/{id}` 不物理删除文件，默认软删除。
11. 任意试图通过路径读写 `notes/` 外文件的行为都会失败。

## 推荐实施顺序

### 阶段一：后端最小闭环

1. 新增 `domain.Note` 和 `domain.NoteContent`。
2. 新增 `NoteFileStore`。
3. 扩展 SQLite 初始化 SQL，增加 `notes` 表。
4. 新增 SQLite 笔记元数据方法。
5. 新增 `NoteService`。
6. 在 `main.go` 初始化并注入。
7. 在 HTTP handler 增加 `/api/notes` 路由。
8. 编写服务层、文件存储、SQLite、HTTP 测试。
9. 运行 `go fmt ./...` 和 `go test ./...`。

### 阶段二：前端最小可用

1. 确认前端源码位置。
2. 增加笔记模块入口。
3. 实现笔记列表和详情加载。
4. 实现新建、编辑、保存、删除。
5. 增加基础 loading、错误和空状态。
6. 构建前端并确认 `web/dist` 更新。

### 阶段三：搜索和体验增强

1. 增加 `q` 搜索。
2. 增加标签筛选。
3. 增加置顶和归档。
4. 增加 Markdown 预览，前端渲染必须做 XSS 防护。
5. 增加自动保存或保存状态提示。

### 阶段四：平台化能力

1. 引入统一标签表 `tags` 和 `taggings`。
2. 引入统一关联表 `links`。
3. 支持笔记关联书签。
4. 支持从笔记提取任务。
5. 增加全局搜索。
6. 增加导入、导出和重建索引能力。

## 长期扩展表设计草案

后续不要急着在第一阶段实现，但新增功能时应尽量兼容以下方向。

统一标签：

```sql
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS taggings (
    tag_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    PRIMARY KEY (tag_id, entity_type, entity_id)
);
```

统一关联：

```sql
CREATE TABLE IF NOT EXISTS links (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relation TEXT NOT NULL,
    created_at TEXT NOT NULL
);
```

示例关系：

```text
note:<id> -> references -> site:<id>
task:<id> -> created_from -> note:<id>
project:<id> -> contains -> note:<id>
```

## 风险和注意事项

1. 不要把 Markdown 正文塞进 SQLite 的 `notes` 表中。正文应该放文件，SQLite 只保留索引和元数据。
2. 不要完全依赖扫描 `notes/` 文件夹作为主数据源。文件系统扫描可以用于后续“重建索引”，但不应作为常规列表接口的数据来源。
3. 不要信任客户端传入的 `filePath`。第一阶段 API 不应允许客户端直接指定文件路径。
4. 不要直接渲染未消毒的 Markdown HTML。
5. 不要物理删除笔记文件，除非以后增加明确的“彻底删除”接口。
6. 不要在 handler 中直接写 SQL 或读写文件，保持现有分层。
7. 如果前端源码缺失，不要直接大量手改 `web/dist/assets/*.js`，应先补源码或确认构建链路。

## AI 执行提示词

可以把下面这段作为后续开发任务提示：

```text
请按 notes-plan.md 的“阶段一：后端最小闭环”实施笔记模块。保持现有 Go 分层风格：domain/service/storage/transport/http。笔记正文必须保存到 <dataDir>/notes/YYYY/MM/note_<id>.md，SQLite 只保存 notes 元数据。所有笔记 API 都需要登录。实现后补充 focused tests，并运行 go fmt ./... 和 go test ./...。不要修改无关功能，不要物理删除用户笔记文件。
```

