package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/workflows"
)

func newPlanCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "plan",
		Short: "运行 plan 阶段工作流(同步)",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			provider, _ := cmd.Flags().GetString("provider")
			model, _ := cmd.Flags().GetString("model")
			authorIntent, _ := cmd.Flags().GetString("author-intent")
			targetWords, _ := cmd.Flags().GetInt("target-words")
			if bookID == 0 || chapterNo == 0 {
				return fmt.Errorf("--book 与 --chapter 必填")
			}
			out, err := ctn.PlanWF.Run(ctx, workflows.PlanInput{
				BookID: bookID, ChapterNo: chapterNo,
				Provider: provider, Model: model,
				AuthorIntent: authorIntent, TargetWords: targetWords,
				ManualEntityRefs: planning.EmptyManualRefs(),
			}, cliNotifier(ctn.Logger, "plan"))
			if err != nil {
				return err
			}
			return printJSON(out)
		}),
	}
	addCommonWFFlags(c)
	c.Flags().String("author-intent", "", "作者意图(可选,留空让模型自动生成草案)")
	c.Flags().Int("target-words", 0, "目标字数(可选)")
	return c
}

func newDraftCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "draft",
		Short: "运行 draft 阶段工作流(同步)",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			provider, _ := cmd.Flags().GetString("provider")
			model, _ := cmd.Flags().GetString("model")
			targetWords, _ := cmd.Flags().GetInt("target-words")
			if bookID == 0 || chapterNo == 0 {
				return fmt.Errorf("--book 与 --chapter 必填")
			}
			out, err := ctn.DraftWF.Run(ctx, workflows.DraftInput{
				BookID: bookID, ChapterNo: chapterNo,
				Provider: provider, Model: model, TargetWords: targetWords,
			}, cliNotifier(ctn.Logger, "draft"))
			if err != nil {
				return err
			}
			return printJSON(out)
		}),
	}
	addCommonWFFlags(c)
	c.Flags().Int("target-words", 0, "目标字数(可选)")
	return c
}

func newReviewCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "review",
		Short: "运行 review 阶段工作流(同步)",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			provider, _ := cmd.Flags().GetString("provider")
			model, _ := cmd.Flags().GetString("model")
			out, err := ctn.ReviewWF.Run(ctx, workflows.ReviewInput{
				BookID: bookID, ChapterNo: chapterNo,
				Provider: provider, Model: model,
			}, cliNotifier(ctn.Logger, "review"))
			if err != nil {
				return err
			}
			return printJSON(out)
		}),
	}
	addCommonWFFlags(c)
	return c
}

func newRepairCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "repair",
		Short: "运行 repair 阶段工作流(同步)",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			provider, _ := cmd.Flags().GetString("provider")
			model, _ := cmd.Flags().GetString("model")
			out, err := ctn.RepairWF.Run(ctx, workflows.RepairInput{
				BookID: bookID, ChapterNo: chapterNo,
				Provider: provider, Model: model,
			}, cliNotifier(ctn.Logger, "repair"))
			if err != nil {
				return err
			}
			return printJSON(out)
		}),
	}
	addCommonWFFlags(c)
	return c
}

func newApproveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "approve",
		Short: "运行 approve 阶段工作流(同步)",
		RunE: withContainer(func(ctx context.Context, ctn *Container, cmd *cobra.Command, _ []string) error {
			bookID, _ := cmd.Flags().GetInt64("book")
			chapterNo, _ := cmd.Flags().GetInt("chapter")
			provider, _ := cmd.Flags().GetString("provider")
			model, _ := cmd.Flags().GetString("model")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			out, err := ctn.ApproveWF.Run(ctx, workflows.ApproveInput{
				BookID: bookID, ChapterNo: chapterNo,
				Provider: provider, Model: model, DryRun: dryRun,
			}, cliNotifier(ctn.Logger, "approve"))
			if err != nil {
				return err
			}
			return printJSON(out)
		}),
	}
	addCommonWFFlags(c)
	c.Flags().Bool("dry-run", false, "只跑两次 LLM 不写库,返回 final + diff 预览")
	return c
}

func addCommonWFFlags(c *cobra.Command) {
	c.Flags().Int64("book", 0, "书 ID")
	c.Flags().Int("chapter", 0, "章节号")
	c.Flags().String("provider", "", "覆盖默认 provider(可选)")
	c.Flags().String("model", "", "覆盖默认 model(可选)")
}

// cliNotifier 把 workflow 的 stage 进度回调打到日志,
// CLI 用户也能从 stderr 看到 stage=plan_intent.extract progress=30 这样的关键里程碑。
func cliNotifier(logger *zap.Logger, wf string) workflows.StageNotifier {
	return func(stage string, progress *int) error {
		fields := []zap.Field{zap.String("workflow", wf), zap.String("stage", stage)}
		if progress != nil {
			fields = append(fields, zap.Int("progress", *progress))
		}
		logger.Info("workflow.progress", fields...)
		return nil
	}
}
