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

func TestNewRunner_DefaultWorkerCount(t *testing.T) {
	r := NewRunner(zap.NewNop(), 0)
	defer r.Shutdown(context.Background())
	require.Equal(t, 4, r.workers)
	require.NotNil(t, r.queue)
}

func TestRunner_SubmitAfterShutdownIsIgnored(t *testing.T) {
	r := NewRunner(zap.NewNop(), 1)
	r.Shutdown(context.Background())
	var ran int32
	r.Submit(func(context.Context) {
		atomic.AddInt32(&ran, 1)
	})
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, int32(0), atomic.LoadInt32(&ran))
}

func TestRunner_SaturatedQueueFallsBackToGoroutine(t *testing.T) {
	r := NewRunner(zap.NewNop(), 1)

	blocked := make(chan struct{})
	release := make(chan struct{})
	r.Submit(func(ctx context.Context) {
		close(blocked)
		<-release
		_ = ctx
	})
	<-blocked

	for i := 0; i < 8; i++ {
		r.Submit(func(context.Context) {})
	}

	var ran int32
	r.Submit(func(context.Context) {
		atomic.AddInt32(&ran, 1)
	})

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&ran) == 1
	}, time.Second, 10*time.Millisecond)

	close(release)
	r.Shutdown(context.Background())
}

func TestRunner_PanicIsRecovered(t *testing.T) {
	r := NewRunner(zap.NewNop(), 1)
	defer r.Shutdown(context.Background())
	var ran int32
	r.Submit(func(context.Context) {
		atomic.AddInt32(&ran, 1)
		panic("boom")
	})
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&ran) == 1
	}, time.Second, 10*time.Millisecond)
}
