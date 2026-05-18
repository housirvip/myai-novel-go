package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"myai-novel-go/internal/domain/book"
)

func newBookCmd() *cobra.Command {
	c := &cobra.Command{Use: "book", Short: "管理书籍"}

	create := &cobra.Command{
		Use:   "create",
		Short: "创建一本书",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				return fmt.Errorf("--title 必填")
			}
			tcc, _ := cmd.Flags().GetInt("target-chapter-count")
			summary, _ := cmd.Flags().GetString("summary")
			row, err := ctn.Book.Create(ctx, book.CreateInput{
				Title:              title,
				Summary:            stringPtr(summary),
				TargetChapterCount: intPtr(tcc),
			})
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	create.Flags().String("title", "", "书名")
	create.Flags().String("summary", "", "简介(可选)")
	create.Flags().Int("target-chapter-count", 0, "目标章节数(可选)")

	list := &cobra.Command{
		Use:   "list",
		Short: "列出全部书籍",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				limit = 50
			}
			rows, err := ctn.Book.List(ctx, limit)
			if err != nil {
				return err
			}
			return printJSON(rows)
		}),
	}
	list.Flags().Int("limit", 50, "最大返回行数")

	show := &cobra.Command{
		Use:   "show",
		Short: "查看一本书",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetInt64("id")
			if id == 0 {
				return fmt.Errorf("--id 必填")
			}
			row, err := ctn.Book.Get(ctx, id)
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	show.Flags().Int64("id", 0, "书籍 ID")

	update := &cobra.Command{
		Use:   "update",
		Short: "更新一本书",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetInt64("id")
			if id == 0 {
				return fmt.Errorf("--id 必填")
			}
			title, _ := cmd.Flags().GetString("title")
			summary, _ := cmd.Flags().GetString("summary")
			tcc, _ := cmd.Flags().GetInt("target-chapter-count")
			in := book.UpdateInput{}
			if title != "" {
				in.Title = &title
			}
			if summary != "" {
				in.Summary = &summary
			}
			if tcc > 0 {
				in.TargetChapterCount = &tcc
			}
			row, err := ctn.Book.Update(ctx, id, in)
			if err != nil {
				return err
			}
			return printJSON(row)
		}),
	}
	update.Flags().Int64("id", 0, "书籍 ID")
	update.Flags().String("title", "", "新书名")
	update.Flags().String("summary", "", "新简介")
	update.Flags().Int("target-chapter-count", 0, "新目标章节数")

	del := &cobra.Command{
		Use:   "delete",
		Short: "删除一本书",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetInt64("id")
			if id == 0 {
				return fmt.Errorf("--id 必填")
			}
			if err := ctn.Book.Remove(ctx, id); err != nil {
				return err
			}
			fmt.Println("book.deleted id=", id)
			return nil
		}),
	}
	del.Flags().Int64("id", 0, "书籍 ID")

	c.AddCommand(create, list, show, update, del)
	return c
}
