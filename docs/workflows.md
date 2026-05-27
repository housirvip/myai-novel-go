# 工作流说明

`myai-novel-go` 的核心能力是把小说章节生成拆成多个明确阶段，并支持同步或异步执行。这篇文档帮助你理解系统是怎么跑这些阶段的，以及应该怎么调用它们。

## 工作流阶段

当前主流程包含五个核心阶段：

1. `plan`
2. `draft`
3. `review`
4. `repair`
5. `approve`

此外还有：

- `author-intent`
- `stage-summary`

其中最常见的使用顺序是：

```text
plan -> draft -> review -> repair -> approve
```

## 两种调用模式

### 同步模式

同步模式会等待阶段执行完成后，直接返回结果。

示例：

```text
POST /api/workflows/plan
POST /api/workflows/draft
POST /api/workflows/review
POST /api/workflows/repair
POST /api/workflows/approve
```

适合：

- 本地调试
- 小规模脚本调用
- 立即查看阶段输出

### 异步模式

异步模式会立即返回任务 ID，由后台 worker 执行，调用方再轮询任务状态。

示例：

```text
POST /api/workflows/plan/tasks
POST /api/workflows/draft/tasks
POST /api/workflows/review/tasks
POST /api/workflows/repair/tasks
POST /api/workflows/approve/tasks
```

适合：

- 前端界面触发长任务
- 避免 HTTP 请求长时间挂起
- 多章节排队执行

## 异步任务的典型调用方式

### 1. 提交任务

```bash
TASK=$(curl -s -X POST http://127.0.0.1:3000/api/workflows/plan/tasks \
  -H 'Content-Type: application/json' \
  -d '{"bookId":1,"chapterNo":2,"provider":"mock"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
```

### 2. 查询任务状态

```bash
curl http://127.0.0.1:3000/api/workflow-tasks/$TASK
```

### 3. 终止任务

```text
POST /api/workflow-tasks/:taskId/terminate
```

## `approve` 的 dry run

`approve` 支持 `dryRun: true`。

这个模式下会：

- 跑 LLM 逻辑
- 返回 final 和 diff 预览
- 不写数据库

适合：

- 先预览最终效果
- 验证 prompt 或模型输出
- 避免直接覆盖最终阶段数据

## 阶段结果查看

工作流执行后，可以通过章节阶段接口查看产物：

```text
GET /api/books/:bookId/chapters/:chapterNo/stages/:stage
GET /api/books/:bookId/chapters/:chapterNo/stages/:stage/history
GET /api/books/:bookId/chapters/:chapterNo/workflow-state
GET /api/books/:bookId/chapters/:chapterNo/lifecycle
```

常见 `stage` 包括：

- `plan`
- `draft`
- `review`
- `final`

## 并发与队列行为

后台执行依赖 worker pool，并且有严格并发上限。

关键配置：

```dotenv
WORKFLOW_MAX_CONCURRENCY=4
```

当队列打满时，系统会返回：

- `workflow_queue_full`
- HTTP 503

这意味着系统不会无限起 goroutine 硬顶，而是明确拒绝超载请求。

## Pointer 守卫

为了防止并发工作流互相覆盖章节结果，工作流会在事务内重新检查章节当前 pointer。

如果在 LLM 运行期间，章节相关阶段已经被其他流程更新，系统会返回：

- `pointer_changed`
- HTTP 409

这是一种保护机制，目的是避免旧结果覆盖新结果。

## 进程重启后的任务恢复

系统启动时会恢复中断状态。

如果进程在任务运行中被重启，原先处于 `pending` 或 `running` 的任务会被标记为失败，错误码为：

- `process_restart`

这样前端或调用方不会永远看到任务卡在 `running`。

## 并发建议

### SQLite

推荐：

- `WORKFLOW_MAX_CONCURRENCY=1~2`

适合：

- 本地开发
- 单人使用
- 低吞吐场景

### MySQL

推荐从：

- `WORKFLOW_MAX_CONCURRENCY=4`

开始，根据数据库和模型吞吐再逐步上调。

### 配套建议

- `DB_POOL_MAX` 至少要高于并发数。
- `LLM_RATE_LIMIT_RPS` 不应低于实际工作流需求。
- 如果单个工作流会触发多次 LLM 调用，可以把并发先控制在 `LLM_RATE_LIMIT_RPS / 2` 左右作为起步值。

## CLI 工作流命令

除了 HTTP 接口，CLI 也支持同步工作流：

```bash
./bin/novel plan --book 1 --chapter 1 --provider mock --author-intent 入门
./bin/novel draft --book 1 --chapter 1 --provider mock
./bin/novel review --book 1 --chapter 1 --provider mock
./bin/novel repair --book 1 --chapter 1 --provider mock
./bin/novel approve --book 1 --chapter 1 --provider mock
./bin/novel approve --book 1 --chapter 1 --provider mock --dry-run
```

CLI 适合：

- 本地脚本化验证
- 快速构造数据并跑通阶段链路
- 不依赖 HTTP 客户端时直接操作服务层

## 一个推荐的最小使用顺序

如果你第一次体验项目，建议按这个顺序：

1. 启动服务
2. 创建一本书
3. 创建一个章节
4. 创建少量人物或世界设定
5. 先跑 `plan`
6. 再跑 `draft`
7. 如果需要完整闭环，再继续 `review -> repair -> approve`

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [CLI 使用说明](./cli.md)
- [部署说明](./deployment.md)