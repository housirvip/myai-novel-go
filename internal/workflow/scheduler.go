package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

type Scheduler struct {
	db           *gorm.DB
	runner       *Runner
	taskSvc      *Service
	logger       *zap.Logger
	bossID       string
	leaseTTL     time.Duration
	pollEvery    time.Duration
	backoffDelay time.Duration

	stopCh  chan struct{}
	wg      sync.WaitGroup
	closeMu sync.Mutex
	stopped bool
}

func NewScheduler(db *gorm.DB, runner *Runner, taskSvc *Service, logger *zap.Logger, bossID string) *Scheduler {
	return &Scheduler{
		db:           db,
		runner:       runner,
		taskSvc:      taskSvc,
		logger:       logger,
		bossID:       bossID,
		leaseTTL:     2 * time.Minute,
		pollEvery:    500 * time.Millisecond,
		backoffDelay: 5 * time.Second,
		stopCh:       make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("workflow.scheduler.start", zap.String("bossId", s.bossID), zap.Duration("pollEvery", s.pollEvery), zap.Duration("leaseTTL", s.leaseTTL))
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.loop(ctx)
	}()
}

func (s *Scheduler) Shutdown(ctx context.Context) {
	s.closeMu.Lock()
	if s.stopped {
		s.closeMu.Unlock()
		return
	}
	s.stopped = true
	close(s.stopCh)
	s.closeMu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(s.pollEvery)
	defer ticker.Stop()
	for {
		if err := s.dispatchDue(ctx); err != nil {
			s.logger.Warn("workflow.scheduler.dispatch_failed", zap.Error(err))
			select {
			case <-time.After(s.backoffDelay):
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) dispatchDue(ctx context.Context) error {
	for {
		row, token, err := s.claimNextDue(ctx)
		if err != nil {
			return err
		}
		if row == nil {
			return nil
		}
		taskID := row.ID
		workflowType := row.WorkflowType
		payload := row.RequestPayload
		s.logger.Info("workflow.scheduler.claimed", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType), zap.String("status", row.Status), zap.String("scheduledAt", row.ScheduledAt))
		if err := s.runner.Submit(func(runCtx context.Context) {
			if err := s.taskSvc.ExecuteClaimedTask(runCtx, taskID, token, workflowType, payload); err != nil {
				s.logger.Error("workflow.scheduler.execute_failed", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType), zap.Error(err))
			}
		}); err != nil {
			_ = s.releaseClaim(ctx, taskID, token)
			s.logger.Warn("workflow.scheduler.submit_failed", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType), zap.Error(err))
			return err
		}
		s.logger.Info("workflow.scheduler.dispatched", zap.Int64("taskId", taskID), zap.String("workflowType", workflowType))
	}
}

func (s *Scheduler) claimNextDue(ctx context.Context) (*models.WorkflowTask, string, error) {
	now := shared.NowISO()
	for {
		var rows []models.WorkflowTask
		res := s.db.WithContext(ctx).
			Where("status = ? AND scheduled_at <= ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)", shared.WorkflowTaskStatusPending, now, now).
			Order("scheduled_at ASC, id ASC").Limit(1).Find(&rows)
		if res.Error != nil {
			return nil, "", res.Error
		}
		if res.RowsAffected == 0 {
			return nil, "", nil
		}
		row := rows[0]

		leaseToken := fmt.Sprintf("%s-%d", s.bossID, row.ID)
		leaseExpires := time.Now().UTC().Add(s.leaseTTL).Format(shared.TimeLayout)
		claimRes := s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where(
			"id = ? AND status = ? AND scheduled_at <= ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)",
			row.ID, shared.WorkflowTaskStatusPending, now, now,
		).Updates(map[string]any{
			"status":           shared.WorkflowTaskStatusClaimed,
			"lease_owner":      s.bossID,
			"lease_token":      leaseToken,
			"lease_expires_at": leaseExpires,
			"updated_at":       shared.NowISO(),
		})
		if claimRes.Error != nil {
			return nil, "", claimRes.Error
		}
		if claimRes.RowsAffected == 0 {
			continue
		}
		s.logger.Info("workflow.scheduler.lease_claimed", zap.Int64("taskId", row.ID), zap.String("workflowType", row.WorkflowType), zap.String("bossId", s.bossID), zap.String("leaseExpiresAt", leaseExpires))
		row.Status = shared.WorkflowTaskStatusClaimed
		row.LeaseOwner = &s.bossID
		row.LeaseToken = &leaseToken
		row.LeaseExpiresAt = &leaseExpires
		return &row, leaseToken, nil
	}
}

func (s *Scheduler) releaseClaim(ctx context.Context, taskID int64, token string) error {
	return s.db.WithContext(ctx).Model(&models.WorkflowTask{}).Where("id = ? AND lease_token = ?", taskID, token).Updates(map[string]any{
		"status":           shared.WorkflowTaskStatusPending,
		"lease_owner":      nil,
		"lease_token":      nil,
		"lease_expires_at": nil,
		"updated_at":       shared.NowISO(),
	}).Error
}
