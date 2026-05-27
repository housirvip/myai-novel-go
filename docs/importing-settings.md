# 从设定文件初始化项目

仓库里自带了一个初始化脚本，可以把一组 markdown 设定文件转换成数据库中的书籍、人物、势力、世界设定等数据，适合快速导入一个已有的小说设定集。

脚本位置：

- `scripts/init_novel_from_settings.sh`

## 这个脚本适合什么场景

适合：

- 你已经有一套 markdown 设定文件
- 你想快速把设定导入到 `myai-novel-go`
- 你希望在跑工作流前先把基础世界观和角色信息灌进去

不适合：

- 作为公开稳定 API 给第三方长期依赖
- 需要高度自定义字段映射的复杂迁移场景

## 前置条件

运行脚本前，你需要准备：

- `jq`
- `python3`
- 已构建好的 CLI：`./bin/novel`

如果 CLI 还没编译：

```bash
make cli
```

## 默认输入目录

脚本默认会读取：

```text
/home/idea/codex/myai-novel/test-project/settings
```

你也可以在执行时传入自己的目录。

## 必需的输入文件

脚本会检查这些文件是否存在：

- `title_pitch.md`
- `volume_outline.md`
- `world_setting.md`
- `character_setting.md`
- `faction_location_setting.md`
- `profession_system.md`
- `currency_system.md`

缺任何一个，脚本都会直接退出。

## 脚本会做什么

脚本大致流程是：

1. 读取各 markdown 文件
2. 用内嵌 Python 做内容抽取与整理
3. 生成中间 JSON 数据
4. 调用 `novel` CLI 创建书籍
5. 逐步创建世界设定、人物、势力、大纲、关系等资源

也就是说，它不是直接写数据库，而是复用项目自己的 CLI 和 service 层。

## 最简单的使用方式

```bash
./scripts/init_novel_from_settings.sh /path/to/settings
```

如果你的 `novel` 可执行文件不在默认位置，可以指定：

```bash
NOVEL_BIN=./bin/novel ./scripts/init_novel_from_settings.sh /path/to/settings
```

## Dry run

如果你想先看看脚本准备执行哪些命令，可以用 dry run：

```bash
DRY_RUN=1 ./scripts/init_novel_from_settings.sh /path/to/settings
```

在这个模式下，脚本不会真正写入数据库，而是把将要执行的命令打印出来。

## 导入前建议

在真实导入前，建议你先：

1. 确认 `.env` 指向正确数据库
2. 先跑 `./bin/novel db check`
3. 先用 `DRY_RUN=1` 预览
4. 在空库或测试库中先试一次

## 可能导入的资源类型

根据脚本逻辑，通常会创建：

- book
- worlds
- characters
- factions
- outlines
- relations

具体能导入哪些内容，取决于你的设定文件结构和可解析程度。

## 常见问题

### 提示 `missing command: jq`

说明当前环境没有安装 `jq`。

### 提示 `novel binary not found or not executable`

说明 CLI 没有编译或不可执行，先运行：

```bash
make cli
```

### 提示缺少某个 markdown 文件

检查输入目录是否正确，以及是否包含脚本要求的全部文件。

## 什么时候更适合直接用 API 或 CLI

如果你只是少量手工录入设定，通常更推荐：

- HTTP API
- `./bin/novel` CLI

这个脚本更像一个“批量初始化工具”，而不是日常编辑入口。

## 相关文档

- [CLI 使用说明](./cli.md)
- [快速开始](./getting-started.md)
- [从 Node 版本迁移](./migration-from-node.md)
- [排障指南](./troubleshooting.md)