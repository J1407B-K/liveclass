package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
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

func (s *fakeOutboxStore) HasEarlierUnpublished(context.Context, int64, time.Time, string) (bool, error) {
	return false, nil
}

func (s *fakeOutboxStore) DeferForOrdering(_ context.Context, _, owner string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.message.Outbox.LeaseOwner != owner {
		return errors.New("lease lost")
	}
	s.message.Outbox.Status = model.OutboxPending
	s.message.Outbox.NextAttemptAt = next
	s.message.Outbox.LeaseOwner = ""
	s.message.Outbox.LeaseUntil = nil
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

type orderedOutboxStore struct {
	mu       sync.Mutex
	messages []model.Message
}

func (s *orderedOutboxStore) ClaimNext(_ context.Context, owner string, now time.Time, lease time.Duration) (model.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.messages {
		state := &s.messages[index].Outbox
		eligible := state.Status == model.OutboxPending && !state.NextAttemptAt.After(now)
		if state.Status == model.OutboxPublishing && state.LeaseUntil != nil && !state.LeaseUntil.After(now) {
			eligible = true
		}
		if !eligible {
			continue
		}
		until := now.Add(lease)
		state.Status, state.LeaseOwner, state.LeaseUntil = model.OutboxPublishing, owner, &until
		return s.messages[index], nil
	}
	return model.Message{}, mongo.ErrNoDocuments
}

func (s *orderedOutboxStore) HasEarlierUnpublished(_ context.Context, lessonID int64, createdAt time.Time, messageID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.messages {
		candidate := s.messages[index]
		if candidate.LessonID != lessonID || candidate.Outbox.Status == model.OutboxPublished {
			continue
		}
		if candidate.CreatedAt.Before(createdAt) || candidate.CreatedAt.Equal(createdAt) && candidate.MessageID < messageID {
			return true, nil
		}
	}
	return false, nil
}

func (s *orderedOutboxStore) DeferForOrdering(_ context.Context, messageID, owner string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	message, err := s.find(messageID)
	if err != nil || message.Outbox.LeaseOwner != owner {
		return errors.New("lease lost")
	}
	message.Outbox.Status = model.OutboxPending
	message.Outbox.NextAttemptAt = next
	message.Outbox.LeaseOwner = ""
	message.Outbox.LeaseUntil = nil
	return nil
}

func (s *orderedOutboxStore) MarkPublished(_ context.Context, messageID, owner string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	message, err := s.find(messageID)
	if err != nil || message.Outbox.LeaseOwner != owner {
		return errors.New("lease lost")
	}
	message.Outbox.Status = model.OutboxPublished
	message.Outbox.PublishedAt = &at
	return nil
}

func (s *orderedOutboxStore) MarkRetry(_ context.Context, messageID, owner, lastError string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	message, err := s.find(messageID)
	if err != nil || message.Outbox.LeaseOwner != owner {
		return errors.New("lease lost")
	}
	message.Outbox.Status = model.OutboxPending
	message.Outbox.NextAttemptAt = next
	message.Outbox.Attempts++
	message.Outbox.LastError = lastError
	message.Outbox.LeaseOwner = ""
	message.Outbox.LeaseUntil = nil
	return nil
}

func (s *orderedOutboxStore) CountPending(context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int64
	for _, message := range s.messages {
		if message.Outbox.Status != model.OutboxPublished {
			count++
		}
	}
	return count, nil
}

func (s *orderedOutboxStore) find(messageID string) (*model.Message, error) {
	for index := range s.messages {
		if s.messages[index].MessageID == messageID {
			return &s.messages[index], nil
		}
	}
	return nil, errors.New("message not found")
}

func (s *orderedOutboxStore) allPublished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, message := range s.messages {
		if message.Outbox.Status != model.OutboxPublished {
			return false
		}
	}
	return true
}

type orderingKafkaWriter struct {
	mu       sync.Mutex
	writes   []string
	aStarted chan struct{}
	cStarted chan struct{}
	releaseA chan struct{}
}

func (w *orderingKafkaWriter) WriteMessages(ctx context.Context, records ...kafka.Message) error {
	var message model.Message
	if err := json.Unmarshal(records[0].Value, &message); err != nil {
		return err
	}
	if message.MessageID == "a" {
		select {
		case w.aStarted <- struct{}{}:
		default:
		}
		select {
		case <-w.releaseA:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	w.mu.Lock()
	w.writes = append(w.writes, message.MessageID)
	w.mu.Unlock()
	if message.MessageID == "c" {
		select {
		case w.cStarted <- struct{}{}:
		default:
		}
	}
	return nil
}

func (w *orderingKafkaWriter) snapshot() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.writes...)
}

func TestOutboxRelayOrdersPerLessonButRunsLessonsInParallel(t *testing.T) {
	now := time.Now().UTC()
	store := &orderedOutboxStore{messages: []model.Message{
		{MessageID: "a", LessonID: 7, CreatedAt: now, Outbox: model.OutboxState{Status: model.OutboxPending, NextAttemptAt: now}},
		{MessageID: "b", LessonID: 7, CreatedAt: now.Add(time.Microsecond), Outbox: model.OutboxState{Status: model.OutboxPending, NextAttemptAt: now}},
		{MessageID: "c", LessonID: 8, CreatedAt: now.Add(2 * time.Microsecond), Outbox: model.OutboxState{Status: model.OutboxPending, NextAttemptAt: now}},
	}}
	sort.Slice(store.messages, func(i, j int) bool {
		if store.messages[i].CreatedAt.Equal(store.messages[j].CreatedAt) {
			return store.messages[i].MessageID < store.messages[j].MessageID
		}
		return store.messages[i].CreatedAt.Before(store.messages[j].CreatedAt)
	})
	writer := &orderingKafkaWriter{aStarted: make(chan struct{}, 1), cStarted: make(chan struct{}, 1), releaseA: make(chan struct{})}
	cfg := testOutboxConfig()
	cfg.Workers = 2
	cfg.WriteTimeout = time.Second
	relay, err := NewOutboxRelay(writer, store, cfg)
	if err != nil {
		t.Fatal(err)
	}
	relay.Start(context.Background())
	select {
	case <-writer.aStarted:
	case <-time.After(time.Second):
		t.Fatal("first lesson-7 message did not start")
	}
	select {
	case <-writer.cStarted:
	case <-time.After(time.Second):
		t.Fatal("lesson 8 was blocked by lesson 7")
	}
	for _, messageID := range writer.snapshot() {
		if messageID == "b" {
			t.Fatal("later lesson-7 message published before its predecessor")
		}
	}
	close(writer.releaseA)
	deadline := time.Now().Add(time.Second)
	for !store.allPublished() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	stopRelay(t, relay)
	writes := writer.snapshot()
	position := map[string]int{}
	for index, messageID := range writes {
		position[messageID] = index
	}
	if len(writes) != 3 || position["a"] >= position["b"] {
		t.Fatalf("Kafka writes = %v, want a before b and all three messages", writes)
	}
}

type stressOrderingWriter struct {
	mu         sync.Mutex
	last       map[int64]int
	writes     int
	violations []string
	done       chan struct{}
	total      int
}

func (w *stressOrderingWriter) WriteMessages(_ context.Context, records ...kafka.Message) error {
	var message model.Message
	if err := json.Unmarshal(records[0].Value, &message); err != nil {
		return err
	}
	sequence, err := strconv.Atoi(message.Content)
	if err != nil {
		return err
	}
	// Vary completion time so the test exercises concurrent writers instead of
	// accidentally passing because every Kafka call takes the same duration.
	time.Sleep(time.Duration(sequence%7) * time.Microsecond)
	w.mu.Lock()
	defer w.mu.Unlock()
	if previous, exists := w.last[message.LessonID]; exists && sequence != previous+1 {
		w.violations = append(w.violations, fmt.Sprintf("lesson=%d previous=%d current=%d", message.LessonID, previous, sequence))
	}
	w.last[message.LessonID] = sequence
	w.writes++
	if w.writes == w.total {
		close(w.done)
	}
	return nil
}

func TestOutboxRelayStressPreservesOrderAcrossWorkers(t *testing.T) {
	const lessons, messagesPerLesson, workers = 10, 100, 8
	now := time.Now().UTC()
	messages := make([]model.Message, 0, lessons*messagesPerLesson)
	for sequence := 0; sequence < messagesPerLesson; sequence++ {
		for lesson := 0; lesson < lessons; lesson++ {
			messages = append(messages, model.Message{
				MessageID: fmt.Sprintf("%02d-%03d", lesson, sequence), LessonID: int64(lesson + 1),
				Content: strconv.Itoa(sequence), CreatedAt: now.Add(time.Duration(sequence*lessons+lesson) * time.Microsecond),
				Outbox: model.OutboxState{Status: model.OutboxPending, NextAttemptAt: now},
			})
		}
	}
	store := &orderedOutboxStore{messages: messages}
	writer := &stressOrderingWriter{last: make(map[int64]int), done: make(chan struct{}), total: len(messages)}
	cfg := testOutboxConfig()
	cfg.Workers = workers
	cfg.PollInterval = 100 * time.Microsecond
	relay, err := NewOutboxRelay(writer, store, cfg)
	if err != nil {
		t.Fatal(err)
	}
	relay.Start(context.Background())
	select {
	case <-writer.done:
	case <-time.After(10 * time.Second):
		stopRelay(t, relay)
		t.Fatalf("timed out after %d/%d writes", writer.writes, writer.total)
	}
	stopRelay(t, relay)
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if len(writer.violations) != 0 {
		t.Fatalf("per-lesson ordering violations: %v", writer.violations)
	}
	t.Logf("verified %d messages across %d lessons with %d relay workers: 0 ordering violations", writer.writes, lessons, workers)
}
