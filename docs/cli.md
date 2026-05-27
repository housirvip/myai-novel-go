# CLI 使用说明

`myai-novel-go` 自带一个名为 `novel` 的命令行工具，用来直接操作数据库中的小说数据，并运行核心工作流。

CLI 和 HTTP 服务共享同一套 domain service，因此很适合：

- 本地建库
- 快速造测试数据
- 章节导入导出
- 工作流冒烟
- 脚本化调用

## 构建 CLI

```bash
make cli
```

构建后可执行文件路径：

```text
./bin/novel
```

查看帮助：

```bash
./bin/novel --help
```

## CLI 的命令分组

当前顶层命令主要包括：

- `db`
- `book`
- `chapter`
- `outline`
- `world`
- `character`
- `faction`
- `relation`
- `item`
- `hook`
- `plan`
- `draft`
- `review`
- `repair`
- `approve`

## 数据库命令

### 初始化数据库

```bash
./bin/novel db init
```

### 再次执行迁移

```bash
./bin/novel db migrate
```

### 检查数据库连通性

```bash
./bin/novel db check
```

### 重置 SQLite 文件

```bash
./bin/novel db reset
```

注意：`db reset` 只支持 SQLite。

## 书籍命令

### 创建一本书

```bash
./bin/novel book create --title 测试书 --target-chapter-count 100
```

### 查看书籍列表

```bash
./bin/novel book list
```

### 查看一本书

```bash
./bin/novel book show --id 1
```

### 更新一本书

```bash
./bin/novel book update --id 1 --title 新书名
```

### 删除一本书

```bash
./bin/novel book delete --id 1
```

## 章节命令

### 创建章节

```bash
./bin/novel chapter create --book 1 --chapter 1 --title 第一章
```

### 列出章节

```bash
./bin/novel chapter list --book 1
```

### 查看章节

```bash
./bin/novel chapter show --book 1 --chapter 1
```

### 更新章节元信息

```bash
./bin/novel chapter update --book 1 --chapter 1 --title 新标题
```

### 删除章节

```bash
./bin/novel chapter delete --book 1 --chapter 1
```

## 阶段导出与导入

### 导出章节阶段为 Markdown

```bash
./bin/novel chapter export --book 1 --chapter 1 --stage final --output /tmp/ch1-final.md
```

### 从 Markdown 回写阶段内容

```bash
./bin/novel chapter import --book 1 --chapter 1 --stage draft --input /tmp/ch1-final.md --force
```

支持导出的常见阶段：

- `plan`
- `draft`
- `review`
- `final`

支持导入的阶段：

- `plan`
- `draft`
- `final`

## 资源 CRUD 命令

CLI 还提供多个资源的基础 CRUD 能力：

- `outline`
- `world`
- `character`
- `faction`
- `relation`
- `item`
- `hook`

例如创建人物：

```bash
./bin/novel character create --book 1 --name 林夜 --keywords 林夜
```

例如创建势力：

```bash
./bin/novel faction create --book 1 --name 青岳宗 --keywords 青岳宗
```

例如创建世界设定：

```bash
./bin/novel world create --book 1 --title 修炼体系 --category system --content "炼体、凝气、筑基" --keywords 修炼
```

## 工作流命令

CLI 支持直接运行同步工作流。

### plan

```bash
./bin/novel plan --book 1 --chapter 1 --provider mock --author-intent 入门
```

### draft

```bash
./bin/novel draft --book 1 --chapter 1 --provider mock
```

### review

```bash
./bin/novel review --book 1 --chapter 1 --provider mock
```

### repair

```bash
./bin/novel repair --book 1 --chapter 1 --provider mock
```

### approve

```bash
./bin/novel approve --book 1 --chapter 1 --provider mock
```

### approve dry run

```bash
./bin/novel approve --book 1 --chapter 1 --provider mock --dry-run
```

## 一个典型的 CLI 使用流程

```bash
make cli
./bin/novel db init
./bin/novel book create --title 测试书 --target-chapter-count 100
./bin/novel chapter create --book 1 --chapter 1 --title 第一章
./bin/novel character create --book 1 --name 林夜 --keywords 林夜
./bin/novel plan --book 1 --chapter 1 --provider mock --author-intent 入门
./bin/novel draft --book 1 --chapter 1 --provider mock
```

## CLI 的定位说明

CLI 当前更偏向：

- 快速建库
- 跑工作流冒烟
- 导入导出章节
- 脚本辅助操作

它并不追求和 HTTP API 在所有复杂字段上完全逐项对齐。对于更细粒度的字段操作，更适合使用 HTTP API 或直接在上层系统中集成。

## 相关文档

- [快速开始](./getting-started.md)
- [工作流说明](./workflows.md)
- [配置说明](./configuration.md)
- [部署说明](./deployment.md)