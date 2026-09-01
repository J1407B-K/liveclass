package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRetriesBoundedAndSucceeds(t *testing.T) {
	var calls atomic.Int32
	got, err := Do(context.Background(), Policy{
		Dependency: "test-reranker", Operation: "score", Timeout: time.Second,
		Attempts: 3, Backoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond,
		RetryIf: func(error) bool { return true },
	}, func(context.Context) (string, error) {
		if calls.Add(1) < 3 {
			return "", errors.New("temporary")
		}
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d, want 3", calls.Load())
	}
}

func TestDoTimeoutIsBounded(t *testing.T) {
	started := time.Now()
	_, err := Do(context.Background(), Policy{
		Dependency: "test-embedding", Operation: "embed", Timeout: 20 * time.Millisecond, Attempts: 1,
	}, func(ctx context.Context) (struct{}, error) {
		<-ctx.Done()
		return struct{}{}, ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("timeout took %v", elapsed)
	}
}

func TestDoDoesNotRetryWhenPolicyRejects(t *testing.T) {
	var calls atomic.Int32
	wantErr := errors.New("permanent")
	_, err := Do(context.Background(), Policy{
		Dependency: "test-llm", Operation: "generate", Attempts: 3,
		RetryIf: func(error) bool { return false },
	}, func(context.Context) (struct{}, error) {
		calls.Add(1)
		return struct{}{}, wantErr
	})
	if !errors.Is(err, wantErr) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
