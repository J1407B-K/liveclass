package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"liveclass/internal/rpc/chat/model"

	"github.com/segmentio/kafka-go"
)

type fakeKafkaWriter struct {
	mu      sync.Mutex
	writes  []model.Message
	started chan struct{}
	gate    chan struct{}
}

func (f *fakeKafkaWriter) WriteMessages(ctx context.Context, messages ...kafka.Message) error {
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, raw := range messages {
		var message model.Message
		if err := json.Unmarshal(raw.Value, &message); err != nil {
			return err
		}
		f.writes = append(f.writes, message)
	}
	return nil
}

func dispatcherTestConfig() DispatcherConfig {
	return DispatcherConfig{
		QueueSize: 8, Workers: 2, EnqueueTimeout: 20 * time.Millisecond,
		WriteTimeout: time.Second, RetryAttempts: 1, RetryBaseBackoff: time.Millisecond,
	}
}

func TestKafkaDispatcherPreservesPerLessonOrder(t *testing.T) {
	writer := &fakeKafkaWriter{}
	dispatcher, err := NewKafkaDispatcher(writer, dispatcherTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	dispatcher.Start()
	for sequence := 1; sequence <= 4; sequence++ {
		for _, lessonID := range []int64{10, 11} {
			if err := dispatcher.Publish(context.Background(), model.Message{
				LessonID: lessonID, Content: string(rune('0' + sequence)),
			}); err != nil {
				t.Fatal(err)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dispatcher.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	writer.mu.Lock()
	defer writer.mu.Unlock()
	byLesson := map[int64]string{}
	for _, message := range writer.writes {
		byLesson[message.LessonID] += message.Content
	}
	if byLesson[10] != "1234" || byLesson[11] != "1234" {
		t.Fatalf("unexpected order: %#v", byLesson)
	}
}

func TestKafkaDispatcherRejectsFullQueue(t *testing.T) {
	writer := &fakeKafkaWriter{started: make(chan struct{}, 1), gate: make(chan struct{})}
	cfg := dispatcherTestConfig()
	cfg.QueueSize = 1
	cfg.Workers = 1
	dispatcher, err := NewKafkaDispatcher(writer, cfg)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher.Start()
	if err := dispatcher.Publish(context.Background(), model.Message{LessonID: 1, Content: "writing"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	if err := dispatcher.Publish(context.Background(), model.Message{LessonID: 1, Content: "queued"}); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	err = dispatcher.Publish(context.Background(), model.Message{LessonID: 1, Content: "rejected"})
	if !errors.Is(err, ErrPublishQueueFull) {
		t.Fatalf("got %v, want queue full", err)
	}
	if elapsed := time.Since(started); elapsed < cfg.EnqueueTimeout {
		t.Fatalf("returned before enqueue timeout: %v", elapsed)
	}
	close(writer.gate)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dispatcher.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
