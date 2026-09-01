package model

import (
	"testing"
	"time"
)

func TestRequestVideoKeyframeSuppressesWhenSourceIsNotReady(t *testing.T) {
	b := &BroadcastBundle{}
	forwarded, err := b.RequestVideoKeyframe(time.Now(), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("RequestVideoKeyframe() error = %v", err)
	}
	if forwarded {
		t.Fatal("request without a publisher video source must not be forwarded")
	}
}

func TestRequestVideoKeyframeSuppressesWithinInterval(t *testing.T) {
	now := time.Now()
	b := &BroadcastBundle{VideoSSRC: 42, lastPLI: now}
	forwarded, err := b.RequestVideoKeyframe(now.Add(100*time.Millisecond), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("RequestVideoKeyframe() error = %v", err)
	}
	if forwarded {
		t.Fatal("request inside aggregation interval must be suppressed")
	}
}
