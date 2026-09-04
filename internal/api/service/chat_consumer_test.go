package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"liveclass/internal/api/chatroom"
	global2 "liveclass/internal/api/global"

	"github.com/segmentio/kafka-go"
)

type fakeChatReader struct {
	fetch  func(context.Context) (kafka.Message, error)
	closed atomic.Bool
}

func TestConsumeChatBroadcastsThenCommitsEveryRecord(t *testing.T) {
	manager, err := chatroom.NewManager(chatroom.Config{
		SendQueueSize: 8, MessageDedupSize: 8, WriteWait: time.Second,
		PongWait: time.Minute, PingPeriod: 30 * time.Second, MaxMessageSize: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	previousManager := global2.ChatRooms
	global2.ChatRooms = manager
	defer func() { global2.ChatRooms = previousManager }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	payload, _ := json.Marshal(map[string]any{"message_id": "m1", "lesson_id": 1})
	var fetches, commits atomic.Int32
	reader := &fakeChatReader{fetch: func(context.Context) (kafka.Message, error) {
		if fetches.Add(1) <= 2 {
			return kafka.Message{Offset: int64(fetches.Load()), Value: payload}, nil
		}
		cancel()
		return kafka.Message{}, context.Canceled
	}}
	readerCommit := &countingChatReader{fakeChatReader: reader, commits: &commits}
	if consumeErr := consumeChat(ctx, readerCommit); !errors.Is(consumeErr, context.Canceled) {
		t.Fatalf("consumeChat error = %v", consumeErr)
	}
	if commits.Load() != 2 {
		t.Fatalf("commits=%d, want 2", commits.Load())
	}
}

type countingChatReader struct {
	*fakeChatReader
	commits *atomic.Int32
}

func (r *countingChatReader) CommitMessages(context.Context, ...kafka.Message) error {
	r.commits.Add(1)
	return nil
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
