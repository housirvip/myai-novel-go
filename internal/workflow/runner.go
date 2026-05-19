package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

var ErrRunnerClosed = errors.New("workflow runner closed")

type QueueFullError struct {
	Workers  int
	QueueLen int
	QueueCap int
}

func (e *QueueFullError) Error() string {
	return fmt.Sprintf("workflow queue full: workers=%d queue=%d/%d", e.Workers, e.QueueLen, e.QueueCap)
}

// Runner 是简单的 goroutine 池 + 队列,负责执行后台 workflow tasks。
// 这里使用带缓冲的 chan 作为有界队列,超过上限时直接拒绝提交。
type Runner struct {
	logger     *zap.Logger
	queue      chan func(context.Context)
	workers    int
	wg         sync.WaitGroup
	rootCtx    context.Context
	cancelFunc context.CancelFunc
	closed     bool
	closeMu    sync.Mutex
}

func NewRunner(logger *zap.Logger, workers int) *Runner {
	if workers <= 0 {
		workers = 4
	}
	rootCtx, cancel := context.WithCancel(context.Background())
	r := &Runner{
		logger:     logger,
		queue:      make(chan func(context.Context), workers*8),
		workers:    workers,
		rootCtx:    rootCtx,
		cancelFunc: cancel,
	}
	for i := 0; i < workers; i++ {
		r.wg.Add(1)
		go r.worker(i)
	}
	return r
}

func (r *Runner) worker(id int) {
	defer r.wg.Done()
	for {
		select {
		case <-r.rootCtx.Done():
			return
		case fn, ok := <-r.queue:
			if !ok {
				return
			}
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						r.logger.Error("workflow.runner.panic", zap.Any("panic", rec), zap.Int("worker", id))
					}
				}()
				fn(r.rootCtx)
			}()
		}
	}
}

// Submit 把任务投递给 worker pool,队列满时直接返回错误。
func (r *Runner) Submit(fn func(context.Context)) error {
	r.closeMu.Lock()
	defer r.closeMu.Unlock()
	if r.closed {
		return ErrRunnerClosed
	}
	select {
	case r.queue <- fn:
		return nil
	default:
		r.logger.Warn("workflow.runner.queue_full", zap.Int("workers", r.workers), zap.Int("queueLen", len(r.queue)), zap.Int("queueCap", cap(r.queue)))
		return &QueueFullError{Workers: r.workers, QueueLen: len(r.queue), QueueCap: cap(r.queue)}
	}
}

// Shutdown 取消正在运行的 ctx 并等待 worker 收尾。
func (r *Runner) Shutdown(ctx context.Context) {
	r.closeMu.Lock()
	if r.closed {
		r.closeMu.Unlock()
		return
	}
	r.closed = true
	close(r.queue)
	r.closeMu.Unlock()

	r.cancelFunc()
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
