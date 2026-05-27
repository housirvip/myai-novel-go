# myai-novel-go

[English README](./README.en.md)

`myai-novel-go` 是 `myai-novel` 服务端的 Go 重写版，面向小说创作工作流场景，重点优化并发执行、运行稳定性和部署体验。

它基于 **Gin + GORM** 构建，提供：

- HTTP API 服务
- 本地 CLI 工具
- 多阶段章节工作流
- SQLite / MySQL 双数据库支持
- mock / OpenAI / Anthropic / custom LLM provider 支持
- Docker Compose 快速部署方式

## 一眼看懂这个项目

如果你是第一次访问这个仓库，可以先记住三件事：

1. **它是什么**：一个面向 AI 小说创作工作流的 Go 后端。
2. **怎么最快跑起来**：复制 `.env.example`，然后 `make run`。
3. **详细内容去哪里看**：从 [文档总览](./docs/index.md) 进入对应文档。

## 项目特点

相比 Node.js 版本，这个项目重点强化了以下能力：

1. **真正的并发执行**：通过 goroutine + worker pool 异步执行长耗时工作流。
2. **更稳定的数据库访问**：支持 GORM 连接池、SQLite WAL 和 MySQL。
3. **更明确的限流与超时控制**：LLM 请求带全局限流与超时保护。
4. **更清晰的服务边界**：HTTP 服务、CLI、工作流和 domain service 共享统一后端能力。

## 适合谁使用

这个仓库适合：

- 想快速搭建 AI 小说创作后端的人
- 想在本地用 mock provider 跑通工作流的人
- 想通过 HTTP API 或 CLI 管理书籍、章节和设定的人
- 想从 Node 版本逐步迁移到 Go 服务端的人

## 快速开始

### 1. 准备环境变量

```bash
cp .env.example .env
```

默认配置已经可以本地跑通：

- `DB_CLIENT=sqlite`
- `LLM_PROVIDER=mock`
- `SERVER_PORT=3000`

### 2. 编译并启动服务

```bash
make server
make run
```

### 3. 做一次健康检查

```bash
curl http://127.0.0.1:3000/health
```

### 4. 继续阅读

如果你想继续跑通最小流程，请看：

- [快速开始文档](./docs/getting-started.md)
- [文档总览](./docs/index.md)

## 文档导航

### 先看哪篇最合适

- 我只想先把项目跑起来：看 [快速开始](./docs/getting-started.md)
- 我想知道环境变量怎么配：看 [配置说明](./docs/configuration.md)
- 我想理解 plan/draft/review/repair/approve：看 [工作流说明](./docs/workflows.md)
- 我想接 HTTP API：看 [API 总览](./docs/api-overview.md)
- 我想用 CLI：看 [CLI 使用说明](./docs/cli.md)
- 我想理解内部实现：看 [架构说明](./docs/architecture.md)
- 我遇到问题了：看 [排障指南](./docs/troubleshooting.md)

### 完整文档入口

- [文档总览](./docs/index.md)

### 面向首次使用者

- [快速开始](./docs/getting-started.md)
- [配置说明](./docs/configuration.md)
- [工作流说明](./docs/workflows.md)
- [CLI 使用说明](./docs/cli.md)
- [部署说明](./docs/deployment.md)

### 面向集成与贡献者

- [API 总览](./docs/api-overview.md)
- [架构说明](./docs/architecture.md)
- [从设定文件初始化项目](./docs/importing-settings.md)
- [从 Node 版本迁移](./docs/migration-from-node.md)
- [排障指南](./docs/troubleshooting.md)
- [贡献指南](./CONTRIBUTING.md)
- [安全说明](./SECURITY.md)
- [行为准则](./CODE_OF_CONDUCT.md)

## 主要能力概览

### HTTP API

服务端提供：

- 健康检查接口
- 书籍、章节、人物、设定等资源 CRUD
- 章节阶段读取与写入
- 工作流同步 / 异步执行
- 任务轮询与终止
- 认证与用户设置接口

### CLI

CLI 二进制名为 `novel`，支持：

- 数据库初始化与检查
- 书籍 / 章节 / 设定资源操作
- 阶段 Markdown 导入导出
- plan / draft / review / repair / approve 工作流执行

### 工作流

当前核心工作流阶段包括：

```text
plan -> draft -> review -> repair -> approve
```

既支持同步执行，也支持异步提交任务后轮询结果。

## 默认运行方式

项目默认强调“先跑通，再扩展”：

- 默认数据库：SQLite
- 默认 LLM：mock provider
- 默认监听：`127.0.0.1:3000`
- 默认可以不依赖任何真实模型 Key

这让仓库非常适合本地体验、开发调试和写集成脚本。

## Docker Compose

如果你想直接用容器启动：

```bash
make docker-up
```

默认会以 SQLite + mock provider 启动服务。

更多部署细节见：

- [部署说明](./docs/deployment.md)

## 当前范围与已知说明

当前仓库已经具备完整后端主链路，但你在阅读和使用时需要注意：

- 顶层 README 现在作为入口文档，详细说明已拆分到 `docs/`
- 已提供 docs、贡献指南、安全说明、行为准则和 MIT License，适合作为开源仓库基础骨架
- 默认推荐先使用 SQLite + mock provider 跑通流程
- CLI 更适合快速建库、冒烟和脚本化操作，不追求覆盖所有复杂字段
- WebUI 静态托管是可选能力，需要通过 `WEBUI_DIST_PATH` 指向已有构建产物

## 常用命令

```bash
make build
make server
make cli
make test
make test-race
make vet
make run
make docker-up
make docker-down
```

## 技术栈

- Go
- Gin
- GORM
- Cobra
- Zap
- SQLite / MySQL

## 开源配套文件

根目录已经提供：

- [贡献指南](./CONTRIBUTING.md)
- [安全说明](./SECURITY.md)
- [行为准则](./CODE_OF_CONDUCT.md)
- [MIT License](./LICENSE)

## 后续可以继续完善的方向

如果你想继续打磨这个仓库的开源展示效果，下一步通常会优先考虑：

- 补一版英文 README 或双语文档
- 增加 API 示例集合或 OpenAPI 文档
- 增加架构图或请求流转图
- 增加更完整的部署与发布说明
