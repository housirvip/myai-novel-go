# myai-novel-go

`myai-novel-go` 是 Node.js 版 [myai-novel](../myai-novel) 服务端的 Go 重写,
聚焦高并发场景下的稳定性与可观测性,使用 **Gin + GORM**,在以下四个方面相比原 Node.js 版本有
明显改善:

1. **真正的并发**:Goroutine + worker pool 异步执行 plan/draft/review/repair/approve 长任务,
   HTTP 请求路径 < 50 ms 就拿到 taskId 返回。
2. **数据库连接池**:GORM 连接池 + SQLite WAL + busy_timeout / MySQL `parseTime` 默认开启,
   读写分离友好。
3. **LLM 限流 + 超时**:全局 token bucket 控制对外 LLM RPS,每次调用走 `context.WithTimeout`。
4. **请求级 context 链路**:从 Gin handler → service → GORM → LLM 全程透传,SIGINT 时优雅取消。

## 与原 Node.js 项目的对应关系

| 子系统 | Node 项目 | 此项目 |
|---|---|---|
| HTTP 框架 | Hono | Gin |
| ORM | Kysely | GORM |
| 配置 | dotenv + zod | godotenv + 自写 schema |
| 日志 | pino | zap |
| 数据库 | SQLite + MySQL | SQLite + MySQL(GORM 双 driver) |
| 异步任务 | `void runTaskInBackground` | Worker pool + `chan func` |
| LLM provider | mock/openai/anthropic/custom | 一一对应 |
| 5 阶段工作流 | plan→draft→review→repair→approve | 1:1 复刻,含 pointer 守卫与 dryRun |
| 检索主链 | 规则候选 + 嵌入 + 重排 + sidecar | 规则候选 + 启发式打分 + sidecar(简化) |

未实现:CLI(`novel` 二进制)、WebUI 静态托管、嵌入向量召回、混合检索重排。

## 目录结构

```
cmd/server/main.go               进程入口:加载 config → 初始化 db/logger → 跑迁移 → 异步任务恢复 → 启动 gin
internal/
  config/                        .env 解析与默认值
  logger/                        zap 适配
  db/                            GORM 客户端 + AutoMigrate
    models/                      18 张表的 GORM model
  llm/                           Client interface + ratelimit
    providers/                   mock / openai / anthropic / custom
  llmfactory/                    LLM provider 工厂(避免 import cycle)
  domain/
    shared/                      constants / errors / utils
    book/ chapter/ outline/ ...  9 个资源 service
    planning/                    检索服务 + prompt 构造器 + intent 解析
    workflows/                   plan/draft/review/repair/approve/stage_summary
  workflow/                      worker pool + 异步 task 持久化
  server/                        gin 引擎组装
    middleware/                  request id / recovery / error responder
    handler/                     全部 HTTP 路由
```

## 启动

### 安装

需要 Go 1.22+。

```bash
go mod tidy
go build -o bin/server ./cmd/server
```

### 配置

```bash
cp .env.example .env
```

默认是 SQLite + mock provider,**无需任何 LLM API key 即可跑通完整工作流**(mock provider 内置
关键字分支会返回与原 Node 项目一致的 stub 响应)。

切到真实 LLM:

```dotenv
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
LLM_LOW_MODEL=gpt-4o-mini
LLM_MID_MODEL=gpt-4o
LLM_HIGH_MODEL=gpt-4o
```

切到 MySQL:

```dotenv
DB_CLIENT=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=myai_novel
DB_USER=root
DB_PASSWORD=...
```

### 运行

```bash
./bin/server
# 默认监听 127.0.0.1:3000
# 通过 SERVER_PORT=3300 ./bin/server 切换端口
```

启动日志样例:

```
config.loaded   {"dbClient": "sqlite", "llmProvider": "mock", ...}
db.connected    {"client": "sqlite"}
db.migrate.ok
server.starting {"addr": "127.0.0.1:3000"}
```

`SIGINT/SIGTERM` 触发优雅关停:停止接收新请求 → 等待运行中的 goroutine 任务完成或超时
(`SHUTDOWN_TIMEOUT_SECONDS`,默认 15 秒)。

## API 速览

### 健康
```
GET  /health
```

### 资源 CRUD(每个资源 5 个端点)
```
GET|POST|PATCH|DELETE  /api/books[/:bookId]
GET|POST|PATCH|DELETE  /api/books/:bookId/{chapters,outlines,world-settings,characters,factions,relations,items,hooks}[/:id]
```

### 章节阶段视图
```
GET  /api/books/:bookId/chapters/:chapterNo/stages/:stage         # plan|draft|review|final
PUT  /api/books/:bookId/chapters/:chapterNo/stages/:stage         # 写入新版本(plan|draft|final)
GET  /api/books/:bookId/chapters/:chapterNo/stages/:stage/history
GET  /api/books/:bookId/chapters/:chapterNo/workflow-state
GET  /api/books/:bookId/chapters/:chapterNo/lifecycle             # 等价 workflow-state
```

### 工作流(同步 = 等结果,异步 = 立刻返回 taskId)
```
POST /api/workflows/plan          |  POST /api/workflows/plan/tasks
POST /api/workflows/draft         |  POST /api/workflows/draft/tasks
POST /api/workflows/review        |  POST /api/workflows/review/tasks
POST /api/workflows/repair        |  POST /api/workflows/repair/tasks
POST /api/workflows/approve       |  POST /api/workflows/approve/tasks
POST /api/workflows/author-intent |  POST /api/workflows/author-intent/tasks
POST /api/workflows/stage-summary

GET  /api/workflow-tasks/:taskId
POST /api/workflow-tasks/:taskId/terminate                              # 终止 pending/running 任务
GET  /api/books/:bookId/chapters/:chapterNo/workflow-tasks               # 历史列表
GET  /api/books/:bookId/chapters/:chapterNo/workflow-tasks/latest?type=plan
```

`approve` 支持 `dryRun: true`,只跑两次 LLM 不写 DB,直接返回 final + diff。

### 认证 + 用户设置 + 元信息
```
POST   /api/auth/register                    # email + password (>= 8) + displayName
POST   /api/auth/login                       # email + password,返回 user + Set-Cookie
POST   /api/auth/logout
GET    /api/auth/session                     # 当前 cookie 对应的 user(未登录返回 user: null)

GET    /api/user-settings/runtime            # 当前用户的 LLM provider/model 覆盖(需登录)
PUT    /api/user-settings/runtime
DELETE /api/user-settings/runtime

GET    /api/meta                             # name / version / nodeEnv / llmProvider / webui

POST   /api/books/:bookId/embeddings/refresh # 同步全量重建嵌入索引
```

未登录请求会得到 `actor=anonymous`,大多数读端点仍然可用(供集成 / 单租户场景);
`/api/user-settings/runtime` 强制要求登录。Cookie 名默认 `myai_novel_session`,
HMAC-SHA256 签名(secret = `AUTH_SESSION_SECRET`),server 端存 sha256(token)。

## 端到端冒烟(基于 mock provider)

```bash
# 1. 健康
curl localhost:3000/health

# 2. 创建书 + 设定
curl -X POST localhost:3000/api/books \
  -H 'Content-Type: application/json' \
  -d '{"title":"青岳入门录","targetChapterCount":200}'

curl -X POST localhost:3000/api/books/1/chapters \
  -H 'Content-Type: application/json' \
  -d '{"chapterNo":1,"title":"黑铁令"}'

curl -X POST localhost:3000/api/books/1/characters -H 'Content-Type: application/json' -d '{"name":"林夜","keywords":"林夜"}'
curl -X POST localhost:3000/api/books/1/factions   -H 'Content-Type: application/json' -d '{"name":"青岳宗","keywords":"青岳宗"}'
curl -X POST localhost:3000/api/books/1/items      -H 'Content-Type: application/json' -d '{"name":"黑铁令","ownerType":"none","keywords":"黑铁令"}'
curl -X POST localhost:3000/api/books/1/hooks      -H 'Content-Type: application/json' -d '{"title":"黑铁令异常","hookType":"mystery","keywords":"黑铁令"}'

# 3. 跑工作流(同步)
curl -X POST localhost:3000/api/workflows/plan    -H 'Content-Type: application/json' -d '{"bookId":1,"chapterNo":1,"provider":"mock","authorIntent":"林夜入宗"}'
curl -X POST localhost:3000/api/workflows/draft   -H 'Content-Type: application/json' -d '{"bookId":1,"chapterNo":1,"provider":"mock"}'
curl -X POST localhost:3000/api/workflows/review  -H 'Content-Type: application/json' -d '{"bookId":1,"chapterNo":1,"provider":"mock"}'
curl -X POST localhost:3000/api/workflows/repair  -H 'Content-Type: application/json' -d '{"bookId":1,"chapterNo":1,"provider":"mock"}'
curl -X POST localhost:3000/api/workflows/approve -H 'Content-Type: application/json' -d '{"bookId":1,"chapterNo":1,"provider":"mock"}'

# 4. 异步任务
TASK=$(curl -s -X POST localhost:3000/api/workflows/plan/tasks \
  -H 'Content-Type: application/json' \
  -d '{"bookId":1,"chapterNo":2,"provider":"mock"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
curl localhost:3000/api/workflow-tasks/$TASK   # 轮询直到 status=succeeded

# 5. 阶段读
curl localhost:3000/api/books/1/chapters/1/stages/final
curl localhost:3000/api/books/1/chapters/1/lifecycle
```

## 关键并发设计

### 异步 worker pool
- `internal/workflow/runner.go` 用带缓冲 chan + N 个 worker(`WORKFLOW_MAX_CONCURRENCY`,默认 4)。
- `WORKFLOW_MAX_CONCURRENCY` 现在是严格并发上限;队列打满时返回 `workflow_queue_full`(HTTP 503),不会再偷偷起额外 goroutine。
- 进程关停时 cancel root context,worker 见到 ctx.Done 立即退出。

### `WORKFLOW_MAX_CONCURRENCY` 推荐值
- SQLite: 推荐 `1~2`;只有在确认写竞争、`database is locked` 和任务排队都可接受时再升到 `3~4`。
- MySQL: 推荐从 `4` 起步,常见可用区间是 `4~8`。
- `DB_POOL_MAX` 至少应高于 `WORKFLOW_MAX_CONCURRENCY`;保守建议预留 `+2~4` 连接给登录、资源 CRUD 和轮询接口。
- `LLM_RATE_LIMIT_RPS` 也要高于并发数;若单个 workflow 常包含多次 LLM 调用,建议把 `WORKFLOW_MAX_CONCURRENCY` 控制在 `LLM_RATE_LIMIT_RPS / 2` 以内作为起步值。
- 如果你看到大量 `workflow_queue_full`,优先先看数据库和 LLM 是否已成瓶颈,不要只靠继续调大并发。

| 场景 | DB_CLIENT | WORKFLOW_MAX_CONCURRENCY | DB_POOL_MAX | LLM_RATE_LIMIT_RPS | 说明 |
|---|---|---:|---:|---:|---|
| 本地开发 / SQLite + mock | `sqlite` | `1` | `4` | `10` | 最稳妥,适合单人开发和调试 workflow 链路 |
| 本地开发 / SQLite + 真 LLM | `sqlite` | `2` | `6` | `8~12` | 先保守限制写竞争,把并发留给 LLM 往返耗时 |
| 小规模部署 / MySQL | `mysql` | `4` | `8` | `12~20` | 默认推荐组合,适合少量用户并发触发章节工作流 |
| 中等吞吐 / MySQL | `mysql` | `6` | `12` | `20~30` | 只有在 DB 和 LLM 都稳定时再升到这一档 |
| 偏高吞吐 / MySQL | `mysql` | `8` | `16` | `30+` | 需要重点监控排队、慢查询和 provider 限流,不要直接跳到更高 |

可直接参考的 `.env` 组合:

```dotenv
# SQLite 开发环境
DB_CLIENT=sqlite
DB_POOL_MAX=4
WORKFLOW_MAX_CONCURRENCY=1
LLM_RATE_LIMIT_RPS=10

# MySQL 小规模部署
DB_CLIENT=mysql
DB_POOL_MAX=8
WORKFLOW_MAX_CONCURRENCY=4
LLM_RATE_LIMIT_RPS=16
```

### Pointer 守卫
每个工作流在事务内会再读一次 chapter,与 LLM 调用前的快照对比 `currentPlanId/DraftId/ReviewId/FinalId`,
如有变化抛 409 `pointer_changed`,防止并发执行覆盖彼此的成果。

### 启动恢复
`workflow.Service.RecoverInterrupted` 启动时把所有 `pending/running` 任务标 `failed`
(`error_code=process_restart`),防止 UI 永远显示 running。

### 限流
`internal/llm/ratelimit.go` 用 `golang.org/x/time/rate` 全局限制对外 LLM RPS;每次调用还包一层
`context.WithTimeout(LLM_REQUEST_TIMEOUT_SECONDS)`,模型卡住时不会拖死 worker。

## 数据模型

18 张表(完全对齐原 SQLite/MySQL schema):

```
books, outlines, world_settings, characters, factions, relations, items, story_hooks
chapters, chapter_plans, chapter_drafts, chapter_reviews, chapter_finals
retrieval_documents, retrieval_facts, story_events, chapter_segments
workflow_tasks
```

`chapter_plans.retrieved_context`(LongText)固化 plan 阶段产出的 RetrievedContext JSON,
draft / review / repair / approve 阶段直接复用,不再重做检索。

## 嵌入 / 混合检索 / 重排

`internal/domain/planning/` 现在按策略模式装配:

```
RetrievalService.Retrieve
  └── CandidateProvider.LoadCandidates
        ├── RuleCandidateProvider          (规则召回:关键词命中 + manual_id + open-hook)
        └── EmbeddingCandidateProvider     (规则候选基础上叠加嵌入命中,reason=embedding_match/support)
              └── EmbeddingSearcher
                    ├── BasicEmbeddingSearcher  (纯 cosine)
                    └── HybridEmbeddingSearcher (cosine·w + lexical·w + entity_type bonus)
  └── Reranker.Rerank
        ├── HeuristicReranker  (默认:base + manual=40 + keyword=15 + embedding=10/6 + continuity + hook 距离)
        └── PassthroughReranker
```

启用嵌入只需在 `.env` 中:

```dotenv
PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=hash       # 离线测试用 32 维 hash provider
PLANNING_RETRIEVAL_EMBEDDING_SEARCH_MODE=hybrid  # basic 或 hybrid
# 或接 OpenAI 兼容协议的远端嵌入服务
PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=custom
CUSTOM_EMBEDDING_BASE_URL=https://api.example.com/v1
CUSTOM_EMBEDDING_API_KEY=sk-...
CUSTOM_EMBEDDING_MODEL=text-embedding-3-small
```

启用后调 `POST /api/books/:bookId/embeddings/refresh` 同步全量重建索引。
索引文档写入 `retrieval_documents`(layer=embedding)的 `payload_json` 字段(包含向量)。

## Markdown 导入 / 导出

每个章节阶段(plan/draft/review/final)都可以导出为 markdown:

```bash
curl -s "localhost:3000/api/books/1/chapters/1/stages/final/export" -o chapter-1-final.md
# 修改文件后回写
curl -X POST localhost:3000/api/books/1/chapters/1/stages/draft/import \
  -H 'Content-Type: text/markdown' --data-binary @chapter-1-final.md
```

格式规约(与原 Node 项目字节级一致):frontmatter (`key: value`) + `# Title` + `## Summary` + `## Content`。
import 校验 frontmatter 的 `book_id/chapter_no/stage` 必须与请求路径一致。

## CLI 二进制 (`novel`)

```bash
go build -o bin/novel ./cmd/novel
./bin/novel --help
./bin/novel db init
./bin/novel book create --title 测试书 --target-chapter-count 100
./bin/novel chapter create --book 1 --chapter 1 --title 第一章
./bin/novel character create --book 1 --name 林夜 --keywords 林夜
./bin/novel plan    --book 1 --chapter 1 --provider mock --author-intent 入门
./bin/novel draft   --book 1 --chapter 1 --provider mock
./bin/novel review  --book 1 --chapter 1 --provider mock
./bin/novel repair  --book 1 --chapter 1 --provider mock
./bin/novel approve --book 1 --chapter 1 --provider mock
./bin/novel chapter export --book 1 --chapter 1 --stage final --output /tmp/ch1.md
./bin/novel chapter import --book 1 --chapter 1 --stage draft --input /tmp/ch1.md --force
```

CLI 与 HTTP server 共享同一份 `internal/domain/...` service 层 + 同一份 SQLite/MySQL,
差别仅在入口形态。

## WebUI 静态托管

`.env` 设 `WEBUI_DIST_PATH=../myai-novel/webui/dist` 后,server 会:
- `/app/*` 直接服务静态资源(index.html / assets / ...)
- `/app/<任何 SPA 路径>` 找不到文件时回退到 `index.html`
- `/api/*` 与其它路由不受影响
- `WEBUI_DIST_PATH` 留空则不挂载,行为退化为现有的 `/api/*` only。

## 开发命令

统一通过 `make` 管理:

```bash
make build         # 编译 server + novel 双二进制
make test          # go test ./...
make test-race     # go test -race -count=1 ./...
make vet           # go vet ./...
make run           # 起服务(走 .env 默认配置)
make docker        # docker build -t myai-novel-go:dev
make docker-up     # docker compose up -d --build
make docker-down   # docker compose down -v
make clean         # 清理 bin/ 和 SQLite 文件
```

也可以直接走 `go` 命令(`go build ./...`、`go test -race ./...`、`go vet ./...`)。

## 容器化部署

```bash
make docker-up                       # 默认 SQLite + mock provider 一键起,127.0.0.1:3000
docker compose --profile mysql up -d # 启 MySQL + server-mysql 在 3001
docker compose --profile mysql down -v
```

`Dockerfile` 是 multi-stage(golang:1.22-alpine → alpine:3.20),约 60 MB。
`HEALTHCHECK` 命中 `/health`;k8s 环境再用 `/healthz/ready` 做 readiness probe(检查 DB 连通)。

## CI

`.github/workflows/ci.yml`:Linux/macOS 双环境跑 `go vet`、`go test -race`、`go build`,
push 到主分支额外跑 `docker build` + 容器内冒烟。

## 从原 Node 项目迁移数据

两边表结构完全对齐(Go 端就是按 Node schema 建的),用内置工具一键复制:

```bash
go build -o bin/migrate-from-node ./cmd/migrate-from-node
DB_SQLITE_PATH=./data/novel.db ./bin/migrate-from-node \
  --source ../myai-novel/data/novel.db \
  --dry-run     # 先 dry-run 看行数

DB_SQLITE_PATH=./data/novel.db ./bin/migrate-from-node \
  --source ../myai-novel/data/novel.db
```

工具按 `models.All()` 21 张表逐一搬运,主键 ID 保留。目标库需先为空(否则主键冲突)。
也支持 MySQL → MySQL / SQLite → MySQL(`--source-driver mysql --source-dsn ...` 与 `--target-driver mysql`)。

## 多租户与权限

- **匿名访问**:server 默认允许匿名调用 `/api/*` 大多数路由(向后兼容单租户场景);
  `/api/user-settings/runtime` 强制要求登录。
- **书归属**:`books.owner_user_id` 决定能否被访问:
  - `IS NULL` → 任意 actor 可访问(系统级 / 引导期)
  - 已分配 → 仅该 user 本人或 system actor 可访问,其它 actor 直接 403
- **List 过滤**:`GET /api/books` 对登录用户返回 `owner_user_id IN (NULL, self)`;
  对匿名仅返回 `owner_user_id IS NULL`。
- **Create 自动归属**:登录用户创建的 book 自动 `owner_user_id = self.id`;
  匿名创建的 book 是无主的(供测试 / 集成场景)。
- 实现:`internal/server/middleware/book_access.go::RequireBookAccess`,挂在全局中间件链上,
  通过 `c.Param("bookId")` 自动只对匹配到 `:bookId` 的路由生效。

## 已知简化

- 检索 observability 输出简化为单条 zap info,未输出原项目的 funnel/分桶细节。
- `LLM-based reranker` 暂不实现(原项目 `retrieval-reranker-factory.ts` 中保留扩展点,我们也跳过)。
- `OpenAI` 原生 embedding provider 未单独实现:用 `PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=custom` 配 OpenAI base URL 即可达到等效。
- prompts.go 的中文 prompt 与原项目语义一致(关键字保留),但具体措辞不是逐字一致,
  实际生产时需要按业务侧口径再细调。
- WebUI 构建产物不打入二进制,运行时按 env 路径读盘。
- MySQL 集成测试未在 CI 里跑(代码支持但 unit test 都用 SQLite)。

## License

私有项目。
