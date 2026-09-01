package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerClosedOpenHalfOpenClosed(t *testing.T) {
	now := time.Unix(100, 0)
	breaker, err := NewCircuitBreaker(BreakerConfig{
		Dependency: "test-qdrant", RollingWindow: time.Minute, MinimumRequests: 4,
		FailureThreshold: .5, OpenDuration: 10 * time.Second, HalfOpenProbes: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	breaker.now = func() time.Time { return now }

	for _, success := range []bool{true, false, true, false} {
		permit, acquireErr := breaker.Acquire()
		if acquireErr != nil {
			t.Fatal(acquireErr)
		}
		permit.Done(success)
	}
	if got := breaker.State(); got != StateOpen {
		t.Fatalf("state=%v, want open", got)
	}
	if _, err := breaker.Acquire(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("acquire error=%v, want circuit open", err)
	}

	now = now.Add(10 * time.Second)
	for i := 0; i < 2; i++ {
		permit, acquireErr := breaker.Acquire()
		if acquireErr != nil {
			t.Fatal(acquireErr)
		}
		permit.Done(true)
	}
	if got := breaker.State(); got != StateClosed {
		t.Fatalf("state=%v, want closed", got)
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	now := time.Unix(200, 0)
	breaker, err := NewCircuitBreaker(BreakerConfig{
		Dependency: "test-llm", RollingWindow: time.Minute, MinimumRequests: 1,
		FailureThreshold: 1, OpenDuration: time.Second, HalfOpenProbes: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	breaker.now = func() time.Time { return now }
	permit, _ := breaker.Acquire()
	permit.Done(false)
	now = now.Add(time.Second)
	permit, err = breaker.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	permit.Done(false)
	if breaker.State() != StateOpen {
		t.Fatal("half-open failure did not reopen breaker")
	}
}
