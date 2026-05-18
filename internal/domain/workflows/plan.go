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

const defaultTargetWords = 3000

type PlanInput struct {
	BookID           int64                    `json:"bookId" binding:"required"`
	ChapterNo        int                      `json:"chapterNo" binding:"required"`
	Provider         string                   `json:"provider"`
	Model            string                   `json:"model"`
	AuthorIntent     string                   `json:"authorIntent"`
	TargetWords      int                      `json:"targetWords"`
	ManualEntityRefs planning.ManualEntityRefs `json:"manualEntityRefs"`
}

type PlanOutput struct {
	ChapterID       int64                       `json:"chapterId"`
	PlanID          int64                       `json:"planId"`
	AuthorIntent    string                      `json:"authorIntent"`
	IntentSource    string                      `json:"intentSource"`
	IntentKeywords  []string                    `json:"intentKeywords"`
	IntentSummary   string                      `json:"intentSummary"`
	MustInclude     []string                    `json:"mustInclude"`
	MustAvoid       []string                    `json:"mustAvoid"`
	RetrievedContext *planning.RetrievedContext `json:"retrievedContext"`
	Content         string                      `json:"content"`
}

type PlanWorkflow struct {
	db        *gorm.DB
	cfg       *config.Config
	llmF      *llmfactory.Factory
	retrieval *planning.RetrievalService
}

func NewPlanWorkflow(db *gorm.DB, cfg *config.Config, llmF *llmfactory.Factory, r *planning.RetrievalService) *PlanWorkflow {
	return &PlanWorkflow{db: db, cfg: cfg, llmF: llmF, retrieval: r}
}

// AuthorIntentInput 是 /api/workflows/author-intent 入参。
type AuthorIntentInput struct {
	BookID           int64                     `json:"bookId" binding:"required"`
	ChapterNo        int                       `json:"chapterNo" binding:"required"`
	Provider         string                    `json:"provider"`
	Model            string                    `json:"model"`
	ManualEntityRefs planning.ManualEntityRefs `json:"manualEntityRefs"`
}

type AuthorIntentOutput struct {
	BookID       int64  `json:"bookId"`
	ChapterNo    int    `json:"chapterNo"`
	AuthorIntent string `json:"authorIntent"`
	Source       string `json:"source"`
}

// GenerateAuthorIntent 跑"初始检索 + 意图草案生成",
// 不写库、不进入 plan 流水线,纯建议性输出。供 /api/workflows/author-intent 用。
func (w *PlanWorkflow) GenerateAuthorIntent(ctx context.Context, in AuthorIntentInput, notify StageNotifier) (*AuthorIntentOutput, error) {
	llmCli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageRetrievingInitial, 15)
	initialCtx, err := w.retrieval.Retrieve(ctx, planning.RetrieveParams{
		BookID: in.BookID, ChapterNo: in.ChapterNo, ManualRefs: in.ManualEntityRefs,
	})
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageGeneratingAuthorIntent, 60)
	res, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model: llm.ResolveModel(w.cfg, in.Model, llm.TierMid),
		Messages: planning.BuildIntentGenerationPrompt(struct {
			BookTitle         string
			ChapterNo         int
			OutlinesText      string
			RecentChapterText string
			ManualFocusText   string
			RiskReminderText  string
		}{
			BookTitle: initialCtx.Book.Title, ChapterNo: in.ChapterNo,
			OutlinesText:      formatLightOutlines(initialCtx),
			RecentChapterText: formatLightRecent(initialCtx),
			ManualFocusText:   "无",
			RiskReminderText:  "无",
		}),
	})
	if err != nil {
		return nil, err
	}
	Notify(notify, shared.WorkflowStageGeneratingAuthorIntent, 100)
	return &AuthorIntentOutput{
		BookID:       in.BookID,
		ChapterNo:    in.ChapterNo,
		AuthorIntent: res.Content,
		Source:       shared.PlanIntentSourceAIGenerated,
	}, nil
}

func (w *PlanWorkflow) Run(ctx context.Context, in PlanInput, notify StageNotifier) (*PlanOutput, error) {
	if in.TargetWords == 0 {
		in.TargetWords = defaultTargetWords
	}

	llmCli, err := w.llmF.Create(llm.ProviderName(in.Provider))
	if err != nil {
		return nil, err
	}

	Notify(notify, shared.WorkflowStageLoadingChapter, 5)
	chapter, err := LoadChapter(ctx, w.db, in.BookID, in.ChapterNo)
	if err != nil {
		return nil, err
	}

	Notify(notify, shared.WorkflowStageRetrievingInitial, 15)
	initialCtx, err := w.retrieval.Retrieve(ctx, planning.RetrieveParams{
		BookID: in.BookID, ChapterNo: in.ChapterNo, ManualRefs: in.ManualEntityRefs,
	})
	if err != nil {
		return nil, err
	}

	authorIntent := in.AuthorIntent
	intentSource := shared.PlanIntentSourceUserInput
	if authorIntent == "" {
		Notify(notify, shared.WorkflowStageGeneratingAuthorIntent, 30)
		// 注:为了让 mock 命中"作者意图草案"分支,这里 prompt 必须包含该字符串
		intentRes, err := llmCli.Generate(ctx, llm.GenerateParams{
			Model: llm.ResolveModel(w.cfg, in.Model, llm.TierMid),
			Messages: planning.BuildIntentGenerationPrompt(struct {
				BookTitle         string
				ChapterNo         int
				OutlinesText      string
				RecentChapterText string
				ManualFocusText   string
				RiskReminderText  string
			}{
				BookTitle: initialCtx.Book.Title, ChapterNo: in.ChapterNo,
				OutlinesText:      formatLightOutlines(initialCtx),
				RecentChapterText: formatLightRecent(initialCtx),
				ManualFocusText:   "无",
				RiskReminderText:  "无",
			}),
		})
		if err != nil {
			return nil, err
		}
		// 强制 prompt 中带"作者意图草案"关键字以匹配 mock provider
		_ = intentRes
		authorIntent = intentRes.Content
		intentSource = shared.PlanIntentSourceAIGenerated
	}

	Notify(notify, shared.WorkflowStageExtractingKeywords, 45)
	kwRes, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model:          llm.ResolveModel(w.cfg, in.Model, llm.TierLow),
		Messages:       planning.BuildKeywordExtractionPrompt(authorIntent),
		ResponseFormat: llm.ResponseFormatJSON,
	})
	if err != nil {
		return nil, err
	}
	extracted := planning.NormalizeExtractedIntent(kwRes.Content)
	keywords, queryText := planning.BuildRetrievalQuery(extracted)

	Notify(notify, shared.WorkflowStageRetrievingFinal, 60)
	finalCtx, err := w.retrieval.Retrieve(ctx, planning.RetrieveParams{
		BookID: in.BookID, ChapterNo: in.ChapterNo,
		Keywords: keywords, QueryText: queryText,
		ManualRefs: in.ManualEntityRefs,
	})
	if err != nil {
		return nil, err
	}

	Notify(notify, shared.WorkflowStageGeneratingPlan, 80)
	intentConstraints := planning.IntentConstraints{
		IntentSummary: extracted.IntentSummary,
		MustInclude:   extracted.MustInclude,
		MustAvoid:     extracted.MustAvoid,
	}
	planRes, err := llmCli.Generate(ctx, llm.GenerateParams{
		Model: llm.ResolveModel(w.cfg, in.Model, llm.TierMid),
		Messages: planning.BuildPlanPrompt(planning.PlanPromptInput{
			BookTitle:         initialCtx.Book.Title,
			ChapterNo:         in.ChapterNo,
			AuthorIntent:      authorIntent,
			IntentConstraints: intentConstraints,
			RetrievedContext:  finalCtx,
			TargetWords:       in.TargetWords,
		}),
	})
	if err != nil {
		return nil, err
	}

	Notify(notify, shared.WorkflowStageSavingArtifacts, 95)
	out := &PlanOutput{
		ChapterID:        chapter.ID,
		AuthorIntent:     authorIntent,
		IntentSource:     intentSource,
		IntentKeywords:   extracted.Keywords,
		IntentSummary:    extracted.IntentSummary,
		MustInclude:      extracted.MustInclude,
		MustAvoid:        extracted.MustAvoid,
		RetrievedContext: finalCtx,
		Content:          planRes.Content,
	}

	err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fresh, err := AssertPointersUnchanged(tx, in.BookID, in.ChapterNo, TakeSnapshot(chapter))
		if err != nil {
			return err
		}
		ver, err := nextVersion(tx, "chapter_plans", fresh.ID)
		if err != nil {
			return err
		}
		now := shared.NowISO()
		ctxJSON, _ := json.Marshal(finalCtx)
		ctxStr := string(ctxJSON)
		manualJSON, _ := json.Marshal(in.ManualEntityRefs)
		manualStr := string(manualJSON)
		intentSummaryStr := extracted.IntentSummary
		intentKwJSON, _ := json.Marshal(extracted.Keywords)
		intentMIJSON, _ := json.Marshal(extracted.MustInclude)
		intentMAJSON, _ := json.Marshal(extracted.MustAvoid)
		intentKwStr := string(intentKwJSON)
		intentMIStr := string(intentMIJSON)
		intentMAStr := string(intentMAJSON)
		modelStr := planRes.Model
		providerStr := string(planRes.Provider)

		row := models.ChapterPlan{
			BookID: in.BookID, ChapterID: fresh.ID, ChapterNo: in.ChapterNo, VersionNo: ver,
			Status: "active", AuthorIntent: &authorIntent, IntentSource: intentSource,
			IntentSummary: &intentSummaryStr, IntentKeywords: &intentKwStr,
			IntentMustInclude: &intentMIStr, IntentMustAvoid: &intentMAStr,
			ManualEntityRefs: &manualStr, RetrievedContext: &ctxStr,
			Content: planRes.Content, Model: &modelStr, Provider: &providerStr,
			SourceType: shared.ChapterSourceAIGenerated,
			CreatedAt:  now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		out.PlanID = row.ID
		updates := map[string]any{
			"current_plan_id": row.ID,
			"status":          shared.ChapterStatusPlanned,
			"updated_at":      now,
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

func formatLightOutlines(ctx *planning.RetrievedContext) string {
	if len(ctx.Outlines) == 0 {
		return "暂无命中大纲。"
	}
	out := ""
	for _, o := range ctx.Outlines {
		out += "- " + o.Title + "\n"
	}
	return out
}

func formatLightRecent(ctx *planning.RetrievedContext) string {
	if len(ctx.RecentChapters) == 0 {
		return "暂无前文章节。"
	}
	out := ""
	for _, r := range ctx.RecentChapters {
		t := ""
		if r.Title != nil {
			t = *r.Title
		}
		out += "- 第" + intToStr(r.ChapterNo) + "章 " + t + "\n"
	}
	return out
}

func intToStr(n int) string {
	// 轻量避免引入 strconv 单独打印,保留可读性
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
