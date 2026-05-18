package workflow

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Runner 是简单的 goroutine 池 + 队列,负责执行后台 workflow tasks。
// 这里使用带缓冲的 chan 避免在突发提交时阻塞 HTTP 请求路径。
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

// Submit 把任务投递给 worker pool,如果队列满会丢入新 goroutine 兜底,保证 HTTP 路径不阻塞。
func (r *Runner) Submit(fn func(context.Context)) {
	r.closeMu.Lock()
	closed := r.closed
	r.closeMu.Unlock()
	if closed {
		return
	}
	select {
	case r.queue <- fn:
	default:
		// 队列饱和时退化为独立 goroutine,避免拒绝任务
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					r.logger.Error("workflow.runner.panic", zap.Any("panic", rec))
				}
			}()
			fn(r.rootCtx)
		}()
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
