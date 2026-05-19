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
	ScheduledAt     string      `json:"scheduledAt"`
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
	logger *zap.Logger

	plan         *workflows.PlanWorkflow
	draft        *workflows.DraftWorkflow
	review       *workflows.ReviewWorkflow
	repair       *workflows.RepairWorkflow
	approve      *workflows.ApproveWorkflow
	stageSummary *workflows.StageSummaryWorkflow
}

func NewService(
	db *gorm.DB, logger *zap.Logger,
	plan *workflows.PlanWorkflow, draft *workflows.DraftWorkflow,
	review *workflows.ReviewWorkflow, repair *workflows.RepairWorkflow,
	approve *workflows.ApproveWorkflow, stageSummary *workflows.StageSummaryWorkflow,
) *Service {
	return &Service{db: db, logger: logger,
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
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypePlan, in)
}

func (s *Service) StartDraft(ctx context.Context, in workflows.DraftInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeDraft, in)
}

func (s *Service) StartReview(ctx context.Context, in workflows.ReviewInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeReview, in)
}

func (s *Service) StartRepair(ctx context.Context, in workflows.RepairInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeRepair, in)
}

func (s *Service) StartApprove(ctx context.Context, in workflows.ApproveInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeApprove, in)
}

func (s *Service) StartAuthorIntent(ctx context.Context, in workflows.AuthorIntentInput) (*TaskView, error) {
	return s.startTask(ctx, in.BookID, in.ChapterNo, shared.WorkflowTaskTypeAuthorIntent, in)
}

func (s *Service) startTask(ctx context.Context, bookID int64, chapterNo int, taskType string, payload any) (*TaskView, error) {
	var chapter models.Chapter
	if err := s.db.WithContext(ctx).Where("book_id = ? AND chapter_no = ?", bookID, chapterNo).First(&chapter).Error; err != nil {
		return nil, shared.NotFound(fmt.Sprintf("chapter not found: book=%d, chapter=%d", bookID, chapterNo))
	}
	var active models.WorkflowTask
	err := s.db.WithContext(ctx).
		Where("book_id = ? AND chapter_no = ? AND status IN ?", bookID, chapterNo, []string{shared.WorkflowTaskStatusPending, shared.WorkflowTaskStatusClaimed, shared.WorkflowTaskStatusRunning}).
		Order("id DESC").First(&active).Error
	if err == nil {
		return nil, shared.Conflict("workflow task already running", map[string]any{"taskId": active.ID, "workflowType": active.WorkflowType})
	}

	now := shared.NowISO()
	payloadStr := mustJSON(payload)
	stage := shared.WorkflowStageQueued
	row := models.WorkflowTask{
		BookID:         bookID,
		ChapterID:      chapter.ID,
		ChapterNo:      chapterNo,
		WorkflowType:   taskType,
		Status:         shared.WorkflowTaskStatusPending,
		Stage:          &stage,
		ScheduledAt:    now,
		RequestPayload: payloadStr,
		AttemptCount:   1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return toView(&row), nil
}

func (s *Service) ExecuteClaimedTask(ctx context.Context, taskID int64, leaseToken, workflowType, payload string) error {
	s.logger.Info("workflow.task.worker_start", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType))
	if err := s.markRunning(ctx, taskID, leaseToken); err != nil {
		s.logger.Warn("workflow.task.worker_start_failed", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType), zap.Error(err))
		return err
	}
	s.logger.Info("workflow.task.running", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType))

	var lastStage string
	var lastProgress *int
	notify := func(stage string, progress *int) error {
		if stage == lastStage && progEq(progress, lastProgress) {
			return nil
		}
		lastStage = stage
		lastProgress = progress
		s.logger.Info("workflow.task.progress", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType), zap.String("stage", stage), zap.Any("progress", progress))
		return s.notifyProgress(ctx, taskID, stage, progress)
	}

	result, err := s.runWorkflow(ctx, workflowType, payload, notify)
	if err != nil {
		s.markFailed(ctx, taskID, err)
		s.logger.Error("workflow.task.failed", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType), zap.Error(err))
		return err
	}
	s.markSucceeded(ctx, taskID, result)
	s.logger.Info("workflow.task.succeeded", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType))
	return nil
}

func (s *Service) runWorkflow(ctx context.Context, workflowType, payload string, notify workflows.StageNotifier) (any, error) {
	switch workflowType {
	case shared.WorkflowTaskTypePlan:
		var in workflows.PlanInput
		if err := json.Unmarshal([]byte(payload), &in); err != nil {
			return nil, err
		}
		return s.plan.Run(ctx, in, notify)
	case shared.WorkflowTaskTypeDraft:
		var in workflows.DraftInput
		if err := json.Unmarshal([]byte(payload), &in); err != nil {
			return nil, err
		}
		return s.draft.Run(ctx, in, notify)
	case shared.WorkflowTaskTypeReview:
		var in workflows.ReviewInput
		if err := json.Unmarshal([]byte(payload), &in); err != nil {
			return nil, err
		}
		return s.review.Run(ctx, in, notify)
	case shared.WorkflowTaskTypeRepair:
		var in workflows.RepairInput
		if err := json.Unmarshal([]byte(payload), &in); err != nil {
			return nil, err
		}
		return s.repair.Run(ctx, in, notify)
	case shared.WorkflowTaskTypeApprove:
		var in workflows.ApproveInput
		if err := json.Unmarshal([]byte(payload), &in); err != nil {
			return nil, err
		}
		return s.approve.Run(ctx, in, notify)
	case shared.WorkflowTaskTypeAuthorIntent:
		var in workflows.AuthorIntentInput
		if err := json.Unmarshal([]byte(payload), &in); err != nil {
			return nil, err
		}
		return s.plan.GenerateAuthorIntent(ctx, in, notify)
	default:
		return nil, fmt.Errorf("unsupported workflow type: %s", workflowType)
	}
}

func (s *Service) markRunning(ctx context.Context, taskID int64, leaseToken string) error {
	now := shared.NowISO()
	res := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where(
		"id = ? AND status = ? AND lease_token = ?",
		taskID, shared.WorkflowTaskStatusClaimed, leaseToken,
	).Updates(map[string]any{
		"status":           shared.WorkflowTaskStatusRunning,
		"started_at":       now,
		"lease_owner":      nil,
		"lease_token":      nil,
		"lease_expires_at": nil,
		"updated_at":       now,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.Conflict("task is no longer claimable", map[string]any{"taskId": taskID})
	}
	s.logger.Info("workflow.task.mark_running", zap.Int64("taskId", taskID))
	return nil
}

func (s *Service) notifyProgress(ctx context.Context, taskID int64, stage string, progress *int) error {
	updates := map[string]any{
		"stage":      stage,
		"updated_at": shared.NowISO(),
	}
	if progress != nil {
		updates["progress_percent"] = *progress
	}
	res := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ? AND status = ?", taskID, shared.WorkflowTaskStatusRunning).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	return nil
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
		"lease_owner":      nil,
		"lease_token":      nil,
		"lease_expires_at": nil,
	}
	if planID != nil {
		updates["current_plan_id"] = *planID
	}
	if draftID != nil {
		updates["current_draft_id"] = *draftID
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ? AND status = ?", taskID, shared.WorkflowTaskStatusRunning).Updates(updates).Error; err != nil {
		s.logger.Error("workflow.task.finalize_failed", zap.Int64("taskId", taskID), zap.Error(err))
		return
	}
	s.logger.Info("workflow.task.mark_succeeded", zap.Int64("taskId", taskID))
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
		"status":           shared.WorkflowTaskStatusFailed,
		"error_code":       code,
		"error_message":    message,
		"error_details":    details,
		"finished_at":      now,
		"updated_at":       now,
		"lease_owner":      nil,
		"lease_token":      nil,
		"lease_expires_at": nil,
	}
	if e := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ? AND status = ?", taskID, shared.WorkflowTaskStatusRunning).Updates(updates).Error; e != nil {
		s.logger.Error("workflow.task.fail_persist_failed", zap.Int64("taskId", taskID), zap.Error(e))
		return
	}
	s.logger.Info("workflow.task.mark_failed", zap.Int64("taskId", taskID), zap.String("code", code))
}

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

func (s *Service) Terminate(ctx context.Context, taskID int64) (*TaskView, error) {
	view, err := s.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if view.Status != shared.WorkflowTaskStatusPending && view.Status != shared.WorkflowTaskStatusClaimed && view.Status != shared.WorkflowTaskStatusRunning {
		return nil, shared.Conflict("task is not pending/claimed/running, cannot terminate", map[string]any{"status": view.Status})
	}
	now := shared.NowISO()
	updates := map[string]any{
		"status":           shared.WorkflowTaskStatusFailed,
		"error_code":       "terminated_by_user",
		"error_message":    "Task terminated by user",
		"finished_at":      now,
		"updated_at":       now,
		"lease_owner":      nil,
		"lease_token":      nil,
		"lease_expires_at": nil,
	}
	if err := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, taskID)
}

func (s *Service) RecoverInterrupted(ctx context.Context) (int64, error) {
	now := shared.NowISO()
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).
		Where("status = ?", shared.WorkflowTaskStatusClaimed).
		Updates(map[string]any{
			"status":           shared.WorkflowTaskStatusPending,
			"lease_owner":      nil,
			"lease_token":      nil,
			"lease_expires_at": nil,
			"updated_at":       now,
		}).Error; err != nil {
		return total, err
	}
	res := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).
		Where("status = ?", shared.WorkflowTaskStatusRunning).
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
		ScheduledAt: row.ScheduledAt, ProgressPercent: row.ProgressPercent, StartedAt: row.StartedAt, FinishedAt: row.FinishedAt,
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
	case *workflows.AuthorIntentOutput:
		return nil, nil
	}
	return nil, nil
}
