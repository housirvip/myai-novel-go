package workflows

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/llm"
	"myai-novel-go/internal/llmfactory"
)

type ReviewInput struct {
	BookID    int64  `json:"bookId" binding:"required"`
	ChapterNo int    `json:"chapterNo" binding:"required"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

type ReviewOutput struct {
	ChapterID  int64  `json:"chapterId"`
	ReviewID   int64  `json:"reviewId"`
	DraftID    int64  `json:"draftId"`
	RawResult  string `json:"rawResult"`
	Summary    string `json:"summary"`
}

type ReviewWorkflow struct {
	db   *gorm.DB
	cfg  *config.Config
	llmF *llmfactory.Factory
}

func NewReviewWorkflow(db *gorm.DB, cfg *config.Config, llmF *llmfactory.Factory) *ReviewWorkflow {
	return &ReviewWorkflow{db: db, cfg: cfg, llmF: llmF}
}

type reviewParsed struct {
	Summary           string   `json:"summary"`
	Issues            []string `json:"issues"`
	Risks             []string `json:"risks"`
	ContinuityChecks  []string `json:"continuity_checks"`
	RepairSuggestions []string `json:"repair_suggestions"`
}

func (w *ReviewWorkflow) Run(ctx context.Context, in ReviewInput, notify StageNotifier) (*ReviewOutput, error) {
	llmCli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageLoadingChapter, 10)
	chapter, err := LoadChapter(ctx, w.db, in.BookID, in.ChapterNo)
	if err != nil {
		return nil, err
	}
	if chapter.CurrentDraftID == nil || chapter.CurrentPlanID == nil {
		return nil, shared.BadRequest("chapter needs current plan and draft before review")
	}

	var plan models.ChapterPlan
	var draft models.ChapterDraft
	if err := w.db.WithContext(ctx).First(&plan, *chapter.CurrentPlanID).Error; err != nil {
		return nil, err
	}
	if err := w.db.WithContext(ctx).First(&draft, *chapter.CurrentDraftID).Error; err != nil {
		return nil, err
	}

	retrievedCtx, err := LoadRetrievedContext(&plan)
	if err != nil {
		return nil, err
	}

	Notify(notify, shared.WorkflowStageGeneratingReview, 70)
	res, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model:          llm.ResolveModel(w.cfg, in.Model, llm.TierMid),
		ResponseFormat: llm.ResponseFormatJSON,
		Messages: planning.BuildReviewPrompt(planning.ReviewPromptInput{
			PlanContent: plan.Content, DraftContent: draft.Content, RetrievedContext: retrievedCtx,
		}),
	})
	if err != nil {
		return nil, err
	}

	parsed := reviewParsed{}
	_ = json.Unmarshal([]byte(stripFence(res.Content)), &parsed)

	Notify(notify, shared.WorkflowStageSavingArtifacts, 95)
	out := &ReviewOutput{ChapterID: chapter.ID, DraftID: draft.ID, RawResult: res.Content, Summary: parsed.Summary}
	err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fresh, err := AssertPointersUnchanged(tx, in.BookID, in.ChapterNo, TakeSnapshot(chapter))
		if err != nil {
			return err
		}
		ver, err := nextVersion(tx, "chapter_reviews", fresh.ID)
		if err != nil {
			return err
		}
		now := shared.NowISO()
		issuesJSON := mustMarshal(parsed.Issues)
		risksJSON := mustMarshal(parsed.Risks)
		ccJSON := mustMarshal(parsed.ContinuityChecks)
		rsJSON := mustMarshal(parsed.RepairSuggestions)
		summaryStr := parsed.Summary
		modelStr := res.Model
		providerStr := string(res.Provider)
		row := models.ChapterReview{
			BookID: in.BookID, ChapterID: fresh.ID, ChapterNo: in.ChapterNo, DraftID: draft.ID, VersionNo: ver,
			Status: "active", Summary: &summaryStr, Issues: &issuesJSON, Risks: &risksJSON,
			ContinuityChecks: &ccJSON, RepairSuggestions: &rsJSON,
			RawResult: res.Content, Model: &modelStr, Provider: &providerStr,
			SourceType: shared.ChapterSourceAIGenerated,
			CreatedAt:  now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		out.ReviewID = row.ID
		return tx.Model(&models.Chapter{}).Where("id = ?", fresh.ID).Updates(map[string]any{
			"current_review_id": row.ID,
			"status":            shared.ChapterStatusReviewed,
			"updated_at":        now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func stripFence(s string) string {
	out := s
	for _, p := range []string{"```json", "```"} {
		if len(out) >= len(p) && out[:len(p)] == p {
			out = out[len(p):]
		}
	}
	if l := len(out); l >= 3 && out[l-3:] == "```" {
		out = out[:l-3]
	}
	return out
}
