package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeChatReader struct {
	fetch  func(context.Context) (kafka.Message, error)
	closed atomic.Bool
}

func (f *fakeChatReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return f.fetch(ctx)
}

func (f *fakeChatReader) CommitMessages(context.Context, ...kafka.Message) error { return nil }
func (f *fakeChatReader) Close() error {
	f.closed.Store(true)
	return nil
}

func TestRunChatConsumerReconnectsAndShutsDown(t *testing.T) {
	first := &fakeChatReader{fetch: func(context.Context) (kafka.Message, error) {
		return kafka.Message{}, errors.New("broker unavailable")
	}}
	second := &fakeChatReader{fetch: func(ctx context.Context) (kafka.Message, error) {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}}
	var factories atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunChatConsumer(ctx, func() ChatReader {
			if factories.Add(1) == 1 {
				return first
			}
			return second
		})
	}()

	deadline := time.Now().Add(time.Second)
	for factories.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if factories.Load() < 2 {
		t.Fatal("consumer did not reconnect")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shutdown returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("consumer did not shut down")
	}
	if !first.closed.Load() || !second.closed.Load() {
		t.Fatal("readers were not closed")
	}
}
