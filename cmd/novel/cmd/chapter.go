package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"myai-novel-go/internal/domain/chapter"
)

func newChapterCmd() *cobra.Command {
	c := &cobra.Command{Use: "chapter", Short: "管理章节"}

	create := &cobra.Command{
		Use:   "create",
		Short: "创建章节",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			title, _ := cmd.Flags().GetString("title")
			twc, _ := cmd.Flags().GetInt("target-word-count")
			if bookID == 0 || chapterNo == 0 {
				return fmt.Errorf("--book 与 --chapter 必填")
			}
			row, err := ctn.Chapter.Create(ctx, chapter.CreateInput{
				BookID:          bookID,
				ChapterNo:       chapterNo,
				Title:           stringPtr(title),
				TargetWordCount: intPtr(twc),
			})
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	create.Flags().Int64("book", 0, "书 ID")
	create.Flags().Int("chapter", 0, "章节号")
	create.Flags().String("title", "", "章节标题")
	create.Flags().Int("target-word-count", 0, "目标字数")

	list := &cobra.Command{
		Use:   "list",
		Short: "列出章节",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				limit = 50
			}
			rows, err := ctn.Chapter.List(ctx, bookID, limit, "")
			if err != nil {
				return err
			}
			return printJSON(rows)
		}),
	}
	list.Flags().Int64("book", 0, "书 ID")
	list.Flags().Int("limit", 50, "最大行数")

	show := &cobra.Command{
		Use:   "show",
		Short: "查看章节",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			row, err := ctn.Chapter.Get(ctx, bookID, chapterNo)
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	show.Flags().Int64("book", 0, "书 ID")
	show.Flags().Int("chapter", 0, "章节号")

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "把章节阶段内容导出为 markdown",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			stage, _ := cmd.Flags().GetString("stage")
			out, _ := cmd.Flags().GetString("output")
			md, _, err := ctn.Chapter.ExportStage(ctx, bookID, chapterNo, stage)
			if err != nil {
				return err
			}
			if out == "" {
				out = filepath.Join(".", fmt.Sprintf("chapter-%04d-%s.md", chapterNo, stage))
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
				return err
			}
			fmt.Println("chapter.export.ok path=", out)
			return nil
		}),
	}
	exportCmd.Flags().Int64("book", 0, "书 ID")
	exportCmd.Flags().Int("chapter", 0, "章节号")
	exportCmd.Flags().String("stage", "final", "plan|draft|review|final")
	exportCmd.Flags().String("output", "", "输出文件路径(默认 ./chapter-XXXX-stage.md)")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "从 markdown 文件回写阶段",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			stage, _ := cmd.Flags().GetString("stage")
			input, _ := cmd.Flags().GetString("input")
			force, _ := cmd.Flags().GetBool("force")
			if input == "" {
				input = filepath.Join(".", fmt.Sprintf("chapter-%04d-%s.md", chapterNo, stage))
			}
			raw, err := os.ReadFile(input)
			if err != nil {
				return err
			}
			entry, err := ctn.Chapter.ImportStage(ctx, bookID, chapterNo, stage, string(raw), force)
			if err != nil {
				return err
			}
			return printJSON(entry)
		}),
	}
	importCmd.Flags().Int64("book", 0, "书 ID")
	importCmd.Flags().Int("chapter", 0, "章节号")
	importCmd.Flags().String("stage", "draft", "plan|draft|final")
	importCmd.Flags().String("input", "", "输入 markdown 路径")
	importCmd.Flags().Bool("force", false, "final 阶段允许在非 approved 状态下覆盖")

	update := &cobra.Command{
		Use:   "update",
		Short: "更新章节元信息(title/summary/target-word-count)",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			if bookID == 0 || chapterNo == 0 {
				return fmt.Errorf("--book 与 --chapter 必填")
			}
			in := chapter.UpdateInput{}
			if v, _ := cmd.Flags().GetString("title"); v != "" {
				in.Title = &v
			}
			if v, _ := cmd.Flags().GetString("summary"); v != "" {
				in.Summary = &v
			}
			if v, _ := cmd.Flags().GetInt("target-word-count"); v > 0 {
				in.TargetWordCount = &v
			}
			row, err := ctn.Chapter.Update(ctx, bookID, chapterNo, in)
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	update.Flags().Int64("book", 0, "书 ID")
	update.Flags().Int("chapter", 0, "章节号")
	update.Flags().String("title", "", "新标题")
	update.Flags().String("summary", "", "新摘要")
	update.Flags().Int("target-word-count", 0, "新目标字数")

	del := &cobra.Command{
		Use:   "delete",
		Short: "删除章节",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			if bookID == 0 || chapterNo == 0 {
				return fmt.Errorf("--book 与 --chapter 必填")
			}
			if err := ctn.Chapter.Remove(ctx, bookID, chapterNo); err != nil {
				return err
			}
			fmt.Println("chapter.deleted book=", bookID, "chapter=", chapterNo)
			return nil
		}),
	}
	del.Flags().Int64("book", 0, "书 ID")
	del.Flags().Int("chapter", 0, "章节号")

	c.AddCommand(create, list, show, update, del, exportCmd, importCmd)
	return c
}
