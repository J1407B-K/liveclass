package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"liveclass/internal/rpc/chat/model"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type fakeOutboxStore struct {
	mu              sync.Mutex
	message         model.Message
	published       bool
	markFailures    int
	publishedSignal chan struct{}
}

func (s *fakeOutboxStore) ClaimNext(_ context.Context, owner string, now time.Time, lease time.Duration) (model.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := &s.message.Outbox
	eligible := state.Status == model.OutboxPending && !state.NextAttemptAt.After(now)
	if state.Status == model.OutboxPublishing && state.LeaseUntil != nil && !state.LeaseUntil.After(now) {
		eligible = true
	}
	if !eligible || s.published {
		return model.Message{}, mongo.ErrNoDocuments
	}
	until := now.Add(lease)
	state.Status, state.LeaseOwner, state.LeaseUntil = model.OutboxPublishing, owner, &until
	return s.message, nil
}

func (s *fakeOutboxStore) MarkPublished(_ context.Context, _, owner string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.markFailures > 0 {
		s.markFailures--
		return errors.New("synthetic mark failure")
	}
	if s.message.Outbox.LeaseOwner != owner {
		return errors.New("lease lost")
	}
	s.message.Outbox.Status = model.OutboxPublished
	s.message.Outbox.PublishedAt = &at
	s.published = true
	select {
	case s.publishedSignal <- struct{}{}:
	default:
	}
	return nil
}

func (s *fakeOutboxStore) MarkRetry(_ context.Context, _, owner, lastError string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.message.Outbox.LeaseOwner != owner {
		return errors.New("lease lost")
	}
	s.message.Outbox.Status = model.OutboxPending
	s.message.Outbox.Attempts++
	s.message.Outbox.LastError = lastError
	s.message.Outbox.NextAttemptAt = next
	s.message.Outbox.LeaseOwner = ""
	s.message.Outbox.LeaseUntil = nil
	return nil
}

func (s *fakeOutboxStore) CountPending(context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.published {
		return 0, nil
	}
	return 1, nil
}

type fakeKafkaWriter struct {
	mu       sync.Mutex
	writes   int
	failures int
}

func (w *fakeKafkaWriter) WriteMessages(context.Context, ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	if w.failures > 0 {
		w.failures--
		return errors.New("synthetic kafka failure")
	}
	return nil
}

func testOutboxConfig() OutboxConfig {
	return OutboxConfig{
		Workers: 1, PollInterval: 2 * time.Millisecond, LeaseDuration: 10 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond, RetryAttempts: 1,
		RetryBaseBackoff: 2 * time.Millisecond, RetryMaxBackoff: 10 * time.Millisecond,
	}
}

func newFakeOutboxStore(markFailures int) *fakeOutboxStore {
	now := time.Now().UTC()
	return &fakeOutboxStore{
		message: model.Message{
			MessageID: "m1", LessonID: 7, Content: "hello", CreatedAt: now,
			Outbox: model.OutboxState{Status: model.OutboxPending, NextAttemptAt: now},
		},
		markFailures: markFailures, publishedSignal: make(chan struct{}, 1),
	}
}

func waitPublished(t *testing.T, store *fakeOutboxStore) {
	t.Helper()
	select {
	case <-store.publishedSignal:
	case <-time.After(time.Second):
		t.Fatal("outbox record was not published")
	}
}

func stopRelay(t *testing.T, relay *OutboxRelay) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := relay.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOutboxRelayPublishesAndMarksRecord(t *testing.T) {
	store, writer := newFakeOutboxStore(0), &fakeKafkaWriter{}
	relay, err := NewOutboxRelay(writer, store, testOutboxConfig())
	if err != nil {
		t.Fatal(err)
	}
	relay.Start(context.Background())
	relay.Notify()
	waitPublished(t, store)
	stopRelay(t, relay)
	if writer.writes != 1 {
		t.Fatalf("Kafka writes = %d, want 1", writer.writes)
	}
}

func TestOutboxRelayRetriesKafkaFailureFromMongo(t *testing.T) {
	store, writer := newFakeOutboxStore(0), &fakeKafkaWriter{failures: 1}
	relay, err := NewOutboxRelay(writer, store, testOutboxConfig())
	if err != nil {
		t.Fatal(err)
	}
	relay.Start(context.Background())
	waitPublished(t, store)
	stopRelay(t, relay)
	if writer.writes < 2 || store.message.Outbox.Attempts != 1 {
		t.Fatalf("writes=%d attempts=%d, want durable retry", writer.writes, store.message.Outbox.Attempts)
	}
}

func TestOutboxRelayRepublishesAfterPublishedMarkFailure(t *testing.T) {
	store, writer := newFakeOutboxStore(1), &fakeKafkaWriter{}
	relay, err := NewOutboxRelay(writer, store, testOutboxConfig())
	if err != nil {
		t.Fatal(err)
	}
	relay.Start(context.Background())
	waitPublished(t, store)
	stopRelay(t, relay)
	if writer.writes < 2 {
		t.Fatalf("Kafka writes = %d, want duplicate after lease recovery", writer.writes)
	}
}
