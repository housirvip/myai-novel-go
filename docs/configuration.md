# 配置说明

这篇文档介绍 `myai-novel-go` 的主要配置项和常见配置组合。第一版不追求逐行解释所有环境变量，而是按使用场景组织，让你知道应该改哪些值。

## 配置文件

项目使用环境变量配置，默认从根目录下的 `.env` 读取：

```bash
cp .env.example .env
```

默认 `.env.example` 已经适合本地开发：

- `DB_CLIENT=sqlite`
- `DB_SQLITE_PATH=./data/novel.db`
- `LLM_PROVIDER=mock`
- `SERVER_HOST=127.0.0.1`
- `SERVER_PORT=3000`

## 场景一：本地开发，使用 SQLite + mock

这是最推荐的起步方式。

```dotenv
DB_CLIENT=sqlite
DB_SQLITE_PATH=./data/novel.db
LLM_PROVIDER=mock
SERVER_HOST=127.0.0.1
SERVER_PORT=3000
```

特点：

- 无需 MySQL
- 无需外部 LLM API Key
- 能完整跑通工作流主链路
- 最适合本地调试和阅读代码

## 场景二：本地开发，使用真实 LLM

如果你想测试真实模型输出，可以切换 provider。

### OpenAI

```dotenv
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini
```

也可以显式覆盖低、中、高三档模型：

```dotenv
LLM_LOW_MODEL=gpt-4o-mini
LLM_MID_MODEL=gpt-4o
LLM_HIGH_MODEL=gpt-4o
```

### Anthropic

```dotenv
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-key
ANTHROPIC_MODEL=claude-sonnet-4-20250514
```

### 自定义兼容服务

```dotenv
LLM_PROVIDER=custom
CUSTOM_LLM_BASE_URL=https://api.example.com/v1
CUSTOM_LLM_API_KEY=your-key
CUSTOM_LLM_MODEL=custom-default
```

## 场景三：切换到 MySQL

如果你准备长期运行、多人使用，或者希望减少 SQLite 写竞争，可以切到 MySQL。

```dotenv
DB_CLIENT=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=myai_novel
DB_USER=root
DB_PASSWORD=your-password
```

项目启动时会自动执行迁移。

## 日志配置

常用日志项：

```dotenv
LOG_LEVEL=info
LOG_FORMAT=pretty
LOG_DIR=./logs
LOG_LLM_CONTENT_ENABLED=false
LOG_LLM_CONTENT_MAX_CHARS=4000
```

建议：

- 本地开发：`LOG_FORMAT=pretty`
- 容器环境：`LOG_FORMAT=json`
- 生产默认不要开启完整 LLM 内容日志

## 服务监听

```dotenv
SERVER_HOST=127.0.0.1
SERVER_PORT=3000
SERVER_BODY_LIMIT=1048576
```

如果你在容器中运行，通常会改成：

```dotenv
SERVER_HOST=0.0.0.0
```

## 工作流并发与限流

工作流相关配置是运行稳定性的关键。

```dotenv
WORKFLOW_MAX_CONCURRENCY=4
LLM_RATE_LIMIT_RPS=20
LLM_REQUEST_TIMEOUT_SECONDS=120
SHUTDOWN_TIMEOUT_SECONDS=15
```

### 推荐起步值

#### SQLite 开发环境

```dotenv
DB_CLIENT=sqlite
DB_POOL_MAX=4
WORKFLOW_MAX_CONCURRENCY=1
LLM_RATE_LIMIT_RPS=10
```

#### MySQL 小规模部署

```dotenv
DB_CLIENT=mysql
DB_POOL_MAX=8
WORKFLOW_MAX_CONCURRENCY=4
LLM_RATE_LIMIT_RPS=16
```

### 调整建议

- SQLite 下优先用 `1~2` 的并发。
- MySQL 下可以从 `4` 起步。
- `DB_POOL_MAX` 应高于 `WORKFLOW_MAX_CONCURRENCY`。
- 如果频繁出现 `workflow_queue_full`，先检查数据库和 LLM 是否已成为瓶颈，不要只是一味调大并发。

## 检索与嵌入配置

默认情况下，嵌入检索是关闭的。

```dotenv
PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=none
PLANNING_RETRIEVAL_EMBEDDING_SEARCH_MODE=basic
```

### 使用本地 hash embedding 做离线测试

```dotenv
PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=hash
PLANNING_RETRIEVAL_EMBEDDING_SEARCH_MODE=hybrid
```

### 使用自定义 embedding 服务

```dotenv
PLANNING_RETRIEVAL_EMBEDDING_PROVIDER=custom
CUSTOM_EMBEDDING_BASE_URL=https://api.example.com/v1
CUSTOM_EMBEDDING_API_KEY=your-key
CUSTOM_EMBEDDING_MODEL=custom-embedding-v1
CUSTOM_EMBEDDING_BATCH_SIZE=10
```

启用后，可以通过接口重建嵌入索引：

```text
POST /api/books/:bookId/embeddings/refresh
```

## 认证与会话

认证相关配置如下：

```dotenv
AUTH_SESSION_SECRET=dev-session-secret-change-me
AUTH_SESSION_TTL_HOURS=168
AUTH_COOKIE_NAME=myai_novel_session
AUTH_COOKIE_SECURE=false
```

### 生产环境建议

- 一定要显式修改 `AUTH_SESSION_SECRET`
- 如果走 HTTPS，设置 `AUTH_COOKIE_SECURE=true`
- 不要使用默认开发 secret 对外部署

## WebUI 静态托管

如果你有前端构建产物，可以让服务端顺带托管它：

```dotenv
WEBUI_DIST_PATH=../myai-novel/webui/dist
```

留空则不挂载 WebUI，仅提供 API。

## 一组常用配置模板

### 模板 A：本地最小体验

```dotenv
DB_CLIENT=sqlite
DB_SQLITE_PATH=./data/novel.db
LLM_PROVIDER=mock
WORKFLOW_MAX_CONCURRENCY=1
LLM_RATE_LIMIT_RPS=10
SERVER_HOST=127.0.0.1
SERVER_PORT=3000
```

### 模板 B：本地真实模型调试

```dotenv
DB_CLIENT=sqlite
DB_SQLITE_PATH=./data/novel.db
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini
WORKFLOW_MAX_CONCURRENCY=1
LLM_RATE_LIMIT_RPS=4
```

### 模板 C：小规模 MySQL 部署

```dotenv
DB_CLIENT=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=myai_novel
DB_USER=novel
DB_PASSWORD=novelpw
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-key
WORKFLOW_MAX_CONCURRENCY=4
DB_POOL_MAX=8
LLM_RATE_LIMIT_RPS=16
AUTH_SESSION_SECRET=replace-me
```

## 相关文档

- [快速开始](./getting-started.md)
- [工作流说明](./workflows.md)
- [CLI 使用说明](./cli.md)
- [部署说明](./deployment.md)