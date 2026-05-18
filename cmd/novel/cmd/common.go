package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db"
	"myai-novel-go/internal/domain/book"
	"myai-novel-go/internal/domain/chapter"
	"myai-novel-go/internal/domain/character"
	"myai-novel-go/internal/domain/faction"
	"myai-novel-go/internal/domain/item"
	"myai-novel-go/internal/domain/outline"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/relation"
	"myai-novel-go/internal/domain/story_hook"
	"myai-novel-go/internal/domain/workflows"
	"myai-novel-go/internal/domain/world_setting"
	"myai-novel-go/internal/llmfactory"
	"myai-novel-go/internal/logger"
)

// Container 把所有 service 与 workflow 凑齐,
// 让每个 cobra 子命令都能直接调用 domain service,而不必重新初始化。
type Container struct {
	Cfg    *config.Config
	Logger *zap.Logger
	DB     *gorm.DB

	Book        *book.Service
	Chapter     *chapter.Service
	Outline     *outline.Service
	World       *world_setting.Service
	Character   *character.Service
	Faction     *faction.Service
	Relation    *relation.Service
	Item        *item.Service
	Hook        *story_hook.Service
	LLMFactory  *llmfactory.Factory
	Retrieval   *planning.RetrievalService
	PlanWF      *workflows.PlanWorkflow
	DraftWF     *workflows.DraftWorkflow
	ReviewWF    *workflows.ReviewWorkflow
	RepairWF    *workflows.RepairWorkflow
	ApproveWF   *workflows.ApproveWorkflow
	StageSumWF  *workflows.StageSummaryWorkflow
}

func openContainer() (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	zlog, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return nil, err
	}
	gdb, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(gdb); err != nil {
		return nil, err
	}
	llmF := llmfactory.New(cfg)
	embClient := llmF.NewEmbedding()
	retrieval := planning.NewRetrievalServiceWithEmbedding(gdb, cfg, embClient)
	return &Container{
		Cfg:        cfg,
		Logger:     zlog,
		DB:         gdb,
		Book:       book.NewService(gdb),
		Chapter:    chapter.NewService(gdb),
		Outline:    outline.NewService(gdb),
		World:      world_setting.NewService(gdb),
		Character:  character.NewService(gdb),
		Faction:    faction.NewService(gdb),
		Relation:   relation.NewService(gdb),
		Item:       item.NewService(gdb),
		Hook:       story_hook.NewService(gdb),
		LLMFactory: llmF,
		Retrieval:  retrieval,
		PlanWF:     workflows.NewPlanWorkflow(gdb, cfg, llmF, retrieval),
		DraftWF:    workflows.NewDraftWorkflow(gdb, cfg, llmF),
		ReviewWF:   workflows.NewReviewWorkflow(gdb, cfg, llmF),
		RepairWF:   workflows.NewRepairWorkflow(gdb, cfg, llmF),
		ApproveWF:  workflows.NewApproveWorkflow(gdb, cfg, llmF),
		StageSumWF: workflows.NewStageSummaryWorkflow(gdb, cfg, llmF),
	}, nil
}

// withContainer 是给 RunE 用的包装:打开容器、执行业务、统一处理错误。
func withContainer(fn func(ctx context.Context, c *Container, cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := openContainer()
		if err != nil {
			return err
		}
		defer c.Logger.Sync()
		return fn(cmd.Context(), c, cmd, args)
	}
}

// printJSON pretty-print 任意结构体到 stdout。
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// NewRoot 创建顶层 root command。
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "novel",
		Short:         "AI 小说创作命令行工具",
		Long:          "管理设定、章节与写作工作流的小说 CLI(Go 重写版,与 HTTP 服务端共享 service 层)",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       "0.1.0",
	}
	root.SetContext(context.Background())
	root.AddCommand(
		newDBCmd(),
		newBookCmd(),
		newChapterCmd(),
		newOutlineCmd(),
		newWorldCmd(),
		newCharacterCmd(),
		newFactionCmd(),
		newRelationCmd(),
		newItemCmd(),
		newHookCmd(),
		newPlanCmd(),
		newDraftCmd(),
		newReviewCmd(),
		newRepairCmd(),
		newApproveCmd(),
	)
	return root
}

// stringPtr 把空字符串视为 nil(用于 *string 选填字段)。
func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// intPtr 把零值视为 nil(用于 *int 选填字段)。
func intPtr(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func mustParseInt(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	return n
}
