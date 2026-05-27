# 文档总览

欢迎来到 `myai-novel-go` 的文档目录。

如果你是第一次接触这个项目，建议先从 [README](../README.md) 开始；如果你已经知道自己要解决什么问题，可以直接按下面的分类进入对应文档。

## 我只想先把项目跑起来

按这个顺序阅读最合适：

1. [快速开始](./getting-started.md)
2. [配置说明](./configuration.md)
3. [部署说明](./deployment.md)

适合：

- 第一次克隆仓库
- 想先本地体验
- 想快速验证服务可用性

## 我想理解核心工作流

优先看：

1. [工作流说明](./workflows.md)
2. [API 总览](./api-overview.md)
3. [CLI 使用说明](./cli.md)

适合：

- 想调用 plan / draft / review / repair / approve
- 想理解同步和异步任务差异
- 想把这个系统接到自己的前端或脚本里

## 我想通过接口或命令行集成

优先看：

- [API 总览](./api-overview.md)
- [CLI 使用说明](./cli.md)
- [配置说明](./configuration.md)

适合：

- 做 HTTP 集成
- 写自动化脚本
- 想用 CLI 管理数据和章节

## 我想理解内部实现

优先看：

- [架构说明](./architecture.md)
- [工作流说明](./workflows.md)
- [从 Node 版本迁移](./migration-from-node.md)

适合：

- 新加入的维护者
- 想改工作流或服务端逻辑的人
- 想理解 CLI 和 HTTP 为什么共用一套 service 的人

## 我已经有设定文件，想快速导入

优先看：

- [从设定文件初始化项目](./importing-settings.md)
- [CLI 使用说明](./cli.md)

适合：

- 已有 markdown 设定集
- 想快速初始化一本书的基础数据

## 我正在从旧系统迁移

优先看：

- [从 Node 版本迁移](./migration-from-node.md)
- [排障指南](./troubleshooting.md)

适合：

- 你已经在用 `myai-novel` Node 版
- 想迁数据库到 Go 版
- 想验证迁移后是否正常

## 我遇到问题了

先看：

- [排障指南](./troubleshooting.md)
- [配置说明](./configuration.md)
- [部署说明](./deployment.md)

适合：

- 服务起不来
- 数据库连接失败
- 工作流任务堵塞
- 模型配置报错

## 我想参与贡献

建议阅读：

- [贡献指南](../CONTRIBUTING.md)
- [安全说明](../SECURITY.md)
- [行为准则](../CODE_OF_CONDUCT.md)
- [架构说明](./architecture.md)

## 全部文档列表

### 使用类

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [工作流说明](./workflows.md)
- [CLI 使用说明](./cli.md)
- [部署说明](./deployment.md)

### 深入类

- [API 总览](./api-overview.md)
- [架构说明](./architecture.md)
- [从设定文件初始化项目](./importing-settings.md)
- [从 Node 版本迁移](./migration-from-node.md)
- [排障指南](./troubleshooting.md)
