package workflows

import (
	"context"

	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/llm"
	"myai-novel-go/internal/llmfactory"
)

type StageSummaryInput struct {
	BookID    int64  `json:"bookId" binding:"required"`
	ChapterNo int    `json:"chapterNo" binding:"required"`
	Stage     string `json:"stage" binding:"required,oneof=plan draft final"`
	Content   string `json:"content" binding:"required,min=1"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

type StageSummaryOutput struct {
	Summary string `json:"summary"`
}

type StageSummaryWorkflow struct {
	db   *gorm.DB
	cfg  *config.Config
	llmF *llmfactory.Factory
}

func NewStageSummaryWorkflow(db *gorm.DB, cfg *config.Config, llmF *llmfactory.Factory) *StageSummaryWorkflow {
	return &StageSummaryWorkflow{db: db, cfg: cfg, llmF: llmF}
}

func (w *StageSummaryWorkflow) Run(ctx context.Context, in StageSummaryInput) (*StageSummaryOutput, error) {
	cli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}
	res, err := cli.Generate(ctx, llm.GenerateParams{
		Model:    llm.ResolveModel(w.cfg, in.Model, llm.TierLow),
		Messages: planning.BuildStageSummaryPrompt(in.Stage, in.Content),
	})
	if err != nil {
		return nil, err
	}
	return &StageSummaryOutput{Summary: res.Content}, nil
}
