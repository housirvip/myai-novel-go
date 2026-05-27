# 贡献指南

感谢你关注 `myai-novel-go`。

这份指南帮助你更快理解如何在这个仓库里提交 issue、改代码和发起 PR。第一版保持简洁，重点是让外部贡献者少走弯路。

## 开始之前

建议你先读这些文档：

- [README](./README.md)
- [快速开始](./docs/getting-started.md)
- [架构说明](./docs/architecture.md)
- [工作流说明](./docs/workflows.md)

## 本地开发建议

### 1. 使用默认配置先跑通

优先用：

- SQLite
- mock provider

这样不需要准备额外的数据库和模型 API Key。

### 2. 常用命令

```bash
make build
make server
make cli
make test
make test-race
make vet
make run
```

### 3. 提交前建议检查

至少运行：

```bash
make test
make vet
```

如果改动涉及并发、数据库或底层行为，建议额外运行：

```bash
make test-race
```

## 代码改动建议

这个项目更偏向直接、清晰、可维护的实现方式。

请尽量遵循这些原则：

- 先复用现有 service 和工作流结构
- 不要为一次性改动引入过度抽象
- 小改动尽量保持局部
- 新逻辑优先放在合适的 domain / handler / workflow 层
- 文档、命令、配置项改动要同步更新说明

## 提交 issue 时建议提供的信息

如果你要提 bug，请尽量带上：

- 你使用的运行方式（本地 / Docker / MySQL / SQLite）
- 关键 `.env` 配置片段（注意去掉密钥）
- 复现步骤
- 实际结果
- 预期结果
- 相关日志或报错信息

## Pull Request 建议

PR 说明里建议写清楚：

1. 你改了什么
2. 为什么要改
3. 怎么验证
4. 是否影响配置、部署或兼容性

如果改动涉及这些内容，也请同步更新：

- `README.md`
- `docs/`
- `.env.example`
- `docker-compose.yml`

## 哪些改动尤其欢迎

- 文档完善
- 配置或部署体验优化
- 工作流稳定性改进
- 测试补充
- API / CLI 可用性优化
- 排障体验改进

## 安全问题

如果你发现的是安全问题，请不要直接公开提交 issue，优先查看：

- [SECURITY.md](./SECURITY.md)

## 行为准则

参与本项目即表示你同意遵守：

- [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)
