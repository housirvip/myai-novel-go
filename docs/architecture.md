# 架构说明

这篇文档面向希望理解仓库内部组织方式的开发者，重点说明服务启动路径、核心模块边界，以及请求和工作流如何在系统中流动。

## 总体结构

项目可以分成五层：

1. **入口层**：`cmd/server`、`cmd/novel`、`cmd/migrate-from-node`
2. **配置与基础设施层**：`internal/config`、`internal/logger`、`internal/db`
3. **业务领域层**：`internal/domain/*`
4. **服务接口层**：`internal/server/handler`、`internal/server/middleware`
5. **异步工作流层**：`internal/workflow`

## 主要目录

```text
cmd/
  server/             HTTP 服务入口
  novel/              CLI 入口
  migrate-from-node/  Node 版数据库迁移工具

internal/
  config/             .env 解析与默认值
  logger/             zap 日志初始化
  db/                 GORM 打开与 AutoMigrate
  domain/             业务服务与工作流实现
  llm/                provider 接口与限流
  llmfactory/         LLM / embedding provider 工厂
  server/             Gin 服务组装
  workflow/           worker pool、调度与任务持久化
```

## HTTP 服务启动路径

服务入口在：

- `cmd/server/main.go`

启动顺序大致如下：

1. 加载配置
2. 初始化日志
3. 打开数据库连接
4. 执行数据库迁移
5. 创建 HTTP Server
6. 启动异步任务恢复与调度
7. 监听信号，支持优雅关停

相关文件：

- `cmd/server/main.go:18`
- `internal/server/server.go:46`

## HTTP Server 如何组装

核心组装逻辑在：

- `internal/server/server.go:46`

这里会完成：

- 创建 Gin engine
- 注册中间件
- 初始化各 domain service
- 初始化 LLM factory 和 retrieval service
- 初始化各工作流对象
- 初始化 runner / scheduler / task service
- 注册 API 路由
- 可选挂载 WebUI 静态资源

## 中间件链

当前中间件主要包括：

- 请求上下文与日志注入
- panic recovery
- 统一错误响应
- session 识别
- 书籍访问控制

相关位置：

- `internal/server/server.go:53`
- `internal/server/middleware/`

## 业务层的组织方式

大部分业务能力都在 `internal/domain/` 下，按资源或职责拆分。

例如：

- `book`
- `chapter`
- `outline`
- `world_setting`
- `character`
- `faction`
- `relation`
- `item`
- `story_hook`
- `auth`
- `user_settings`
- `planning`
- `workflows`

这种组织方式的特点是：

- HTTP handler 不直接写复杂业务逻辑
- CLI 和 HTTP 可以复用同一套 service
- 工作流也能直接调用 domain service

## HTTP 请求的流向

一个典型请求大致会这样流动：

```text
Gin Route
  -> Handler
  -> Domain Service
  -> GORM / DB
  -> 返回 JSON
```

例如创建书籍：

- 路由注册：`internal/server/handler/book.go:16`
- handler 调用 service：`internal/server/handler/book.go:44`

## 工作流的流向

一个异步工作流的大致链路如下：

```text
HTTP / CLI
  -> workflows.*Workflow
  -> LLM / Retrieval / DB
  -> workflow.Service 持久化任务
  -> workflow.Runner 执行后台任务
  -> workflow.Scheduler 调度提交
```

### 工作流对象

核心工作流包括：

- `PlanWorkflow`
- `DraftWorkflow`
- `ReviewWorkflow`
- `RepairWorkflow`
- `ApproveWorkflow`
- `StageSummaryWorkflow`

初始化位置：

- `internal/server/server.go:76`

### Runner

`Runner` 是一个有界队列 + worker pool。

特点：

- 有固定 worker 数
- 队列满时直接拒绝提交
- 进程关闭时统一 cancel 上下文

相关文件：

- `internal/workflow/runner.go:24`

### Scheduler

`scheduler` 负责把持久化的任务和后台执行器串起来。

### Task Service

`workflow.Service` 负责：

- 创建任务记录
- 查询任务状态
- 终止任务
- 重启后恢复中断任务

## 为什么 CLI 和 HTTP 能共用一套逻辑

CLI 入口在：

- `cmd/novel/main.go`
- `cmd/novel/cmd/common.go:56`

CLI 会先打开一个 `Container`，里面初始化：

- config
- logger
- DB
- 各资源 service
- retrieval service
- 各 workflow

所以 CLI 与 HTTP 的差别主要是入口形态不同，而不是底层逻辑不同。

## 数据库层

数据库相关逻辑集中在：

- `internal/db/client.go`
- `internal/db/models/`

特点：

- 使用 GORM
- 启动时自动迁移
- 支持 SQLite 和 MySQL
- SQLite 启用 WAL / busy timeout

## LLM 与检索层

LLM 相关能力分成两块：

1. `internal/llm`
   - provider 接口
   - 限流
2. `internal/llmfactory`
   - provider / embedding client 的创建

规划与检索相关能力在：

- `internal/domain/planning/`

这里负责：

- 检索候选加载
- prompt 构造
- intent 解析
- 嵌入检索与重排

## WebUI 托管

如果设置了 `WEBUI_DIST_PATH`，服务会额外挂载：

- `/app/*`

相关文件：

- `internal/server/webui/static.go`

## 优雅关停

服务收到 `SIGINT` 或 `SIGTERM` 时，会：

1. 取消根 context
2. 关闭 HTTP 服务
3. 停止 scheduler
4. 关闭 runner

相关位置：

- `cmd/server/main.go:54`
- `internal/server/server.go:128`

## 适合从哪里开始读代码

如果你第一次阅读这个仓库，建议顺序：

1. `README.md`
2. `cmd/server/main.go`
3. `internal/server/server.go`
4. `internal/server/handler/`
5. `internal/domain/workflows/`
6. `internal/workflow/`
7. `cmd/novel/cmd/common.go`

## 相关文档

- [API 总览](./api-overview.md)
- [工作流说明](./workflows.md)
- [CLI 使用说明](./cli.md)
- [从 Node 版本迁移](./migration-from-node.md)