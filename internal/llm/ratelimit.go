package llm

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// Limiter 限制全局对外 LLM 调用速率,防止打满 provider 配额。
type Limiter struct {
	limiter *rate.Limiter
}

func NewLimiter(rps int) *Limiter {
	if rps <= 0 {
		rps = 20
	}
	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(rps), rps),
	}
}

func (l *Limiter) Wait(ctx context.Context) error {
	if l == nil || l.limiter == nil {
		return nil
	}
	return l.limiter.Wait(ctx)
}

type RateLimitedClient struct {
	inner   Client
	limiter *Limiter
	timeout time.Duration
}

func WithRateLimit(inner Client, limiter *Limiter, timeout time.Duration) Client {
	return &RateLimitedClient{inner: inner, limiter: limiter, timeout: timeout}
}

func (c *RateLimitedClient) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	return c.inner.Generate(ctx, params)
}
