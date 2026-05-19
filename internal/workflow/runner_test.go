package workflow

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunner_BurstSubmitsAllExecuted(t *testing.T) {
	r := NewRunner(zap.NewNop(), 4)
	defer r.Shutdown(context.Background())

	const total = 32
	var done int32
	for i := 0; i < total; i++ {
		require.NoError(t, r.Submit(func(_ context.Context) {
			atomic.AddInt32(&done, 1)
		}))
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&done) >= total {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.Equal(t, int32(total), atomic.LoadInt32(&done), "队列容量内的突发提交都应被执行")
}

func TestRunner_ShutdownCancelsInflight(t *testing.T) {
	r := NewRunner(zap.NewNop(), 2)
	started := make(chan struct{}, 2)
	canceled := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		require.NoError(t, r.Submit(func(ctx context.Context) {
			started <- struct{}{}
			<-ctx.Done()
			canceled <- struct{}{}
		}))
	}
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

func TestRunner_SubmitAfterShutdownReturnsClosed(t *testing.T) {
	r := NewRunner(zap.NewNop(), 1)
	r.Shutdown(context.Background())
	var ran int32
	err := r.Submit(func(context.Context) {
		atomic.AddInt32(&ran, 1)
	})
	require.ErrorIs(t, err, ErrRunnerClosed)
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, int32(0), atomic.LoadInt32(&ran))
}

func TestRunner_SaturatedQueueReturnsError(t *testing.T) {
	r := NewRunner(zap.NewNop(), 1)
	defer r.Shutdown(context.Background())

	blocked := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, r.Submit(func(ctx context.Context) {
		close(blocked)
		<-release
		_ = ctx
	}))
	<-blocked

	for i := 0; i < cap(r.queue); i++ {
		require.NoError(t, r.Submit(func(context.Context) {}))
	}

	err := r.Submit(func(context.Context) {})
	var qErr *QueueFullError
	require.ErrorAs(t, err, &qErr)
	require.Equal(t, 1, qErr.Workers)
	require.Equal(t, cap(r.queue), qErr.QueueCap)

	close(release)
}

func TestRunner_DoesNotExceedWorkerCount(t *testing.T) {
	r := NewRunner(zap.NewNop(), 2)
	defer r.Shutdown(context.Background())

	const total = 10
	start := make(chan struct{})
	finish := make(chan struct{})
	var running int32
	var maxRunning int32
	var wg sync.WaitGroup

	for i := 0; i < total; i++ {
		wg.Add(1)
		require.NoError(t, r.Submit(func(context.Context) {
			defer wg.Done()
			<-start
			cur := atomic.AddInt32(&running, 1)
			for {
				seen := atomic.LoadInt32(&maxRunning)
				if cur <= seen || atomic.CompareAndSwapInt32(&maxRunning, seen, cur) {
					break
				}
			}
			<-finish
			atomic.AddInt32(&running, -1)
		}))
	}

	close(start)
	require.Eventually(t, func() bool { return atomic.LoadInt32(&maxRunning) == 2 }, time.Second, 10*time.Millisecond)
	close(finish)
	wg.Wait()
	require.Equal(t, int32(2), atomic.LoadInt32(&maxRunning))
}

func TestRunner_PanicIsRecovered(t *testing.T) {
	r := NewRunner(zap.NewNop(), 1)
	defer r.Shutdown(context.Background())
	var ran int32
	require.NoError(t, r.Submit(func(context.Context) {
		atomic.AddInt32(&ran, 1)
		panic("boom")
	}))
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&ran) == 1
	}, time.Second, 10*time.Millisecond)
}
