package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"myai-novel-go/internal/domain/character"
	"myai-novel-go/internal/domain/faction"
	"myai-novel-go/internal/domain/item"
	"myai-novel-go/internal/domain/outline"
	"myai-novel-go/internal/domain/relation"
	"myai-novel-go/internal/domain/story_hook"
	"myai-novel-go/internal/domain/world_setting"
)

// 注:这里 7 个资源命令保留最常用的 create/list/show/update/delete 入口,
// 复杂字段(如 character.professions / faction.leader)通过 HTTP API 或直接编辑 DB 设置;
// CLI 主要服务于"快速建库 + 跑工作流冒烟",不追求与原 Node 项目逐字段对齐。

func newOutlineCmd() *cobra.Command {
	c := &cobra.Command{Use: "outline", Short: "管理大纲"}
	c.AddCommand(
		simpleCreate("create", "创建大纲", []flag{
			{name: "book", kind: "int64", required: true, desc: "书 ID"},
			{name: "title", kind: "string", required: true, desc: "标题"},
			{name: "story-core", kind: "string", desc: "故事核"},
			{name: "main-plot", kind: "string", desc: "主线"},
			{name: "chapter-start", kind: "int", desc: "章节区间起点"},
			{name: "chapter-end", kind: "int", desc: "章节区间终点"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			return ctn.Outline.Create(ctx, outline.CreateInput{
				BookID:         fs.Int64("book"),
				Title:          fs.Str("title"),
				StoryCore:      stringPtr(fs.Str("story-core")),
				MainPlot:       stringPtr(fs.Str("main-plot")),
				ChapterStartNo: intPtr(fs.Int("chapter-start")),
				ChapterEndNo:   intPtr(fs.Int("chapter-end")),
			})
		}),
		simpleList("list", "列出大纲", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.Outline.List(ctx, bookID, limit)
		}),
		simpleGet("get", "查看大纲", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.Outline.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新大纲(支持 --title/--story-core/--main-plot)", []string{"title", "story-core", "main-plot"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := outline.UpdateInput{}
			if v := fs.Str("title"); v != "" {
				in.Title = &v
			}
			if v := fs.Str("story-core"); v != "" {
				in.StoryCore = &v
			}
			if v := fs.Str("main-plot"); v != "" {
				in.MainPlot = &v
			}
			return ctn.Outline.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除大纲", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.Outline.Remove(ctx, bookID, id)
		}),
	)
	return c
}

func newWorldCmd() *cobra.Command {
	c := &cobra.Command{Use: "world", Short: "管理世界设定"}
	c.AddCommand(
		simpleCreate("create", "创建世界设定", []flag{
			{name: "book", kind: "int64", required: true},
			{name: "title", kind: "string", required: true},
			{name: "category", kind: "string", required: true, desc: "类别(如 magic / society)"},
			{name: "content", kind: "string", required: true, desc: "正文"},
			{name: "keywords", kind: "string"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			return ctn.World.Create(ctx, world_setting.CreateInput{
				BookID: fs.Int64("book"), Title: fs.Str("title"),
				Category: fs.Str("category"), Content: fs.Str("content"),
				Keywords: stringPtr(fs.Str("keywords")),
			})
		}),
		simpleList("list", "列出世界设定", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.World.List(ctx, bookID, limit, "")
		}),
		simpleGet("get", "查看世界设定", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.World.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新世界设定", []string{"title", "content", "keywords"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := world_setting.UpdateInput{}
			if v := fs.Str("title"); v != "" {
				in.Title = &v
			}
			if v := fs.Str("content"); v != "" {
				in.Content = &v
			}
			if v := fs.Str("keywords"); v != "" {
				in.Keywords = &v
			}
			return ctn.World.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除世界设定", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.World.Remove(ctx, bookID, id)
		}),
	)
	return c
}

func newCharacterCmd() *cobra.Command {
	c := &cobra.Command{Use: "character", Short: "管理人物"}
	c.AddCommand(
		simpleCreate("create", "创建人物", []flag{
			{name: "book", kind: "int64", required: true},
			{name: "name", kind: "string", required: true},
			{name: "alias", kind: "string"},
			{name: "personality", kind: "string"},
			{name: "background", kind: "string"},
			{name: "goal", kind: "string"},
			{name: "keywords", kind: "string"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			return ctn.Character.Create(ctx, character.CreateInput{
				BookID:      fs.Int64("book"),
				Name:        fs.Str("name"),
				Alias:       stringPtr(fs.Str("alias")),
				Personality: stringPtr(fs.Str("personality")),
				Background:  stringPtr(fs.Str("background")),
				Goal:        stringPtr(fs.Str("goal")),
				Keywords:    stringPtr(fs.Str("keywords")),
			})
		}),
		simpleList("list", "列出人物", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.Character.List(ctx, bookID, limit, "")
		}),
		simpleGet("get", "查看人物", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.Character.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新人物", []string{"name", "alias", "background", "goal", "keywords"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := character.UpdateInput{}
			if v := fs.Str("name"); v != "" {
				in.Name = &v
			}
			if v := fs.Str("alias"); v != "" {
				in.Alias = &v
			}
			if v := fs.Str("background"); v != "" {
				in.Background = &v
			}
			if v := fs.Str("goal"); v != "" {
				in.Goal = &v
			}
			if v := fs.Str("keywords"); v != "" {
				in.Keywords = &v
			}
			return ctn.Character.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除人物", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.Character.Remove(ctx, bookID, id)
		}),
	)
	return c
}

func newFactionCmd() *cobra.Command {
	c := &cobra.Command{Use: "faction", Short: "管理势力"}
	c.AddCommand(
		simpleCreate("create", "创建势力", []flag{
			{name: "book", kind: "int64", required: true},
			{name: "name", kind: "string", required: true},
			{name: "category", kind: "string"},
			{name: "core-goal", kind: "string"},
			{name: "description", kind: "string"},
			{name: "keywords", kind: "string"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			return ctn.Faction.Create(ctx, faction.CreateInput{
				BookID:      fs.Int64("book"),
				Name:        fs.Str("name"),
				Category:    stringPtr(fs.Str("category")),
				CoreGoal:    stringPtr(fs.Str("core-goal")),
				Description: stringPtr(fs.Str("description")),
				Keywords:    stringPtr(fs.Str("keywords")),
			})
		}),
		simpleList("list", "列出势力", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.Faction.List(ctx, bookID, limit, "")
		}),
		simpleGet("get", "查看势力", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.Faction.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新势力", []string{"name", "core-goal", "description", "keywords"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := faction.UpdateInput{}
			if v := fs.Str("name"); v != "" {
				in.Name = &v
			}
			if v := fs.Str("core-goal"); v != "" {
				in.CoreGoal = &v
			}
			if v := fs.Str("description"); v != "" {
				in.Description = &v
			}
			if v := fs.Str("keywords"); v != "" {
				in.Keywords = &v
			}
			return ctn.Faction.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除势力", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.Faction.Remove(ctx, bookID, id)
		}),
	)
	return c
}

func newRelationCmd() *cobra.Command {
	c := &cobra.Command{Use: "relation", Short: "管理关系"}
	c.AddCommand(
		simpleCreate("create", "创建关系", []flag{
			{name: "book", kind: "int64", required: true},
			{name: "source-type", kind: "string", required: true, desc: "character|faction|item"},
			{name: "source-id", kind: "int64", required: true},
			{name: "target-type", kind: "string", required: true, desc: "character|faction|item"},
			{name: "target-id", kind: "int64", required: true},
			{name: "type", kind: "string", required: true, desc: "关系类型"},
			{name: "description", kind: "string"},
			{name: "keywords", kind: "string"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			return ctn.Relation.Create(ctx, relation.CreateInput{
				BookID:       fs.Int64("book"),
				SourceType:   fs.Str("source-type"),
				SourceID:     fs.Int64("source-id"),
				TargetType:   fs.Str("target-type"),
				TargetID:     fs.Int64("target-id"),
				RelationType: fs.Str("type"),
				Description:  stringPtr(fs.Str("description")),
				Keywords:     stringPtr(fs.Str("keywords")),
			})
		}),
		simpleList("list", "列出关系", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.Relation.List(ctx, bookID, limit)
		}),
		simpleGet("get", "查看关系", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.Relation.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新关系", []string{"description", "keywords"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := relation.UpdateInput{}
			if v := fs.Str("description"); v != "" {
				in.Description = &v
			}
			if v := fs.Str("keywords"); v != "" {
				in.Keywords = &v
			}
			return ctn.Relation.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除关系", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.Relation.Remove(ctx, bookID, id)
		}),
	)
	return c
}

func newItemCmd() *cobra.Command {
	c := &cobra.Command{Use: "item", Short: "管理物品"}
	c.AddCommand(
		simpleCreate("create", "创建物品", []flag{
			{name: "book", kind: "int64", required: true},
			{name: "name", kind: "string", required: true},
			{name: "owner-type", kind: "string", required: true, desc: "character|faction|none"},
			{name: "owner-id", kind: "int64"},
			{name: "category", kind: "string"},
			{name: "description", kind: "string"},
			{name: "keywords", kind: "string"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			oid := fs.Int64("owner-id")
			var oidPtr *int64
			if oid != 0 {
				oidPtr = &oid
			}
			return ctn.Item.Create(ctx, item.CreateInput{
				BookID:      fs.Int64("book"),
				Name:        fs.Str("name"),
				OwnerType:   fs.Str("owner-type"),
				OwnerID:     oidPtr,
				Category:    stringPtr(fs.Str("category")),
				Description: stringPtr(fs.Str("description")),
				Keywords:    stringPtr(fs.Str("keywords")),
			})
		}),
		simpleList("list", "列出物品", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.Item.List(ctx, bookID, limit)
		}),
		simpleGet("get", "查看物品", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.Item.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新物品", []string{"name", "description", "keywords"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := item.UpdateInput{}
			if v := fs.Str("name"); v != "" {
				in.Name = &v
			}
			if v := fs.Str("description"); v != "" {
				in.Description = &v
			}
			if v := fs.Str("keywords"); v != "" {
				in.Keywords = &v
			}
			return ctn.Item.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除物品", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.Item.Remove(ctx, bookID, id)
		}),
	)
	return c
}

func newHookCmd() *cobra.Command {
	c := &cobra.Command{Use: "hook", Short: "管理伏笔/钩子"}
	c.AddCommand(
		simpleCreate("create", "创建钩子", []flag{
			{name: "book", kind: "int64", required: true},
			{name: "title", kind: "string", required: true},
			{name: "hook-type", kind: "string", desc: "mystery|conflict|...,可选"},
			{name: "description", kind: "string"},
			{name: "target-chapter", kind: "int"},
			{name: "keywords", kind: "string"},
		}, func(ctx context.Context, ctn *Container, fs flagSet) (any, error) {
			return ctn.Hook.Create(ctx, story_hook.CreateInput{
				BookID:          fs.Int64("book"),
				Title:           fs.Str("title"),
				HookType:        stringPtr(fs.Str("hook-type")),
				Description:     stringPtr(fs.Str("description")),
				TargetChapterNo: intPtr(fs.Int("target-chapter")),
				Keywords:        stringPtr(fs.Str("keywords")),
			})
		}),
		simpleList("list", "列出钩子", func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error) {
			return ctn.Hook.List(ctx, bookID, limit, "")
		}),
		simpleGet("get", "查看钩子", func(ctx context.Context, ctn *Container, bookID, id int64) (any, error) {
			return ctn.Hook.Get(ctx, bookID, id)
		}),
		simpleUpdate("update", "更新钩子", []string{"title", "description", "keywords"}, func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error) {
			in := story_hook.UpdateInput{}
			if v := fs.Str("title"); v != "" {
				in.Title = &v
			}
			if v := fs.Str("description"); v != "" {
				in.Description = &v
			}
			if v := fs.Str("keywords"); v != "" {
				in.Keywords = &v
			}
			return ctn.Hook.Update(ctx, bookID, id, in)
		}),
		simpleDelete("delete", "删除钩子", func(ctx context.Context, ctn *Container, bookID, id int64) error {
			return ctn.Hook.Remove(ctx, bookID, id)
		}),
	)
	return c
}

// ----- 通用 get / update / delete-by-id helpers -----

// 注:为减少为 7 个资源各写一份 update flag 解析的重复,
// 这里 update 只支持 --keywords 与 --status 两个最常用字段;
// 复杂字段建议通过 HTTP API 或直接 SQL 修改。

// 这里在每个资源命令的 simpleCreate/list/delete 之间会再追加 get + update。
// 因为各资源 Get 返回的 model 类型不同,我们在每个具体命令里就地 closure。

// ---------------------------------------------------

type flag struct {
	name     string
	kind     string // "string" / "int" / "int64"
	required bool
	desc     string
}

type flagSet struct{ cmd *cobra.Command }

func (f flagSet) Str(name string) string {
	v, _ := f.cmd.Flags().GetString(name)
	return v
}
func (f flagSet) Int(name string) int {
	v, _ := f.cmd.Flags().GetInt(name)
	return v
}
func (f flagSet) Int64(name string) int64 {
	v, _ := f.cmd.Flags().GetInt64(name)
	return v
}

func simpleCreate(use, short string, flags []flag, run func(ctx context.Context, ctn *Container, fs flagSet) (any, error)) *cobra.Command {
	c := &cobra.Command{
		Use: use, Short: short,
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			fs := flagSet{cmd: cmd}
			for _, f := range flags {
				if !f.required {
					continue
				}
				switch f.kind {
				case "string":
					if fs.Str(f.name) == "" {
						return fmt.Errorf("--%s 必填", f.name)
					}
				case "int":
					if fs.Int(f.name) == 0 {
						return fmt.Errorf("--%s 必填", f.name)
					}
				case "int64":
					if fs.Int64(f.name) == 0 {
						return fmt.Errorf("--%s 必填", f.name)
					}
				}
			}
			row, err := run(ctx, ctn, fs)
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	for _, f := range flags {
		switch f.kind {
		case "string":
			c.Flags().String(f.name, "", f.desc)
		case "int":
			c.Flags().Int(f.name, 0, f.desc)
		case "int64":
			c.Flags().Int64(f.name, 0, f.desc)
		}
	}
	return c
}

func simpleList(use, short string, run func(ctx context.Context, ctn *Container, bookID int64, limit int) (any, error)) *cobra.Command {
	c := &cobra.Command{
		Use: use, Short: short,
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				limit = 50
			}
			rows, err := run(ctx, ctn, bookID, limit)
			if err != nil {
				return err
			}
			return printJSON(rows)
		}),
	}
	c.Flags().Int64("book", 0, "书 ID")
	c.Flags().Int("limit", 50, "最大行数")
	return c
}

func simpleGet(use, short string, run func(ctx context.Context, ctn *Container, bookID, id int64) (any, error)) *cobra.Command {
	c := &cobra.Command{
		Use: use, Short: short,
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			id, _ := cmd.Flags().GetInt64("id")
			if bookID == 0 || id == 0 {
				return fmt.Errorf("--book 与 --id 必填")
			}
			row, err := run(ctx, ctn, bookID, id)
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	c.Flags().Int64("book", 0, "书 ID")
	c.Flags().Int64("id", 0, "资源 ID")
	return c
}

func simpleUpdate(use, short string, fields []string, run func(ctx context.Context, ctn *Container, bookID, id int64, fs flagSet) (any, error)) *cobra.Command {
	c := &cobra.Command{
		Use: use, Short: short,
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			id, _ := cmd.Flags().GetInt64("id")
			if bookID == 0 || id == 0 {
				return fmt.Errorf("--book 与 --id 必填")
			}
			row, err := run(ctx, ctn, bookID, id, flagSet{cmd: cmd})
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	c.Flags().Int64("book", 0, "书 ID")
	c.Flags().Int64("id", 0, "资源 ID")
	for _, name := range fields {
		c.Flags().String(name, "", "更新该字段(留空则不改)")
	}
	return c
}

func simpleDelete(use, short string, run func(ctx context.Context, ctn *Container, bookID, id int64) error) *cobra.Command {
	c := &cobra.Command{
		Use: use, Short: short,
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			id, _ := cmd.Flags().GetInt64("id")
			if bookID == 0 || id == 0 {
				return fmt.Errorf("--book 与 --id 必填")
			}
			if err := run(ctx, ctn, bookID, id); err != nil {
				return err
			}
			fmt.Println("deleted id=", id)
			return nil
		}),
	}
	c.Flags().Int64("book", 0, "书 ID")
	c.Flags().Int64("id", 0, "资源 ID")
	return c
}
