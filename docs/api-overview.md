# API 总览

这篇文档给你一个面向集成者的 HTTP API 地图。它不是完整 OpenAPI，但足够帮助你快速理解服务提供了哪些能力、资源如何组织、以及工作流相关接口怎么调用。

## 基本信息

- 默认地址：`http://127.0.0.1:3000`
- 内容类型：大部分写接口使用 `application/json`
- 返回格式：统一 JSON

## 健康与元信息

### 健康检查

```text
GET /health
```

适合用于：

- 本地启动校验
- 容器健康检查
- 反向代理探活

### 服务元信息

```text
GET /api/meta
```

返回内容包括：

- 项目名称
- 版本
- 运行环境
- 当前 LLM provider
- 是否挂载 WebUI

## 认证与会话

### 注册

```text
POST /api/auth/register
```

请求字段：

- `email`
- `password`
- `displayName`

约束：

- `password` 至少 8 位

### 登录

```text
POST /api/auth/login
```

登录成功后会返回用户信息，并通过 `Set-Cookie` 写入会话 cookie。

### 登出

```text
POST /api/auth/logout
```

### 查询当前会话

```text
GET /api/auth/session
```

未登录时，返回的 `user` 为 `null`。

## 书籍资源

### 列表 / 创建

```text
GET  /api/books
POST /api/books
```

### 单本书查询 / 更新 / 删除

```text
GET    /api/books/:bookId
PATCH  /api/books/:bookId
DELETE /api/books/:bookId
```

### 权限行为

项目支持匿名访问和登录态访问共存：

- 匿名请求可以访问未归属的书
- 登录用户创建的书会自动归属到当前用户
- 已归属的书只能被拥有者或系统 actor 访问

## 章节资源

### 列表 / 创建

```text
GET  /api/books/:bookId/chapters
POST /api/books/:bookId/chapters
```

### 查询 / 更新 / 删除

```text
GET    /api/books/:bookId/chapters/:chapterNo
PATCH  /api/books/:bookId/chapters/:chapterNo
DELETE /api/books/:bookId/chapters/:chapterNo
```

## 章节阶段接口

### 查看阶段内容

```text
GET /api/books/:bookId/chapters/:chapterNo/stages/:stage
```

可读阶段：

- `plan`
- `draft`
- `review`
- `final`

### 查看阶段历史

```text
GET /api/books/:bookId/chapters/:chapterNo/stages/:stage/history
```

### 写入阶段内容

```text
PUT /api/books/:bookId/chapters/:chapterNo/stages/:stage
```

可写阶段：

- `plan`
- `draft`
- `final`

### 查看工作流状态

```text
GET /api/books/:bookId/chapters/:chapterNo/workflow-state
GET /api/books/:bookId/chapters/:chapterNo/lifecycle
```

这两个接口在当前实现中等价。

## 阶段 Markdown 导入导出

### 导出为 Markdown

```text
GET /api/books/:bookId/chapters/:chapterNo/stages/:stage/export
```

### 从 Markdown 导入

```text
POST /api/books/:bookId/chapters/:chapterNo/stages/:stage/import
```

导入方式支持两种：

1. 直接发送 `text/markdown`
2. 发送 JSON：
   - `markdown`
   - `force`

## 设定类资源

这些资源都挂在书籍之下，遵循统一的 REST 风格：

- `outlines`
- `world-settings`
- `characters`
- `factions`
- `relations`
- `items`
- `hooks`

统一模式如下：

```text
GET    /api/books/:bookId/<resource>
POST   /api/books/:bookId/<resource>
GET    /api/books/:bookId/<resource>/:id
PATCH  /api/books/:bookId/<resource>/:id
DELETE /api/books/:bookId/<resource>/:id
```

例如：

```text
GET    /api/books/:bookId/characters
POST   /api/books/:bookId/characters
GET    /api/books/:bookId/characters/:id
PATCH  /api/books/:bookId/characters/:id
DELETE /api/books/:bookId/characters/:id
```

## 工作流接口

### 同步执行

```text
POST /api/workflows/plan
POST /api/workflows/draft
POST /api/workflows/review
POST /api/workflows/repair
POST /api/workflows/approve
POST /api/workflows/author-intent
POST /api/workflows/stage-summary
```

同步接口会等待执行完成后直接返回结果。

### 异步执行

```text
POST /api/workflows/plan/tasks
POST /api/workflows/draft/tasks
POST /api/workflows/review/tasks
POST /api/workflows/repair/tasks
POST /api/workflows/approve/tasks
POST /api/workflows/author-intent/tasks
```

异步接口会返回任务对象，调用方后续再轮询状态。

### 一个例外：approve dry run

当 `approve` 请求里带有 `dryRun=true` 时，当前实现不会走异步任务，而是直接同步执行并返回结果。

## 工作流任务接口

### 查询单个任务

```text
GET /api/workflow-tasks/:taskId
```

### 终止任务

```text
POST /api/workflow-tasks/:taskId/terminate
```

### 查询某章节历史任务

```text
GET /api/books/:bookId/chapters/:chapterNo/workflow-tasks
```

### 查询某类型的最新任务

```text
GET /api/books/:bookId/chapters/:chapterNo/workflow-tasks/latest?type=plan
```

`type` 必填。

## 用户设置

```text
GET    /api/user-settings/runtime
PUT    /api/user-settings/runtime
DELETE /api/user-settings/runtime
```

这一组接口要求登录。

## 嵌入索引刷新

```text
POST /api/books/:bookId/embeddings/refresh
```

用于同步重建当前书籍的嵌入索引。

## 推荐的集成顺序

如果你准备接一个外部前端或脚本，通常可以按这个顺序理解接口：

1. `/health`
2. `/api/auth/*`
3. `/api/books`
4. `/api/books/:bookId/chapters`
5. 章节阶段接口
6. `/api/workflows/*`
7. `/api/workflow-tasks/*`

## 相关文档

- [快速开始](./getting-started.md)
- [工作流说明](./workflows.md)
- [架构说明](./architecture.md)
- [排障指南](./troubleshooting.md)