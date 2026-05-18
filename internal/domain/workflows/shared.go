package workflows

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/shared"
)

// StageNotifier 用于在 LLM 长任务过程中向 workflow_tasks 写阶段进度。
type StageNotifier func(stage string, progress *int) error

func NoopNotifier(_ string, _ *int) error { return nil }

// PointerSnapshot 抓取 chapter 当前阶段指针,用于事务前后比对。
type PointerSnapshot struct {
	CurrentPlanID   *int64
	CurrentDraftID  *int64
	CurrentReviewID *int64
	CurrentFinalID  *int64
}

func TakeSnapshot(c *models.Chapter) PointerSnapshot {
	return PointerSnapshot{
		CurrentPlanID:   c.CurrentPlanID,
		CurrentDraftID:  c.CurrentDraftID,
		CurrentReviewID: c.CurrentReviewID,
		CurrentFinalID:  c.CurrentFinalID,
	}
}

// AssertPointersUnchanged 在事务里再次读 chapter 与 snapshot 比对,防止生成期间被并发改动。
func AssertPointersUnchanged(tx *gorm.DB, bookID int64, chapterNo int, snap PointerSnapshot) (*models.Chapter, error) {
	var current models.Chapter
	if err := tx.Where("book_id = ? AND chapter_no = ?", bookID, chapterNo).First(&current).Error; err != nil {
		return nil, err
	}
	if !int64PtrEq(current.CurrentPlanID, snap.CurrentPlanID) ||
		!int64PtrEq(current.CurrentDraftID, snap.CurrentDraftID) ||
		!int64PtrEq(current.CurrentReviewID, snap.CurrentReviewID) ||
		!int64PtrEq(current.CurrentFinalID, snap.CurrentFinalID) {
		return nil, shared.Conflict("chapter pointer changed during workflow", map[string]any{
			"expected": snap, "actual": map[string]any{
				"currentPlanId":  current.CurrentPlanID,
				"currentDraftId": current.CurrentDraftID,
				"currentReviewId": current.CurrentReviewID,
				"currentFinalId": current.CurrentFinalID,
			},
		})
	}
	return &current, nil
}

func int64PtrEq(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// LoadChapter 是工作流共用入口,加载并校验章节存在。
func LoadChapter(ctx context.Context, db *gorm.DB, bookID int64, chapterNo int) (*models.Chapter, error) {
	var c models.Chapter
	if err := db.WithContext(ctx).Where("book_id = ? AND chapter_no = ?", bookID, chapterNo).First(&c).Error; err != nil {
		return nil, shared.NotFound("chapter not found")
	}
	return &c, nil
}

// ReadIntentConstraints 从 chapter_plans 行里读出 IntentConstraints。
func ReadIntentConstraints(plan *models.ChapterPlan) planning.IntentConstraints {
	out := planning.IntentConstraints{}
	if plan.IntentSummary != nil {
		out.IntentSummary = *plan.IntentSummary
	}
	if plan.IntentMustInclude != nil && *plan.IntentMustInclude != "" {
		var arr []string
		_ = json.Unmarshal([]byte(*plan.IntentMustInclude), &arr)
		out.MustInclude = arr
	}
	if plan.IntentMustAvoid != nil && *plan.IntentMustAvoid != "" {
		var arr []string
		_ = json.Unmarshal([]byte(*plan.IntentMustAvoid), &arr)
		out.MustAvoid = arr
	}
	return out
}

func LoadRetrievedContext(plan *models.ChapterPlan) (*planning.RetrievedContext, error) {
	if plan.RetrievedContext == nil || *plan.RetrievedContext == "" {
		return &planning.RetrievedContext{}, nil
	}
	var ctxObj planning.RetrievedContext
	if err := json.Unmarshal([]byte(*plan.RetrievedContext), &ctxObj); err != nil {
		return nil, err
	}
	return &ctxObj, nil
}

func Notify(notifier StageNotifier, stage string, progress int) {
	if notifier == nil {
		return
	}
	p := progress
	_ = notifier(stage, &p)
}

func nextVersion(tx *gorm.DB, table string, chapterID int64) (int, error) {
	var n int
	if err := tx.Table(table).Where("chapter_id = ?", chapterID).Select("COALESCE(MAX(version_no),0)").Scan(&n).Error; err != nil {
		return 0, err
	}
	return n + 1, nil
}
