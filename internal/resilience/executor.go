package resilience

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

type Policy struct {
	Dependency string
	Operation  string
	Timeout    time.Duration
	Attempts   int
	Backoff    time.Duration
	MaxBackoff time.Duration
	RetryIf    func(error) bool
	Breaker    *CircuitBreaker
}

func Do[T any](ctx context.Context, policy Policy, operation func(context.Context) (T, error)) (T, error) {
	var zero T
	if ctx == nil {
		ctx = context.Background()
	}
	if policy.Attempts <= 0 {
		policy.Attempts = 1
	}
	dependencyRequestsTotal.WithLabelValues(policy.Dependency, policy.Operation).Inc()
	started := time.Now()
	defer func() {
		dependencyLatency.WithLabelValues(policy.Dependency, policy.Operation).Observe(time.Since(started).Seconds())
	}()

	var lastErr error
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			dependencyErrorsTotal.WithLabelValues(policy.Dependency, policy.Operation).Inc()
			return zero, err
		}

		var permit *Permit
		if policy.Breaker != nil {
			var err error
			permit, err = policy.Breaker.Acquire()
			if err != nil {
				dependencyErrorsTotal.WithLabelValues(policy.Dependency, policy.Operation).Inc()
				return zero, err
			}
		}

		attemptCtx := ctx
		cancel := func() {}
		if policy.Timeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, policy.Timeout)
		}
		value, err := operation(attemptCtx)
		attemptErr := attemptCtx.Err()
		cancel()
		if err == nil {
			if permit != nil {
				permit.Done(true)
			}
			return value, nil
		}
		lastErr = err
		if permit != nil {
			permit.Done(false)
		}
		if errors.Is(attemptErr, context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			dependencyTimeoutTotal.WithLabelValues(policy.Dependency, policy.Operation).Inc()
		}
		if attempt == policy.Attempts || policy.RetryIf == nil || !policy.RetryIf(err) {
			break
		}
		dependencyRetryTotal.WithLabelValues(policy.Dependency, policy.Operation).Inc()
		if err := waitBackoff(ctx, policy, attempt); err != nil {
			lastErr = err
			break
		}
	}

	dependencyErrorsTotal.WithLabelValues(policy.Dependency, policy.Operation).Inc()
	return zero, lastErr
}

func waitBackoff(ctx context.Context, policy Policy, failedAttempt int) error {
	if policy.Backoff <= 0 {
		return nil
	}
	delay := policy.Backoff << (failedAttempt - 1)
	if policy.MaxBackoff > 0 && delay > policy.MaxBackoff {
		delay = policy.MaxBackoff
	}
	jitter := time.Duration(rand.Int63n(int64(delay/2) + 1))
	timer := time.NewTimer(delay + jitter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
