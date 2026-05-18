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

const (
	targetWordToleranceRatio = 0.1
	minLengthRepairDelta     = 300
)

type DraftInput struct {
	BookID      int64  `json:"bookId" binding:"required"`
	ChapterNo   int    `json:"chapterNo" binding:"required"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	TargetWords int    `json:"targetWords"`
}

type DraftOutput struct {
	ChapterID     int64  `json:"chapterId"`
	DraftID       int64  `json:"draftId"`
	BasedOnPlanID int64  `json:"basedOnPlanId"`
	WordCount     int    `json:"wordCount"`
	Content       string `json:"content"`
}

type DraftWorkflow struct {
	db   *gorm.DB
	cfg  *config.Config
	llmF *llmfactory.Factory
}

func NewDraftWorkflow(db *gorm.DB, cfg *config.Config, llmF *llmfactory.Factory) *DraftWorkflow {
	return &DraftWorkflow{db: db, cfg: cfg, llmF: llmF}
}

func (w *DraftWorkflow) Run(ctx context.Context, in DraftInput, notify StageNotifier) (*DraftOutput, error) {
	llmCli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageLoadingChapter, 5)
	chapter, err := LoadChapter(ctx, w.db, in.BookID, in.ChapterNo)
	if err != nil {
		return nil, err
	}
	if chapter.CurrentPlanID == nil {
		return nil, shared.BadRequest("chapter does not have a current plan")
	}

	Notify(notify, shared.WorkflowStageLoadingPlanContext, 20)
	var plan models.ChapterPlan
	if err := w.db.WithContext(ctx).First(&plan, *chapter.CurrentPlanID).Error; err != nil {
		return nil, err
	}
	if plan.ChapterID != chapter.ID || plan.BookID != in.BookID {
		return nil, shared.BadRequest("current plan pointer is invalid")
	}

	retrievedCtx, err := LoadRetrievedContext(&plan)
	if err != nil {
		return nil, err
	}
	intent := ReadIntentConstraints(&plan)

	target := in.TargetWords
	if target == 0 {
		if chapter.TargetWordCount != nil {
			target = *chapter.TargetWordCount
		} else {
			target = defaultTargetWords
		}
	}

	Notify(notify, shared.WorkflowStageGeneratingDraft, 60)
	model := llm.ResolveModel(w.cfg, in.Model, llm.TierHigh)
	res, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model: model,
		Messages: planning.BuildDraftPrompt(planning.DraftPromptInput{
			PlanContent: plan.Content, IntentConstraints: intent,
			RetrievedContext: retrievedCtx, TargetWords: target,
		}),
	})
	if err != nil {
		return nil, err
	}

	maxRounds := w.cfg.DraftLengthRepairMaxRounds
	if maxRounds < 0 {
		maxRounds = 0
	}
	wc := shared.EstimateWordCount(res.Content)
	for round := 1; round <= maxRounds; round++ {
		if !shouldRepairLength(wc, target) {
			break
		}
		Notify(notify, shared.WorkflowStageRepairingLength, repairProgress(round, maxRounds))
		var msgs []llm.Message
		if shouldAggressivelyCompress(wc, target) {
			msgs = planning.BuildDraftAggressiveCompressionPrompt(struct {
				DraftContent     string
				PlanContent      string
				TargetWords      int
				CurrentWordCount int
			}{res.Content, plan.Content, target, wc})
		} else {
			msgs = planning.BuildDraftLengthRepairPrompt(struct {
				DraftContent     string
				PlanContent      string
				TargetWords      int
				CurrentWordCount int
			}{res.Content, plan.Content, target, wc})
		}
		res2, err := llmCli.Generate(ctx, llm.GenerateParams{Model: model, Messages: msgs})
		if err != nil {
			return nil, err
		}
		res = res2
		wc = shared.EstimateWordCount(res.Content)
	}

	Notify(notify, shared.WorkflowStageSavingArtifacts, 95)
	out := &DraftOutput{ChapterID: chapter.ID, BasedOnPlanID: plan.ID, WordCount: wc, Content: res.Content}
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
		summary := chapter.Summary
		if summary == nil && plan.AuthorIntent != nil {
			summary = plan.AuthorIntent
		}
		sourceType := shared.ChapterSourceAIGenerated
		if chapter.CurrentReviewID != nil {
			sourceType = shared.ChapterSourceRepaired
		}
		row := models.ChapterDraft{
			BookID: in.BookID, ChapterID: fresh.ID, ChapterNo: in.ChapterNo, VersionNo: ver,
			BasedOnPlanID: &plan.ID, BasedOnDraftID: chapter.CurrentDraftID, BasedOnReviewID: chapter.CurrentReviewID,
			Status: "active", Content: res.Content, Summary: summary, WordCount: &wc,
			Model: &modelStr, Provider: &providerStr, SourceType: sourceType,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		out.DraftID = row.ID
		updates := map[string]any{
			"current_draft_id": row.ID,
			"status":           shared.ChapterStatusDrafted,
			"word_count":       wc,
			"updated_at":       now,
		}
		if in.TargetWords > 0 {
			updates["target_word_count"] = in.TargetWords
		}
		return tx.Model(&models.Chapter{}).Where("id = ?", fresh.ID).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func shouldRepairLength(wc, target int) bool {
	allowed := int(float64(target) * targetWordToleranceRatio)
	if allowed < minLengthRepairDelta {
		allowed = minLengthRepairDelta
	}
	delta := wc - target
	if delta < 0 {
		delta = -delta
	}
	return delta > allowed
}

func shouldAggressivelyCompress(wc, target int) bool {
	threshold := int(float64(target) * 1.2)
	if threshold < target+minLengthRepairDelta {
		threshold = target + minLengthRepairDelta
	}
	return wc > threshold
}

func repairProgress(round, max int) int {
	if max <= 1 {
		return 85
	}
	p := 85 + ((round-1)*8)/(max-1)
	if p > 93 {
		p = 93
	}
	return p
}
