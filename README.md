# navigation

一个轻量级个人导航/书签站。后端使用 Go 1.22 提供 HTTP API，数据存储在 SQLite 中，前端由单个 `index.html` 页面提供。

## 功能

- 管理导航站点：新增、编辑、删除、搜索。
- 按分类浏览站点，并支持删除分类。
- 提供站点数量、分类数量、分类统计等接口。
- 首次启动时可从 `data/sites.json` 导入旧数据到 SQLite。
- 支持本地运行、二进制构建和 Docker 部署。

## 技术栈

- Go 1.22
- SQLite，驱动为 `github.com/mattn/go-sqlite3`
- 原生 HTML/CSS/JavaScript 前端

## 项目结构

```text
.
├── main.go                         # 程序入口，初始化配置、存储、服务和 HTTP 路由
├── index.html                      # 前端页面
├── internal/config                 # 命令行参数和运行配置
├── internal/domain                 # 领域模型
├── internal/service                # 业务逻辑和校验
├── internal/storage                # SQLite 存储实现
├── internal/transport/http         # HTTP 路由、处理器和响应封装
├── data/                           # 运行时数据目录
├── build.sh                        # 本地构建脚本
└── Dockerfile                      # 容器镜像构建文件
```

## 本地运行

```bash
go run . -port 8080 -data data
```

启动后访问：

```text
http://localhost:8080
```

参数说明：

- `-port`：HTTP 服务端口，默认 `8080`。
- `-data`：数据目录，默认 `data`。SQLite 数据库会写入 `${data}/sites.db`。

注意：项目使用 `go-sqlite3`，需要启用 CGO。本地环境必须有可用的 C 编译工具链。

## 构建

```bash
./build.sh
```

默认输出到：

```text
bin/navigation
```

也可以指定输出路径：

```bash
OUTPUT=/tmp/navigation ./build.sh
```

## Docker

构建镜像：

```bash
docker build -t navigation .
```

运行容器：

```bash
docker run -p 8080:8080 -v "$PWD/data:/app/data" navigation
```

容器内数据目录为 `/app/data`，建议挂载到宿主机，避免容器删除后丢数据。

## API

### 站点

- `GET /api/sites`：获取站点列表。
- `GET /api/sites?category=工具&q=go`：按分类和关键词过滤站点。
- `POST /api/sites`：创建站点。
- `PUT /api/sites/{id}`：更新站点。
- `DELETE /api/sites/{id}`：删除站点。

站点 JSON 字段：

```json
{
  "id": "site_xxx",
  "name": "OpenAI",
  "url": "https://openai.com",
  "category": "AI",
  "icon": "🔗",
  "description": "AI 平台",
  "glow": "rgba(96,165,250,.45)",
  "sort": 1,
  "createdAt": "2026-05-26T10:00:00+08:00",
  "updatedAt": "2026-05-26T10:00:00+08:00"
}
```

创建和更新时必须提供：

- `name`
- `url`，必须以 `http://` 或 `https://` 开头
- `category`

### 分类

- `GET /api/categories`：获取分类列表，结果包含 `全部`。
- `DELETE /api/categories/{name}`：删除分类，并将该分类下站点的分类清空。
- `GET /api/category-stats`：获取每个分类的站点数量。

### 统计

- `GET /api/stats`：获取站点数量、分类数量和覆盖率。

## 数据说明

- 默认数据库文件：`data/sites.db`。
- 旧版导入文件：`data/sites.json`。
- 首次启动时，如果存在旧版 JSON 数据，存储层会导入到 SQLite。
- 不要提交生成的数据库文件、WAL/SHM 文件或本地构建产物。

## 测试

```bash
go test ./...
```

## 开发约定

- HTTP 层只负责请求解析和响应输出。
- 业务校验放在 `internal/service`。
- SQL 和持久化细节放在 `internal/storage`。
- 修改 Go 代码后运行 `go fmt ./...` 和 `go test ./...`。
