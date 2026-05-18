package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/domain/workflows"
)

type TaskView struct {
	ID              int64       `json:"id"`
	BookID          int64       `json:"bookId"`
	ChapterID       int64       `json:"chapterId"`
	ChapterNo       int         `json:"chapterNo"`
	WorkflowType    string      `json:"workflowType"`
	Status          string      `json:"status"`
	Stage           *string     `json:"stage"`
	ProgressPercent *int        `json:"progressPercent"`
	StartedAt       *string     `json:"startedAt"`
	FinishedAt      *string     `json:"finishedAt"`
	CurrentPlanID   *int64      `json:"currentPlanId"`
	CurrentDraftID  *int64      `json:"currentDraftId"`
	Result          interface{} `json:"result"`
	Error           interface{} `json:"error"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
}

type Service struct {
	db     *gorm.DB
	runner *Runner
	logger *zap.Logger

	plan         *workflows.PlanWorkflow
	draft        *workflows.DraftWorkflow
	review       *workflows.ReviewWorkflow
	repair       *workflows.RepairWorkflow
	approve      *workflows.ApproveWorkflow
	stageSummary *workflows.StageSummaryWorkflow
}

func NewService(
	db *gorm.DB, runner *Runner, logger *zap.Logger,
	plan *workflows.PlanWorkflow, draft *workflows.DraftWorkflow,
	review *workflows.ReviewWorkflow, repair *workflows.RepairWorkflow,
	approve *workflows.ApproveWorkflow, stageSummary *workflows.StageSummaryWorkflow,
) *Service {
	return &Service{db: db, runner: runner, logger: logger,
		plan: plan, draft: draft, review: review, repair: repair, approve: approve, stageSummary: stageSummary}
}

func (s *Service) Get(ctx context.Context, id int64) (*TaskView, error) {
	var row models.WorkflowTask
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.NotFound(fmt.Sprintf("task not found: %d", id))
		}
		return nil, err
	}
	return toView(&row), nil
}

func (s *Service) GetLatest(ctx context.Context, bookID int64, chapterNo int, taskType string) (*TaskView, error) {
	var row models.WorkflowTask
	err := s.db.WithContext(ctx).Where("book_id = ? AND chapter_no = ? AND workflow_type = ?", bookID, chapterNo, taskType).
		Order("id DESC").First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toView(&row), nil
}

func (s *Service) StartPlan(ctx context.Context, in workflows.PlanInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypePlan, in, func(ctx context.Context, notify workflows.StageNotifier) (any, error) {
		return s.plan.Run(ctx, in, notify)
	})
}

func (s *Service) StartDraft(ctx context.Context, in workflows.DraftInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeDraft, in, func(ctx context.Context, notify workflows.StageNotifier) (any, error) {
		return s.draft.Run(ctx, in, notify)
	})
}

func (s *Service) StartReview(ctx context.Context, in workflows.ReviewInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeReview, in, func(ctx context.Context, notify workflows.StageNotifier) (any, error) {
		return s.review.Run(ctx, in, notify)
	})
}

func (s *Service) StartRepair(ctx context.Context, in workflows.RepairInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeRepair, in, func(ctx context.Context, notify workflows.StageNotifier) (any, error) {
		return s.repair.Run(ctx, in, notify)
	})
}

func (s *Service) StartApprove(ctx context.Context, in workflows.ApproveInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeApprove, in, func(ctx context.Context, notify workflows.StageNotifier) (any, error) {
		return s.approve.Run(ctx, in, notify)
	})
}

func (s *Service) StartAuthorIntent(ctx context.Context, in workflows.AuthorIntentInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeAuthorIntent, in, func(ctx context.Context, notify workflows.StageNotifier) (any, error) {
		return s.plan.GenerateAuthorIntent(ctx, in, notify)
	})
}

func (s *Service) startTask(
	ctx context.Context, bookID int64, chapterNo int, taskType string,
	payload any, exec func(ctx context.Context, notify workflows.StageNotifier) (any, error),
) (*TaskView, error) {
	var chapter models.Chapter
	if err := s.db.WithContext(ctx).Where("book_id = ? AND chapter_no = ?", bookID, chapterNo).First(&chapter).Error; err != nil {
		return nil, shared.NotFound(fmt.Sprintf("chapter not found: book=%d, chapter=%d", bookID, chapterNo))
	}
	var active models.WorkflowTask
	err := s.db.WithContext(ctx).Where("book_id = ? AND chapter_no = ? AND status IN ?", bookID, chapterNo, []string{shared.WorkflowTaskStatusPending, shared.WorkflowTaskStatusRunning}).
		Order("id DESC").First(&active).Error
	if err == nil {
		return nil, shared.Conflict("workflow task already running", map[string]any{"taskId": active.ID, "workflowType": active.WorkflowType})
	}
	now := shared.NowISO()
	payloadStr := mustJSON(payload)
	stage := shared.WorkflowStageQueued
	row := models.WorkflowTask{
		BookID: bookID, ChapterID: chapter.ID, ChapterNo: chapterNo,
		WorkflowType: taskType, Status: shared.WorkflowTaskStatusPending,
		Stage: &stage, RequestPayload: payloadStr,
		AttemptCount: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}

	taskID := row.ID
	s.runner.Submit(func(runCtx context.Context) {
		s.executeAsync(runCtx, taskID, exec)
	})
	return toView(&row), nil
}

func (s *Service) executeAsync(ctx context.Context, taskID int64, exec func(ctx context.Context, notify workflows.StageNotifier) (any, error)) {
	now := shared.NowISO()
	startedAt := now
	if err := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":     shared.WorkflowTaskStatusRunning,
		"started_at": startedAt,
		"updated_at": now,
	}).Error; err != nil {
		s.logger.Error("workflow.task.start_failed", zap.Int64("taskId", taskID), zap.Error(err))
		return
	}

	var lastStage string
	var lastProgress *int

	notify := func(stage string, progress *int) error {
		if stage == lastStage && progEq(progress, lastProgress) {
			return nil
		}
		lastStage = stage
		lastProgress = progress
		updates := map[string]any{
			"stage":      stage,
			"updated_at": shared.NowISO(),
		}
		if progress != nil {
			updates["progress_percent"] = *progress
		}
		return s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ?", taskID).Updates(updates).Error
	}

	result, err := exec(ctx, notify)
	if err != nil {
		s.markFailed(ctx, taskID, err)
		s.logger.Error("workflow.task.failed", zap.Int64("taskId", taskID), zap.Error(err))
		return
	}
	s.markSucceeded(ctx, taskID, result)
}

func (s *Service) markSucceeded(ctx context.Context, taskID int64, result any) {
	now := shared.NowISO()
	resultJSON := mustJSON(result)
	planID, draftID := extractPointerIDs(result)
	progress := 100
	updates := map[string]any{
		"status":           shared.WorkflowTaskStatusSucceeded,
		"result_payload":   resultJSON,
		"progress_percent": progress,
		"finished_at":      now,
		"updated_at":       now,
	}
	if planID != nil {
		updates["current_plan_id"] = *planID
	}
	if draftID != nil {
		updates["current_draft_id"] = *draftID
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		s.logger.Error("workflow.task.finalize_failed", zap.Int64("taskId", taskID), zap.Error(err))
	}
}

func (s *Service) markFailed(ctx context.Context, taskID int64, err error) {
	now := shared.NowISO()
	code := "internal_error"
	message := err.Error()
	var details *string
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		code = appErr.Code
		message = appErr.Message
		if appErr.Details != nil {
			d := mustJSON(appErr.Details)
			details = &d
		}
	}
	updates := map[string]any{
		"status":        shared.WorkflowTaskStatusFailed,
		"error_code":    code,
		"error_message": message,
		"error_details": details,
		"finished_at":   now,
		"updated_at":    now,
	}
	if e := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ?", taskID).Updates(updates).Error; e != nil {
		s.logger.Error("workflow.task.fail_persist_failed", zap.Int64("taskId", taskID), zap.Error(e))
	}
}

// List 列出某章节的全部 workflow_tasks(按 id DESC),用于 UI 显示历史。
func (s *Service) List(ctx context.Context, bookID int64, chapterNo int, limit int) ([]*TaskView, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []models.WorkflowTask
	if err := s.db.WithContext(ctx).
		Where("book_id = ? AND chapter_no = ?", bookID, chapterNo).
		Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*TaskView, 0, len(rows))
	for i := range rows {
		out = append(out, toView(&rows[i]))
	}
	return out, nil
}

// Terminate 把 pending/running 状态的任务直接标 failed,error_code=terminated_by_user。
// 不主动 cancel goroutine(因为 ctx 是从 runner 共享的);worker 跑完会发现 task 状态被更新,
// 之后的 markSucceeded/markFailed 会被覆盖 —— 接受这点轻微竞争换来零额外锁开销。
func (s *Service) Terminate(ctx context.Context, taskID int64) (*TaskView, error) {
	view, err := s.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if view.Status != shared.WorkflowTaskStatusPending && view.Status != shared.WorkflowTaskStatusRunning {
		return nil, shared.Conflict("task is not pending/running, cannot terminate", map[string]any{"status": view.Status})
	}
	now := shared.NowISO()
	updates := map[string]any{
		"status":        shared.WorkflowTaskStatusFailed,
		"error_code":    "terminated_by_user",
		"error_message": "Task terminated by user",
		"finished_at":   now,
		"updated_at":    now,
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, taskID)
}

// RecoverInterrupted 在进程启动时把残留的 pending/running 任务标 failed,
// 防止 UI 永远显示 running 而后端早已不在执行。
func (s *Service) RecoverInterrupted(ctx context.Context) (int64, error) {
	now := shared.NowISO()
	res := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).
		Where("status IN ?", []string{shared.WorkflowTaskStatusPending, shared.WorkflowTaskStatusRunning}).
		Updates(map[string]any{
			"status":        shared.WorkflowTaskStatusFailed,
			"error_code":    "process_restart",
			"error_message": "Task interrupted by process restart",
			"finished_at":   now,
			"updated_at":    now,
		})
	return res.RowsAffected, res.Error
}

func toView(row *models.WorkflowTask) *TaskView {
	v := &TaskView{
		ID: row.ID, BookID: row.BookID, ChapterID: row.ChapterID, ChapterNo: row.ChapterNo,
		WorkflowType: row.WorkflowType, Status: row.Status, Stage: row.Stage,
		ProgressPercent: row.ProgressPercent, StartedAt: row.StartedAt, FinishedAt: row.FinishedAt,
		CurrentPlanID: row.CurrentPlanID, CurrentDraftID: row.CurrentDraftID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.ResultPayload != nil && *row.ResultPayload != "" {
		var any interface{}
		_ = json.Unmarshal([]byte(*row.ResultPayload), &any)
		v.Result = any
	}
	if row.ErrorCode != nil || row.ErrorMessage != nil {
		errObj := map[string]any{
			"code":    derefStr(row.ErrorCode),
			"message": derefStr(row.ErrorMessage),
		}
		if row.ErrorDetails != nil && *row.ErrorDetails != "" {
			var d interface{}
			_ = json.Unmarshal([]byte(*row.ErrorDetails), &d)
			errObj["details"] = d
		}
		v.Error = errObj
	}
	return v
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func progEq(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// extractPointerIDs 从结果中找 planId / draftId(reflection-free 路径)
func extractPointerIDs(result any) (*int64, *int64) {
	switch v := result.(type) {
	case *workflows.PlanOutput:
		return &v.PlanID, nil
	case *workflows.DraftOutput:
		return &v.BasedOnPlanID, &v.DraftID
	case *workflows.ReviewOutput:
		return nil, &v.DraftID
	case *workflows.RepairOutput:
		return nil, &v.DraftID
	case *workflows.ApproveOutput:
		return nil, nil
	}
	return nil, nil
}
