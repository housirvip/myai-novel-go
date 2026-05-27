# 快速开始

这篇文档面向第一次接触 `myai-novel-go` 的读者，目标是在最短时间内把服务跑起来，并用默认的 `SQLite + mock provider` 跑通一条最小链路。

## 这个项目是什么

`myai-novel-go` 是 `myai-novel` 服务端的 Go 重写版，核心目标是把小说创作工作流做成更稳、更易部署、并发能力更强的后端服务。

项目当前提供两种主要使用方式：

- HTTP 服务：适合 WebUI、脚本集成和接口调用。
- CLI 工具：适合本地初始化数据、导入导出、快速冒烟和脚本化操作。

## 运行前准备

你至少需要：

- Go
- 可用的 shell 环境
- `git`

如果你只想本地体验默认流程，不需要提前准备：

- MySQL
- OpenAI / Anthropic API Key
- WebUI 构建产物

默认配置就是：

- 数据库：SQLite
- LLM provider：mock
- 服务地址：`127.0.0.1:3000`

## 最快启动路径

### 1. 准备环境变量

```bash
cp .env.example .env
```

默认情况下，`.env` 已经足够本地跑通。

### 2. 编译服务端

```bash
make server
```

如果你也想使用 CLI，可以额外编译：

```bash
make cli
```

或者一次性构建两个二进制：

```bash
make build
```

### 3. 启动服务

```bash
make run
```

正常情况下，你会看到类似日志：

```text
config.loaded
 db.connected
 db.migrate.ok
 server.starting
```

默认监听地址：

- `http://127.0.0.1:3000`

## 最小冒烟流程

下面这组命令可以验证服务、数据库和工作流链路是否都正常。

### 1. 健康检查

```bash
curl http://127.0.0.1:3000/health
```

### 2. 创建一本书和第一章

```bash
curl -X POST http://127.0.0.1:3000/api/books \
  -H 'Content-Type: application/json' \
  -d '{"title":"青岳入门录","targetChapterCount":200}'

curl -X POST http://127.0.0.1:3000/api/books/1/chapters \
  -H 'Content-Type: application/json' \
  -d '{"chapterNo":1,"title":"黑铁令"}'
```

### 3. 补一条最小设定数据

```bash
curl -X POST http://127.0.0.1:3000/api/books/1/characters \
  -H 'Content-Type: application/json' \
  -d '{"name":"林夜","keywords":"林夜"}'
```

### 4. 跑一个同步工作流

```bash
curl -X POST http://127.0.0.1:3000/api/workflows/plan \
  -H 'Content-Type: application/json' \
  -d '{"bookId":1,"chapterNo":1,"provider":"mock","authorIntent":"林夜入宗"}'
```

### 5. 查看阶段结果

```bash
curl http://127.0.0.1:3000/api/books/1/chapters/1/stages/plan
```

如果你想继续跑完整链路，可以依次调用：

- `/api/workflows/draft`
- `/api/workflows/review`
- `/api/workflows/repair`
- `/api/workflows/approve`

## 使用异步任务

如果你不希望 HTTP 请求一直等待结果，可以使用 `.../tasks` 形式的异步接口：

```bash
TASK=$(curl -s -X POST http://127.0.0.1:3000/api/workflows/plan/tasks \
  -H 'Content-Type: application/json' \
  -d '{"bookId":1,"chapterNo":2,"provider":"mock"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')

curl http://127.0.0.1:3000/api/workflow-tasks/$TASK
```

你可以轮询这个任务接口，直到状态变成 `succeeded` 或 `failed`。

## 用 CLI 跑一遍最小流程

如果你更喜欢命令行，也可以这样操作：

```bash
make cli
./bin/novel db init
./bin/novel book create --title 测试书 --target-chapter-count 100
./bin/novel chapter create --book 1 --chapter 1 --title 第一章
./bin/novel plan --book 1 --chapter 1 --provider mock --author-intent 入门
```

更多 CLI 用法见 [CLI 使用说明](./cli.md)。

## 用 Docker Compose 启动

如果你更想快速体验容器化部署：

```bash
make docker-up
```

然后访问：

```bash
curl http://127.0.0.1:3000/health
```

更多容器部署信息见 [部署说明](./deployment.md)。

## 常见下一步

跑通后，通常接下来会看这些文档：

- [配置说明](./configuration.md)：了解数据库、LLM、认证、并发配置。
- [工作流说明](./workflows.md)：理解 plan/draft/review/repair/approve 的执行方式。
- [CLI 使用说明](./cli.md)：通过命令行管理数据和章节。
- [部署说明](./deployment.md)：用 Docker Compose 或 MySQL 部署。