package workflows

import (
	"context"

	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/llm"
	"myai-novel-go/internal/llmfactory"
)

type RepairInput struct {
	BookID    int64  `json:"bookId" binding:"required"`
	ChapterNo int    `json:"chapterNo" binding:"required"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

type RepairOutput struct {
	ChapterID      int64  `json:"chapterId"`
	DraftID        int64  `json:"draftId"`
	BasedOnDraftID int64  `json:"basedOnDraftId"`
	BasedOnReviewID int64 `json:"basedOnReviewId"`
	WordCount      int    `json:"wordCount"`
	Content        string `json:"content"`
}

type RepairWorkflow struct {
	db   *gorm.DB
	cfg  *config.Config
	llmF *llmfactory.Factory
}

func NewRepairWorkflow(db *gorm.DB, cfg *config.Config, llmF *llmfactory.Factory) *RepairWorkflow {
	return &RepairWorkflow{db: db, cfg: cfg, llmF: llmF}
}

func (w *RepairWorkflow) Run(ctx context.Context, in RepairInput, notify StageNotifier) (*RepairOutput, error) {
	llmCli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageLoadingChapter, 10)
	chapter, err := LoadChapter(ctx, w.db, in.BookID, in.ChapterNo)
	if err != nil {
		return nil, err
	}
	if chapter.CurrentPlanID == nil || chapter.CurrentDraftID == nil || chapter.CurrentReviewID == nil {
		return nil, shared.BadRequest("chapter needs current plan, draft and review before repair")
	}

	var plan models.ChapterPlan
	var draft models.ChapterDraft
	var review models.ChapterReview
	if err := w.db.WithContext(ctx).First(&plan, *chapter.CurrentPlanID).Error; err != nil {
		return nil, err
	}
	if err := w.db.WithContext(ctx).First(&draft, *chapter.CurrentDraftID).Error; err != nil {
		return nil, err
	}
	if err := w.db.WithContext(ctx).First(&review, *chapter.CurrentReviewID).Error; err != nil {
		return nil, err
	}

	retrievedCtx, err := LoadRetrievedContext(&plan)
	if err != nil {
		return nil, err
	}
	intent := ReadIntentConstraints(&plan)

	Notify(notify, shared.WorkflowStageGeneratingRepair, 80)
	res, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model: llm.ResolveModel(w.cfg, in.Model, llm.TierHigh),
		Messages: planning.BuildRepairPrompt(planning.RepairPromptInput{
			PlanContent: plan.Content, DraftContent: draft.Content, ReviewContent: review.RawResult,
			IntentConstraints: intent, RetrievedContext: retrievedCtx,
		}),
	})
	if err != nil {
		return nil, err
	}

	wc := shared.EstimateWordCount(res.Content)
	Notify(notify, shared.WorkflowStageSavingArtifacts, 95)
	out := &RepairOutput{ChapterID: chapter.ID, BasedOnDraftID: draft.ID, BasedOnReviewID: review.ID, WordCount: wc, Content: res.Content}
	err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fresh, err := AssertPointersUnchanged(tx, in.BookID, in.ChapterNo, TakeSnapshot(chapter))
		if err != nil {
			return err
		}
		ver, err := nextVersion(tx, "chapter_drafts", fresh.ID)
		if err != nil {
			return err
		}
		now := shared.NowISO()
		modelStr := res.Model
		providerStr := string(res.Provider)
		row := models.ChapterDraft{
			BookID: in.BookID, ChapterID: fresh.ID, ChapterNo: in.ChapterNo, VersionNo: ver,
			BasedOnPlanID: draft.BasedOnPlanID, BasedOnDraftID: &draft.ID, BasedOnReviewID: &review.ID,
			Status: "active", Content: res.Content, Summary: draft.Summary, WordCount: &wc,
			Model: &modelStr, Provider: &providerStr, SourceType: shared.ChapterSourceRepaired,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		out.DraftID = row.ID
		return tx.Model(&models.Chapter{}).Where("id = ?", fresh.ID).Updates(map[string]any{
			"current_draft_id": row.ID,
			"word_count":       wc,
			"status":           shared.ChapterStatusRepaired,
			"updated_at":       now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
