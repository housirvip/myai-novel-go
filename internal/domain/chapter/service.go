package chapter

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateInput struct {
	BookID                int64   `json:"-"`
	ChapterNo             int     `json:"chapterNo" binding:"required,min=1"`
	Title                 *string `json:"title"`
	Summary               *string `json:"summary"`
	WordCount             *int    `json:"wordCount"`
	TargetWordCount       *int    `json:"targetWordCount"`
	Status                *string `json:"status"`
	ActualCharacterIDs    *string `json:"actualCharacterIds"`
	ActualFactionIDs      *string `json:"actualFactionIds"`
	ActualItemIDs         *string `json:"actualItemIds"`
	ActualHookIDs         *string `json:"actualHookIds"`
	ActualWorldSettingIDs *string `json:"actualWorldSettingIds"`
}

type UpdateInput struct {
	Title                 *string `json:"title"`
	Summary               *string `json:"summary"`
	WordCount             *int    `json:"wordCount"`
	TargetWordCount       *int    `json:"targetWordCount"`
	Status                *string `json:"status"`
	ActualCharacterIDs    *string `json:"actualCharacterIds"`
	ActualFactionIDs      *string `json:"actualFactionIds"`
	ActualItemIDs         *string `json:"actualItemIds"`
	ActualHookIDs         *string `json:"actualHookIds"`
	ActualWorldSettingIDs *string `json:"actualWorldSettingIds"`
}

type WriteStageInput struct {
	BookID    int64
	ChapterNo int
	Stage     string
	Summary   *string `json:"summary"`
	Content   string  `json:"content" binding:"required,min=1"`
}

type StageView struct {
	Metadata map[string]any `json:"metadata"`
	Summary  *string        `json:"summary"`
	Content  string         `json:"content"`
}

type StageHistoryEntry struct {
	ID        int64   `json:"id"`
	VersionNo int     `json:"versionNo"`
	Stage     string  `json:"stage"`
	Summary   *string `json:"summary"`
	Content   string  `json:"content"`
	WordCount *int    `json:"wordCount"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	IsCurrent bool    `json:"isCurrent"`
}

type WorkflowStateView struct {
	Status           string   `json:"status"`
	CurrentPlanID    *int64   `json:"currentPlanId"`
	CurrentDraftID   *int64   `json:"currentDraftId"`
	CurrentReviewID  *int64   `json:"currentReviewId"`
	CurrentFinalID   *int64   `json:"currentFinalId"`
	AvailableActions []string `json:"availableActions"`
	HasPlan          bool     `json:"hasPlan"`
	HasDraft         bool     `json:"hasDraft"`
	HasReview        bool     `json:"hasReview"`
	HasFinal         bool     `json:"hasFinal"`
}

func (s *Service) List(ctx context.Context, bookID int64, limit int, status string) ([]models.Chapter, error) {
	q := s.db.WithContext(ctx).Where("book_id = ?", bookID).Order("chapter_no ASC").Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.Chapter
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, bookID int64, chapterNo int) (*models.Chapter, error) {
	var row models.Chapter
	if err := s.db.WithContext(ctx).Where("book_id = ? AND chapter_no = ?", bookID, chapterNo).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound(fmt.Sprintf("chapter not found: book=%d, chapter=%d", bookID, chapterNo))
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*models.Chapter, error) {
	now := shared.NowISO()
	row := models.Chapter{
		BookID: in.BookID, ChapterNo: in.ChapterNo, Title: in.Title, Summary: in.Summary,
		WordCount: in.WordCount, TargetWordCount: in.TargetWordCount,
		Status:                shared.ChapterStatusTodo,
		ActualCharacterIDs:    in.ActualCharacterIDs,
		ActualFactionIDs:      in.ActualFactionIDs,
		ActualItemIDs:         in.ActualItemIDs,
		ActualHookIDs:         in.ActualHookIDs,
		ActualWorldSettingIDs: in.ActualWorldSettingIDs,
		CreatedAt:             now, UpdatedAt: now,
	}
	if in.Status != nil && *in.Status != "" {
		row.Status = *in.Status
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	// 增加 book.current_chapter_count(乐观更新)
	s.db.WithContext(ctx).Model(&models.Book{}).Where("id = ?", in.BookID).
		UpdateColumn("current_chapter_count", gorm.Expr("current_chapter_count + 1"))
	return &row, nil
}

func (s *Service) Update(ctx context.Context, bookID int64, chapterNo int, in UpdateInput) (*models.Chapter, error) {
	row, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": shared.NowISO()}
	if in.Title != nil {
		updates["title"] = in.Title
	}
	if in.Summary != nil {
		updates["summary"] = in.Summary
	}
	if in.WordCount != nil {
		updates["word_count"] = in.WordCount
	}
	if in.TargetWordCount != nil {
		updates["target_word_count"] = in.TargetWordCount
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.ActualCharacterIDs != nil {
		updates["actual_character_ids"] = in.ActualCharacterIDs
	}
	if in.ActualFactionIDs != nil {
		updates["actual_faction_ids"] = in.ActualFactionIDs
	}
	if in.ActualItemIDs != nil {
		updates["actual_item_ids"] = in.ActualItemIDs
	}
	if in.ActualHookIDs != nil {
		updates["actual_hook_ids"] = in.ActualHookIDs
	}
	if in.ActualWorldSettingIDs != nil {
		updates["actual_world_setting_ids"] = in.ActualWorldSettingIDs
	}
	if err := s.db.WithContext(ctx).Model(&models.Chapter{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, bookID, chapterNo)
}

func (s *Service) Remove(ctx context.Context, bookID int64, chapterNo int) error {
	row, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&models.Chapter{}, row.ID).Error; err != nil {
		return err
	}
	s.db.WithContext(ctx).Model(&models.Book{}).Where("id = ? AND current_chapter_count > 0", bookID).
		UpdateColumn("current_chapter_count", gorm.Expr("current_chapter_count - 1"))
	return nil
}

// GetStage 读当前指针对应的 stage 内容,stage ∈ plan/draft/review/final
func (s *Service) GetStage(ctx context.Context, bookID int64, chapterNo int, stage string) (*StageView, error) {
	chapter, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return nil, err
	}
	switch stage {
	case shared.StageNamePlan:
		if chapter.CurrentPlanID == nil {
			return nil, shared.NotFound("chapter has no current plan")
		}
		var row models.ChapterPlan
		if err := s.db.WithContext(ctx).First(&row, *chapter.CurrentPlanID).Error; err != nil {
			return nil, err
		}
		return &StageView{
			Metadata: map[string]any{"id": row.ID, "versionNo": row.VersionNo, "model": row.Model, "provider": row.Provider},
			Summary:  row.IntentSummary,
			Content:  row.Content,
		}, nil
	case shared.StageNameDraft:
		if chapter.CurrentDraftID == nil {
			return nil, shared.NotFound("chapter has no current draft")
		}
		var row models.ChapterDraft
		if err := s.db.WithContext(ctx).First(&row, *chapter.CurrentDraftID).Error; err != nil {
			return nil, err
		}
		return &StageView{
			Metadata: map[string]any{"id": row.ID, "versionNo": row.VersionNo, "wordCount": row.WordCount, "model": row.Model, "provider": row.Provider},
			Summary:  row.Summary,
			Content:  row.Content,
		}, nil
	case shared.StageNameReview:
		if chapter.CurrentReviewID == nil {
			return nil, shared.NotFound("chapter has no current review")
		}
		var row models.ChapterReview
		if err := s.db.WithContext(ctx).First(&row, *chapter.CurrentReviewID).Error; err != nil {
			return nil, err
		}
		return &StageView{
			Metadata: map[string]any{"id": row.ID, "versionNo": row.VersionNo, "model": row.Model, "provider": row.Provider},
			Summary:  row.Summary,
			Content:  row.RawResult,
		}, nil
	case shared.StageNameFinal:
		if chapter.CurrentFinalID == nil {
			return nil, shared.NotFound("chapter has no current final")
		}
		var row models.ChapterFinal
		if err := s.db.WithContext(ctx).First(&row, *chapter.CurrentFinalID).Error; err != nil {
			return nil, err
		}
		return &StageView{
			Metadata: map[string]any{"id": row.ID, "versionNo": row.VersionNo, "wordCount": row.WordCount},
			Summary:  row.Summary,
			Content:  row.Content,
		}, nil
	}
	return nil, shared.BadRequest("unsupported stage: " + stage)
}

func (s *Service) ListStageHistory(ctx context.Context, bookID int64, chapterNo int, stage string, limit int) ([]StageHistoryEntry, error) {
	chapter, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return nil, err
	}
	out := make([]StageHistoryEntry, 0)
	switch stage {
	case shared.StageNamePlan:
		var rows []models.ChapterPlan
		if err := s.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).Order("version_no DESC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, StageHistoryEntry{ID: r.ID, VersionNo: r.VersionNo, Stage: stage, Summary: r.IntentSummary, Content: r.Content, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, IsCurrent: chapter.CurrentPlanID != nil && *chapter.CurrentPlanID == r.ID})
		}
	case shared.StageNameDraft:
		var rows []models.ChapterDraft
		if err := s.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).Order("version_no DESC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, StageHistoryEntry{ID: r.ID, VersionNo: r.VersionNo, Stage: stage, Summary: r.Summary, Content: r.Content, WordCount: r.WordCount, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, IsCurrent: chapter.CurrentDraftID != nil && *chapter.CurrentDraftID == r.ID})
		}
	case shared.StageNameReview:
		var rows []models.ChapterReview
		if err := s.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).Order("version_no DESC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, StageHistoryEntry{ID: r.ID, VersionNo: r.VersionNo, Stage: stage, Summary: r.Summary, Content: r.RawResult, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, IsCurrent: chapter.CurrentReviewID != nil && *chapter.CurrentReviewID == r.ID})
		}
	case shared.StageNameFinal:
		var rows []models.ChapterFinal
		if err := s.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).Order("version_no DESC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, StageHistoryEntry{ID: r.ID, VersionNo: r.VersionNo, Stage: stage, Summary: r.Summary, Content: r.Content, WordCount: r.WordCount, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, IsCurrent: chapter.CurrentFinalID != nil && *chapter.CurrentFinalID == r.ID})
		}
	default:
		return nil, shared.BadRequest("unsupported stage: " + stage)
	}
	return out, nil
}

// WriteStage 把外部内容写入 plan/draft/final 之一,创建新版本并切换 current 指针。
func (s *Service) WriteStage(ctx context.Context, in WriteStageInput) (*StageHistoryEntry, error) {
	chapter, err := s.Get(ctx, in.BookID, in.ChapterNo)
	if err != nil {
		return nil, err
	}
	now := shared.NowISO()

	var entry StageHistoryEntry
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		switch in.Stage {
		case shared.StageNamePlan:
			var maxVer int
			tx.Model(&models.ChapterPlan{}).Where("chapter_id = ?", chapter.ID).Select("COALESCE(MAX(version_no),0)").Scan(&maxVer)
			row := models.ChapterPlan{
				BookID: in.BookID, ChapterID: chapter.ID, ChapterNo: in.ChapterNo, VersionNo: maxVer + 1,
				Status: "active", IntentSource: shared.PlanIntentSourceManual, Content: in.Content,
				IntentSummary: in.Summary, SourceType: shared.ChapterSourceImported,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Chapter{}).Where("id = ?", chapter.ID).Updates(map[string]any{
				"current_plan_id": row.ID, "status": shared.ChapterStatusPlanned, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			entry = StageHistoryEntry{ID: row.ID, VersionNo: row.VersionNo, Stage: in.Stage, Summary: in.Summary, Content: in.Content, CreatedAt: now, UpdatedAt: now, IsCurrent: true}
		case shared.StageNameDraft:
			var maxVer int
			tx.Model(&models.ChapterDraft{}).Where("chapter_id = ?", chapter.ID).Select("COALESCE(MAX(version_no),0)").Scan(&maxVer)
			wc := shared.EstimateWordCount(in.Content)
			row := models.ChapterDraft{
				BookID: in.BookID, ChapterID: chapter.ID, ChapterNo: in.ChapterNo, VersionNo: maxVer + 1,
				BasedOnPlanID: chapter.CurrentPlanID, BasedOnDraftID: chapter.CurrentDraftID, BasedOnReviewID: chapter.CurrentReviewID,
				Status: "active", Content: in.Content, Summary: in.Summary, WordCount: &wc,
				SourceType: shared.ChapterSourceImported,
				CreatedAt:  now, UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Chapter{}).Where("id = ?", chapter.ID).Updates(map[string]any{
				"current_draft_id": row.ID, "status": shared.ChapterStatusDrafted, "word_count": wc, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			entry = StageHistoryEntry{ID: row.ID, VersionNo: row.VersionNo, Stage: in.Stage, Summary: in.Summary, Content: in.Content, WordCount: &wc, CreatedAt: now, UpdatedAt: now, IsCurrent: true}
		case shared.StageNameFinal:
			var maxVer int
			tx.Model(&models.ChapterFinal{}).Where("chapter_id = ?", chapter.ID).Select("COALESCE(MAX(version_no),0)").Scan(&maxVer)
			wc := shared.EstimateWordCount(in.Content)
			row := models.ChapterFinal{
				BookID: in.BookID, ChapterID: chapter.ID, ChapterNo: in.ChapterNo, VersionNo: maxVer + 1,
				BasedOnDraftID: chapter.CurrentDraftID, Status: "active",
				Content: in.Content, Summary: in.Summary, WordCount: &wc,
				SourceType: shared.ChapterSourceImported,
				CreatedAt:  now, UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Chapter{}).Where("id = ?", chapter.ID).Updates(map[string]any{
				"current_final_id": row.ID, "status": shared.ChapterStatusApproved, "word_count": wc, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			entry = StageHistoryEntry{ID: row.ID, VersionNo: row.VersionNo, Stage: in.Stage, Summary: in.Summary, Content: in.Content, WordCount: &wc, CreatedAt: now, UpdatedAt: now, IsCurrent: true}
		default:
			return shared.BadRequest("unsupported writable stage: " + in.Stage)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// ExportStage 把 stage 序列化为 markdown 文本。
// review 阶段也允许导出(只读 raw_result),import 路由仍只允许 plan/draft/final。
func (s *Service) ExportStage(ctx context.Context, bookID int64, chapterNo int, stage string) (string, *shared.ChapterMarkdownMetadata, error) {
	chapter, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return "", nil, err
	}
	view, err := s.GetStage(ctx, bookID, chapterNo, stage)
	if err != nil {
		return "", nil, err
	}
	updatedAt, _ := view.Metadata["updatedAt"].(string)
	if updatedAt == "" {
		updatedAt = chapter.UpdatedAt
	}
	wordCount, _ := view.Metadata["wordCount"].(*int)
	if wordCount == nil {
		wordCount = chapter.WordCount
	}
	meta := shared.ChapterMarkdownMetadata{
		BookID:          bookID,
		ChapterNo:       chapterNo,
		Stage:           stage,
		Title:           chapter.Title,
		Status:          chapter.Status,
		WordCount:       wordCount,
		TargetWordCount: chapter.TargetWordCount,
		UpdatedAt:       updatedAt,
	}
	summary := view.Summary
	if summary == nil {
		summary = chapter.Summary
	}
	return shared.FormatChapterMarkdown(meta, summary, view.Content), &meta, nil
}

// ImportStage 把外部 markdown 写回 stage(plan/draft/final)。
// 与原 Node 项目一致:校验 frontmatter 的 book_id/chapter_no/stage 与入参完全一致;
// final 阶段除非 force=true 否则要求 chapter.status=approved。
func (s *Service) ImportStage(ctx context.Context, bookID int64, chapterNo int, stage string, raw string, force bool) (*StageHistoryEntry, error) {
	if stage == shared.StageNameReview {
		return nil, shared.BadRequest("review stage is not importable")
	}
	chapter, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return nil, err
	}
	parsed, err := shared.ParseChapterMarkdown(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Metadata.BookID != bookID {
		return nil, shared.BadRequest("imported markdown book_id does not match request")
	}
	if parsed.Metadata.ChapterNo != chapterNo {
		return nil, shared.BadRequest("imported markdown chapter_no does not match request")
	}
	if parsed.Metadata.Stage != stage {
		return nil, shared.BadRequest("imported markdown stage does not match request")
	}
	if stage == shared.StageNameFinal && chapter.Status != shared.ChapterStatusApproved && !force {
		return nil, shared.Conflict("importing final content requires approved chapter status or force=true", nil)
	}
	in := WriteStageInput{
		BookID:    bookID,
		ChapterNo: chapterNo,
		Stage:     stage,
		Summary:   parsed.Summary,
		Content:   parsed.Content,
	}
	return s.WriteStage(ctx, in)
}

func (s *Service) GetWorkflowState(ctx context.Context, bookID int64, chapterNo int) (*WorkflowStateView, error) {
	chapter, err := s.Get(ctx, bookID, chapterNo)
	if err != nil {
		return nil, err
	}
	v := &WorkflowStateView{
		Status:          chapter.Status,
		CurrentPlanID:   chapter.CurrentPlanID,
		CurrentDraftID:  chapter.CurrentDraftID,
		CurrentReviewID: chapter.CurrentReviewID,
		CurrentFinalID:  chapter.CurrentFinalID,
		HasPlan:         chapter.CurrentPlanID != nil,
		HasDraft:        chapter.CurrentDraftID != nil,
		HasReview:       chapter.CurrentReviewID != nil,
		HasFinal:        chapter.CurrentFinalID != nil,
	}
	v.AvailableActions = computeAvailableActions(v)
	return v, nil
}

func computeAvailableActions(s *WorkflowStateView) []string {
	out := make([]string, 0, 5)
	out = append(out, "plan")
	if s.HasPlan {
		out = append(out, "draft")
	}
	if s.HasDraft {
		out = append(out, "review")
	}
	if s.HasDraft && s.HasReview {
		out = append(out, "repair")
	}
	if s.HasDraft {
		out = append(out, "approve")
	}
	return out
}
