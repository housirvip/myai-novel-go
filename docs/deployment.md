# 部署说明

这篇文档介绍如何用 `Docker Compose` 或更接近生产的方式运行 `myai-novel-go`。

## 部署前建议

如果你只是本地体验，请优先看 [快速开始](./getting-started.md)。

如果你准备持续运行服务，建议优先考虑：

- 明确数据库选择
- 明确 LLM provider
- 修改认证 secret
- 合理设置工作流并发

## Docker Compose 快速启动

项目内置了 `docker-compose.yml`，默认场景是：

- 服务端使用 SQLite
- LLM provider 使用 mock
- 监听 `3000`

启动：

```bash
make docker-up
```

或者：

```bash
docker compose up -d --build
```

健康检查：

```bash
curl http://127.0.0.1:3000/health
```

停止并清理：

```bash
make docker-down
```

## 默认 compose 行为

默认 `server` 服务会：

- 暴露 `3000:3000`
- 使用 `/app/data/novel.db` 作为 SQLite 数据库
- 挂载数据卷保存数据库和日志
- 使用 `/health` 做容器健康检查

卷包括：

- `novel_data`
- `novel_logs`

## MySQL profile

如果你想用 MySQL 运行，可以启用 `mysql` profile。

启动：

```bash
docker compose --profile mysql up -d
```

这个 profile 会启动：

- `mysql`
- `server-mysql`

其中：

- MySQL 对外端口：`3306`
- `server-mysql` 对外端口：`3001`

健康检查地址：

```bash
curl http://127.0.0.1:3001/health
```

停止：

```bash
docker compose --profile mysql down -v
```

## 真实 LLM 的环境变量透传

compose 文件中会透传这些变量：

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `AUTH_SESSION_SECRET`

因此你可以在本地 `.env` 中配置真实模型 Key，再启动容器。

例如：

```dotenv
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-key
AUTH_SESSION_SECRET=replace-me
```

然后执行：

```bash
docker compose up -d --build
```

## 容器环境下的重要配置

### 服务监听

容器内通常应监听：

```dotenv
SERVER_HOST=0.0.0.0
SERVER_PORT=3000
```

### 认证 secret

不要在对外环境中继续使用默认值：

```dotenv
AUTH_SESSION_SECRET=replace-me-with-a-real-secret
```

### 日志格式

建议容器内使用：

```dotenv
LOG_FORMAT=json
```

## 数据库选择建议

### SQLite

适合：

- 本地开发
- 单人使用
- 轻量测试环境

优点：

- 启动简单
- 零外部依赖

注意：

- 并发写入能力有限
- 工作流并发建议保守

### MySQL

适合：

- 持续运行
- 多人访问
- 更高吞吐需求

优点：

- 更适合多并发工作流
- 更适合长期部署

## 工作流并发建议

### SQLite 场景

建议从下面的配置起步：

```dotenv
DB_CLIENT=sqlite
DB_POOL_MAX=4
WORKFLOW_MAX_CONCURRENCY=1
LLM_RATE_LIMIT_RPS=10
```

### MySQL 场景

建议从下面的配置起步：

```dotenv
DB_CLIENT=mysql
DB_POOL_MAX=8
WORKFLOW_MAX_CONCURRENCY=4
LLM_RATE_LIMIT_RPS=16
```

如果你看到：

- `workflow_queue_full`
- 数据库锁竞争
- 任务排队严重

请先检查数据库和 LLM 吞吐，不要直接无脑调高并发。

## 生产化时至少要检查的事情

在对外部署前，至少确认：

1. `AUTH_SESSION_SECRET` 已替换
2. 选择了合适的数据库
3. `WORKFLOW_MAX_CONCURRENCY` 已按数据库能力调优
4. LLM API Key 通过安全方式注入
5. 日志目录和数据目录有持久化策略
6. 健康检查路径可被上层系统访问

## 镜像构建

也可以单独构建镜像：

```bash
make docker
```

项目使用 multi-stage Dockerfile，运行时镜像较小，适合本地和测试部署。

## 常见部署路径

### 路径一：本地或测试环境

- SQLite
- mock provider
- `docker compose up -d --build`

### 路径二：小规模内网服务

- MySQL
- 真实 LLM provider
- `docker compose --profile mysql up -d`

### 路径三：已有外部 MySQL

- 不一定使用 compose 自带 MySQL
- 通过 `.env` 指向外部数据库
- 仍可复用服务镜像和环境变量体系

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [工作流说明](./workflows.md)
- [CLI 使用说明](./cli.md)