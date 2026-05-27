# 从 Node 版本迁移

`myai-novel-go` 的定位之一，就是作为原 `myai-novel` Node.js 服务端的 Go 重写版。这意味着如果你已经在使用 Node 版本，可以比较平滑地迁移数据和调用方式。

## 迁移工具位置

仓库内置了迁移工具：

- `cmd/migrate-from-node/main.go`

通常会先编译成：

```bash
go build -o bin/migrate-from-node ./cmd/migrate-from-node
```

## 迁移思路

这个工具的思路很直接：

- 因为两边表结构对齐
- 所以迁移本质上是逐表读取再批量写入
- 主键 ID 会保留

它不是做“字段映射转换”，而是做“同构数据搬运”。

## 支持的源与目标

支持：

- SQLite -> SQLite
- SQLite -> MySQL
- MySQL -> MySQL

目标端默认可以直接读取当前 `.env` 里的数据库配置。

## 最常见的迁移方式

### 从 Node 版本的 SQLite 迁到当前 Go 项目的 SQLite

先编译工具：

```bash
go build -o bin/migrate-from-node ./cmd/migrate-from-node
```

先 dry run：

```bash
DB_SQLITE_PATH=./data/novel.db ./bin/migrate-from-node \
  --source ../myai-novel/data/novel.db \
  --dry-run
```

确认没问题后执行真实迁移：

```bash
DB_SQLITE_PATH=./data/novel.db ./bin/migrate-from-node \
  --source ../myai-novel/data/novel.db
```

## 使用 MySQL 作为源或目标

### MySQL 作为源

```bash
./bin/migrate-from-node \
  --source-driver mysql \
  --source-dsn 'user:pass@tcp(127.0.0.1:3306)/myai_novel?parseTime=true&loc=UTC'
```

### 显式指定目标数据库

```bash
./bin/migrate-from-node \
  --source ../myai-novel/data/novel.db \
  --target-driver mysql \
  --target-dsn 'user:pass@tcp(127.0.0.1:3306)/myai_novel?parseTime=true&loc=UTC'
```

## 可选参数

常用参数包括：

- `--source`
- `--source-driver`
- `--source-dsn`
- `--target-driver`
- `--target-dsn`
- `--batch`
- `--dry-run`

### batch

默认批量写入大小为：

- `500`

如果你在迁移大库时有特殊性能需求，可以调整这个值。

## 目标库要求

迁移前最重要的一点：

- **目标表应该是空的**

原因是：

- 工具会保留原始主键 ID
- 如果目标库里已有数据，很容易发生主键冲突

工具不会自动帮你清空目标库。

这意味着你应该显式地：

- 删除 SQLite 文件
- 或重置测试数据库
- 或准备一个全新的目标库

## 工具内部会做什么

迁移工具会：

1. 打开源数据库
2. 打开目标数据库
3. 确保目标 schema 已迁移到位
4. 遍历 `models.All()` 中的表
5. 分页读取源表数据
6. 用 `CreateInBatches` 写入目标库

## 推荐迁移步骤

推荐你按这个顺序做：

1. 先备份原始数据库
2. 准备一个空目标库
3. 先跑 `--dry-run`
4. 再执行真实迁移
5. 启动 `myai-novel-go`
6. 用 `/health` 和最小工作流做一次冒烟验证

## 迁移后建议验证什么

至少检查：

- 书籍数量是否正确
- 章节数量是否正确
- 人物 / 势力 / 世界设定是否存在
- 某个章节阶段是否能正常读取
- 工作流是否还能继续运行

## 迁移不是配置迁移

这个工具迁移的是数据库内容，不包括：

- `.env` 配置
- API Key
- 运行环境变量
- 外部部署配置

这些内容仍然需要你在 Go 项目侧单独准备。

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [架构说明](./architecture.md)
- [排障指南](./troubleshooting.md)