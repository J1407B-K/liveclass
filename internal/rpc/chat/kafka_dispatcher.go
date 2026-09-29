package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/internal/rpc/chat/dao"
	"liveclass/internal/rpc/chat/model"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type OutboxConfig struct {
	Workers               int
	PollInterval          time.Duration
	OrderingRetryInterval time.Duration
	LeaseDuration         time.Duration
	WriteTimeout          time.Duration
	RetryAttempts         int
	RetryBaseBackoff      time.Duration
	RetryMaxBackoff       time.Duration
}

type KafkaWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

type outboxStore interface {
	ClaimNext(context.Context, string, time.Time, time.Duration) (model.Message, error)
	HasEarlierUnpublished(context.Context, int64, time.Time, string) (bool, error)
	DeferForOrdering(context.Context, string, string, time.Time) error
	MarkPublished(context.Context, string, string, time.Time) error
	MarkRetry(context.Context, string, string, string, time.Time) error
	CountPending(context.Context) (int64, error)
}

type affinityOutboxStore interface {
	ClaimNextForWorker(context.Context, string, time.Time, time.Duration, int, int) (model.Message, error)
}

type mongoOutboxStore struct{ collection *mongo.Collection }

func (s mongoOutboxStore) ClaimNext(ctx context.Context, owner string, now time.Time, lease time.Duration) (model.Message, error) {
	return dao.ClaimNextOutbox(ctx, s.collection, owner, now, lease)
}

func (s mongoOutboxStore) ClaimNextForWorker(ctx context.Context, owner string, now time.Time, lease time.Duration, workerID, workerCount int) (model.Message, error) {
	return dao.ClaimNextOutboxForWorker(ctx, s.collection, owner, now, lease, workerID, workerCount)
}

func (s mongoOutboxStore) HasEarlierUnpublished(ctx context.Context, lessonID int64, createdAt time.Time, messageID string) (bool, error) {
	return dao.HasEarlierUnpublishedOutbox(ctx, s.collection, lessonID, createdAt, messageID)
}

func (s mongoOutboxStore) DeferForOrdering(ctx context.Context, messageID, owner string, next time.Time) error {
	return dao.DeferOutboxForOrdering(ctx, s.collection, messageID, owner, next)
}

func (s mongoOutboxStore) MarkPublished(ctx context.Context, messageID, owner string, at time.Time) error {
	return dao.MarkOutboxPublished(ctx, s.collection, messageID, owner, at)
}

func (s mongoOutboxStore) MarkRetry(ctx context.Context, messageID, owner, lastError string, next time.Time) error {
	return dao.MarkOutboxRetry(ctx, s.collection, messageID, owner, lastError, next)
}

func (s mongoOutboxStore) CountPending(ctx context.Context) (int64, error) {
	return dao.CountPendingOutbox(ctx, s.collection)
}

// OutboxRelay drains durable pending records from Mongo into Kafka. Notify is
// only a low-latency hint; polling and expiring leases are the recovery path.
type OutboxRelay struct {
	writer KafkaWriter
	store  outboxStore
	cfg    OutboxConfig
	wake   chan struct{}
	// Workers are assigned a stable lesson affinity at claim time. The Mongo
	// ordering check below remains the cross-instance safety fence.
	lessonGates sync.Map // map[int64]chan struct{}

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewOutboxRelay(writer KafkaWriter, store outboxStore, cfg OutboxConfig) (*OutboxRelay, error) {
	if writer == nil || store == nil {
		return nil, errors.New("outbox relay requires writer and store")
	}
	if cfg.Workers <= 0 || cfg.PollInterval <= 0 || cfg.LeaseDuration <= 0 || cfg.WriteTimeout <= 0 ||
		cfg.RetryAttempts <= 0 || cfg.RetryBaseBackoff <= 0 || cfg.RetryMaxBackoff < cfg.RetryBaseBackoff {
		return nil, errors.New("invalid outbox relay configuration")
	}
	if cfg.OrderingRetryInterval <= 0 {
		cfg.OrderingRetryInterval = 10 * time.Millisecond
	}
	return &OutboxRelay{writer: writer, store: store, cfg: cfg, wake: make(chan struct{}, 1)}, nil
}

func (r *OutboxRelay) Start(parent context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	r.started = true
	instanceID := uuid.NewString()
	for worker := 0; worker < r.cfg.Workers; worker++ {
		r.wg.Add(1)
		go r.runWorker(ctx, fmt.Sprintf("%s-%d", instanceID, worker), worker, worker == 0)
	}
}

func (r *OutboxRelay) Notify() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *OutboxRelay) Stop(ctx context.Context) error {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
	}
	r.mu.Unlock()
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *OutboxRelay) runWorker(ctx context.Context, owner string, workerID int, reportDepth bool) {
	defer r.wg.Done()
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()
	r.drain(ctx, owner, workerID)
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.wake:
			r.drain(ctx, owner, workerID)
		case <-ticker.C:
			if reportDepth {
				depthCtx, cancel := context.WithTimeout(ctx, r.cfg.WriteTimeout)
				if count, err := r.store.CountPending(depthCtx); err == nil {
					chatOutboxPending.Set(float64(count))
				}
				cancel()
			}
			r.drain(ctx, owner, workerID)
		}
	}
}

func (r *OutboxRelay) drain(ctx context.Context, owner string, workerID int) {
	for ctx.Err() == nil {
		claimCtx, cancel := context.WithTimeout(ctx, r.cfg.WriteTimeout)
		message, err := r.claimNext(claimCtx, owner, time.Now().UTC(), workerID)
		cancel()
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		if err != nil {
			log.Printf("[ChatOutbox] claim failed: %v", err)
			return
		}
		chatOutboxClaimedTotal.Inc()
		gate := r.lessonGate(message.LessonID)
		select {
		case gate <- struct{}{}:
		case <-ctx.Done():
			return
		default:
			// Another local worker is already publishing this lesson. Release
			// the claim quickly so it can serve a different lesson; the short
			// ordering retry interval avoids waiting for the coarse poll tick.
			next := time.Now().UTC().Add(r.cfg.OrderingRetryInterval)
			deferCtx, deferCancel := context.WithTimeout(context.Background(), r.cfg.WriteTimeout)
			deferErr := r.store.DeferForOrdering(deferCtx, message.MessageID, owner, next)
			deferCancel()
			if deferErr != nil {
				log.Printf("[ChatOutbox] local ordering defer failed: message_id=%s err=%v", message.MessageID, deferErr)
			}
			chatOutboxOrderingDeferredTotal.Inc()
			continue
		}

		orderCtx, orderCancel := context.WithTimeout(ctx, r.cfg.WriteTimeout)
		chatOutboxOrderingChecksTotal.Inc()
		blocked, orderErr := r.store.HasEarlierUnpublished(orderCtx, message.LessonID, message.CreatedAt, message.MessageID)
		orderCancel()
		if orderErr != nil || blocked {
			<-gate
			// Ordering waits are distinct from an empty-queue poll. A long poll
			// interval here turns a hot lesson into roughly one message per poll.
			next := time.Now().UTC().Add(r.cfg.OrderingRetryInterval)
			deferCtx, deferCancel := context.WithTimeout(context.Background(), r.cfg.WriteTimeout)
			deferErr := r.store.DeferForOrdering(deferCtx, message.MessageID, owner, next)
			deferCancel()
			if orderErr != nil {
				log.Printf("[ChatOutbox] ordering check failed: message_id=%s err=%v", message.MessageID, orderErr)
			}
			if deferErr != nil {
				log.Printf("[ChatOutbox] ordering defer failed: message_id=%s err=%v", message.MessageID, deferErr)
			}
			chatOutboxOrderingDeferredTotal.Inc()
			continue
		}
		if err := r.writeWithRetry(ctx, message); err != nil {
			<-gate
			next := time.Now().UTC().Add(r.retryDelay(message.Outbox.Attempts + 1))
			markCtx, markCancel := context.WithTimeout(context.Background(), r.cfg.WriteTimeout)
			markErr := r.store.MarkRetry(markCtx, message.MessageID, owner, truncateError(err), next)
			markCancel()
			if markErr != nil {
				log.Printf("[ChatOutbox] release failed: message_id=%s err=%v", message.MessageID, markErr)
			}
			chatOutboxRetryTotal.Inc()
			continue
		}

		markCtx, markCancel := context.WithTimeout(context.Background(), r.cfg.WriteTimeout)
		err = r.store.MarkPublished(markCtx, message.MessageID, owner, time.Now().UTC())
		markCancel()
		if err != nil {
			<-gate
			// Kafka may already contain the record. The expired lease deliberately
			// causes a repeat; API consumers deduplicate by message_id.
			log.Printf("[ChatOutbox] published but mark failed: message_id=%s err=%v", message.MessageID, err)
			continue
		}
		chatOutboxPublishedTotal.Inc()
		<-gate
		// A successful predecessor may make a deferred message immediately
		// eligible. Wake a worker instead of waiting for the coarse poll ticker.
		r.Notify()
	}
}

func (r *OutboxRelay) claimNext(ctx context.Context, owner string, now time.Time, workerID int) (model.Message, error) {
	if store, ok := r.store.(affinityOutboxStore); ok {
		return store.ClaimNextForWorker(ctx, owner, now, r.cfg.LeaseDuration, workerID, r.cfg.Workers)
	}
	return r.store.ClaimNext(ctx, owner, now, r.cfg.LeaseDuration)
}

func (r *OutboxRelay) lessonGate(lessonID int64) chan struct{} {
	gate := make(chan struct{}, 1)
	actual, _ := r.lessonGates.LoadOrStore(lessonID, gate)
	return actual.(chan struct{})
}

func (r *OutboxRelay) writeWithRetry(ctx context.Context, message model.Message) error {
	var lastErr error
	for attempt := 1; attempt <= r.cfg.RetryAttempts; attempt++ {
		if err := r.writeOnce(ctx, message); err != nil {
			lastErr = err
			if attempt == r.cfg.RetryAttempts {
				break
			}
			delay := r.cfg.RetryBaseBackoff << (attempt - 1)
			if delay > r.cfg.RetryMaxBackoff {
				delay = r.cfg.RetryMaxBackoff
			}
			timer := time.NewTimer(delay + time.Duration(rand.Int63n(int64(delay/2)+1)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("publish failed after %d attempts: %w", r.cfg.RetryAttempts, lastErr)
}

func (r *OutboxRelay) writeOnce(ctx context.Context, message model.Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()
	started := time.Now()
	err = r.writer.WriteMessages(writeCtx, kafka.Message{
		Key: []byte(strconv.FormatInt(message.LessonID, 10)), Value: payload,
	})
	chatPublishLatency.Observe(time.Since(started).Seconds())
	if err != nil {
		chatPublishErrorsTotal.Inc()
	}
	return err
}

func (r *OutboxRelay) retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := r.cfg.RetryBaseBackoff
	for i := 1; i < attempt && delay < r.cfg.RetryMaxBackoff; i++ {
		delay *= 2
		if delay > r.cfg.RetryMaxBackoff {
			delay = r.cfg.RetryMaxBackoff
		}
	}
	return delay + time.Duration(rand.Int63n(int64(delay/2)+1))
}

func truncateError(err error) string {
	const maxLength = 512
	value := strings.TrimSpace(err.Error())
	if len(value) > maxLength {
		return value[:maxLength]
	}
	return value
}
