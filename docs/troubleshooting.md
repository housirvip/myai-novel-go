# 排障指南

这篇文档整理了在本地运行、接入真实模型、使用工作流和做迁移时最容易遇到的一些问题。

## 服务启动不了

### 症状

启动 `make run` 或 `./bin/server` 后直接报错退出。

### 优先检查

1. `.env` 是否存在
2. 数据库配置是否正确
3. 端口是否被占用
4. Go 版本是否满足项目要求

### 常见排查方式

```bash
./bin/novel db check
```

或者直接看启动日志中的这些关键项：

- `config.loaded`
- `db.connected`
- `db.migrate.ok`
- `server.starting`

## 健康检查失败

### 症状

```bash
curl http://127.0.0.1:3000/health
```

没有响应或返回异常。

### 可能原因

- 服务根本没有启动成功
- 监听地址或端口不是 `127.0.0.1:3000`
- 容器内服务没映射到本机对应端口

### 排查建议

检查：

- `SERVER_HOST`
- `SERVER_PORT`
- docker compose 端口映射

## `database is locked`

### 常见场景

- SQLite 下并发工作流太高
- 同时有多个写请求在竞争
- 导入、工作流和人工操作同时进行

### 建议处理

优先把并发调低：

```dotenv
WORKFLOW_MAX_CONCURRENCY=1
DB_POOL_MAX=4
```

如果你长期需要更高吞吐，优先切到 MySQL。

## `workflow_queue_full`

### 含义

后台 worker 队列已满，系统主动拒绝新任务。

### 常见原因

- `WORKFLOW_MAX_CONCURRENCY` 太低
- LLM 返回太慢
- 数据库写入成为瓶颈
- 提交任务过快

### 建议处理

1. 先检查任务是否积压
2. 看真实瓶颈在数据库还是 LLM
3. 再决定是否提高并发

不要一上来就盲目把并发拉高。

## 真实模型调用失败

### 常见原因

- 没配置 API Key
- provider 名写错
- base URL 不对
- 模型名与服务不匹配

### 优先检查的配置

OpenAI：

```dotenv
LLM_PROVIDER=openai
OPENAI_API_KEY=...
OPENAI_MODEL=gpt-4o-mini
```

Anthropic：

```dotenv
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=...
ANTHROPIC_MODEL=claude-sonnet-4-20250514
```

Custom：

```dotenv
LLM_PROVIDER=custom
CUSTOM_LLM_BASE_URL=...
CUSTOM_LLM_API_KEY=...
CUSTOM_LLM_MODEL=...
```

## 登录态不生效

### 症状

- 登录成功后后续请求仍像未登录
- `/api/auth/session` 返回 `user: null`

### 检查方向

1. 客户端是否真的带上了 cookie
2. `AUTH_COOKIE_NAME` 是否匹配
3. 如果在 HTTPS 下，`AUTH_COOKIE_SECURE` 是否配置合理
4. 是否误用了旧 cookie

### 一个快速验证方法

先调用：

```text
POST /api/auth/login
```

然后检查响应头里是否有 `Set-Cookie`。

## `/api/user-settings/runtime` 访问失败

### 可能原因

这一组接口要求登录。

如果匿名访问，会失败，这是预期行为。

## Markdown 导入失败

### 常见原因

- 导入的 stage 不支持写入
- frontmatter 中的 `book_id/chapter_no/stage` 与请求路径不一致
- body 为空
- 传的内容类型不对

### 当前可导入阶段

- `plan`
- `draft`
- `final`

## CLI 报 `novel binary not found or not executable`

### 常见场景

运行 `scripts/init_novel_from_settings.sh` 时出现。

### 处理方式

先编译 CLI：

```bash
make cli
```

如果路径不是默认值，可以显式传：

```bash
NOVEL_BIN=./bin/novel ./scripts/init_novel_from_settings.sh /path/to/settings
```

## 初始化脚本报缺少命令

### 常见报错

- `missing command: jq`
- `missing command: python3`

### 处理方式

安装对应依赖后再执行。

## 迁移工具报主键冲突

### 常见原因

目标库不是空库。

### 处理方式

- 换一个空数据库
- 删除旧 SQLite 文件
- 重新准备目标 MySQL 库

迁移工具不会自动清空目标库。

## MySQL 连接失败

### 常见检查项

- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_USER`
- `DB_PASSWORD`

如果使用 compose 的 mysql profile，还要确认：

- MySQL 容器已健康
- 你访问的是正确端口
- 服务是否连的是 compose 内部主机名 `mysql`

## 容器启动了但接口不可用

### 常见原因

- 服务监听地址仍是 `127.0.0.1`
- 容器内没用 `0.0.0.0`
- 对外访问的端口不是实际映射端口

### 建议

容器环境里确保：

```dotenv
SERVER_HOST=0.0.0.0
SERVER_PORT=3000
```

## 不确定问题在哪时先做什么

如果你一时不确定卡在哪里，建议按这个顺序排查：

1. 看服务启动日志
2. 调 `/health`
3. 跑 `./bin/novel db check`
4. 用 mock provider 跑一个最小 `plan`
5. 再切真实模型
6. 再提高并发或切 MySQL

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [API 总览](./api-overview.md)
- [从设定文件初始化项目](./importing-settings.md)
- [从 Node 版本迁移](./migration-from-node.md)