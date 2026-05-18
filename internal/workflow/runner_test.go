package workflow

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunner_BurstSubmitsAllExecuted(t *testing.T) {
	r := NewRunner(zap.NewNop(), 4)
	defer r.Shutdown(context.Background())

	const total = 200
	var done int32
	for i := 0; i < total; i++ {
		r.Submit(func(_ context.Context) {
			atomic.AddInt32(&done, 1)
		})
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&done) >= total {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.Equal(t, int32(total), atomic.LoadInt32(&done), "突发提交所有任务都应被执行")
}

func TestRunner_ShutdownCancelsInflight(t *testing.T) {
	r := NewRunner(zap.NewNop(), 2)
	started := make(chan struct{}, 2)
	canceled := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		r.Submit(func(ctx context.Context) {
			started <- struct{}{}
			<-ctx.Done()
			canceled <- struct{}{}
		})
	}
	// 等待 worker 启动
	<-started
	<-started
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r.Shutdown(shutdownCtx)
	require.Eventually(t, func() bool { return len(canceled) == 2 }, time.Second, 10*time.Millisecond)
}
