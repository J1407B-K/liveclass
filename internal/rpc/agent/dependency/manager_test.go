package dependency

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"liveclass/internal/rpc/agent/config"
)

func TestConfiguredPoliciesRetryReadButNotWrite(t *testing.T) {
	if err := Configure(testConfig()); err != nil {
		t.Fatal(err)
	}

	var readCalls atomic.Int32
	value, err := Do(context.Background(), PostgresRead, "test_read", func(context.Context) (string, error) {
		if readCalls.Add(1) == 1 {
			return "", context.DeadlineExceeded
		}
		return "ok", nil
	})
	if err != nil || value != "ok" || readCalls.Load() != 2 {
		t.Fatalf("read value=%q err=%v calls=%d", value, err, readCalls.Load())
	}

	var writeCalls atomic.Int32
	want := context.DeadlineExceeded
	_, err = Do(context.Background(), PostgresWrite, "test_write", func(context.Context) (struct{}, error) {
		writeCalls.Add(1)
		return struct{}{}, want
	})
	if !errors.Is(err, want) || writeCalls.Load() != 1 {
		t.Fatalf("write err=%v calls=%d, want one attempt", err, writeCalls.Load())
	}
}

func TestTransientClassification(t *testing.T) {
	if !IsTransient(context.DeadlineExceeded) {
		t.Fatal("deadline should be transient")
	}
	if !IsTransient(&HTTPStatusError{StatusCode: 503}) {
		t.Fatal("HTTP 503 should be transient")
	}
	if IsTransient(&HTTPStatusError{StatusCode: 400}) {
		t.Fatal("HTTP 400 should be permanent")
	}
	if IsTransient(errors.New("validation failed")) {
		t.Fatal("unknown error must not be retried")
	}
}

func testConfig() config.ResilienceConfig {
	read := config.DependencyPolicyConfig{Timeout: 50 * time.Millisecond, Attempts: 2, Backoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}
	write := config.DependencyPolicyConfig{Timeout: 50 * time.Millisecond, Attempts: 1}
	return config.ResilienceConfig{
		MainLLM: write, AdvisorLLM: write, ProfileLLM: write,
		Embedding: read, Qdrant: read, Elasticsearch: read, Reranker: read, WebSearch: read,
		PostgresRead: read, PostgresWrite: write, InternalRPC: read,
	}
}
