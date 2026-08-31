package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/internal/rpc/chat/model"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	ErrDispatcherStopped = errors.New("kafka dispatcher stopped")
	ErrPublishQueueFull  = errors.New("kafka dispatcher queue full")
)

type DispatcherConfig struct {
	QueueSize        int
	Workers          int
	EnqueueTimeout   time.Duration
	WriteTimeout     time.Duration
	RetryAttempts    int
	RetryBaseBackoff time.Duration
}

type KafkaWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

type KafkaDispatcher struct {
	writer KafkaWriter
	cfg    DispatcherConfig
	queues []chan model.Message
	wg     sync.WaitGroup

	mu       sync.RWMutex
	started  bool
	stopped  bool
	stopDone chan struct{}
}

func NewKafkaDispatcher(writer KafkaWriter, cfg DispatcherConfig) (*KafkaDispatcher, error) {
	if writer == nil {
		return nil, errors.New("nil kafka writer")
	}
	if cfg.QueueSize <= 0 || cfg.Workers <= 0 || cfg.QueueSize < cfg.Workers {
		return nil, errors.New("kafka dispatcher queue size must be >= workers and both must be positive")
	}
	if cfg.EnqueueTimeout <= 0 || cfg.WriteTimeout <= 0 || cfg.RetryAttempts <= 0 || cfg.RetryBaseBackoff <= 0 {
		return nil, errors.New("kafka dispatcher timeouts, attempts and backoff must be positive")
	}
	queues := make([]chan model.Message, cfg.Workers)
	baseCapacity := cfg.QueueSize / cfg.Workers
	remainder := cfg.QueueSize % cfg.Workers
	for i := range queues {
		capacity := baseCapacity
		if i < remainder {
			capacity++
		}
		queues[i] = make(chan model.Message, capacity)
	}
	return &KafkaDispatcher{writer: writer, cfg: cfg, queues: queues, stopDone: make(chan struct{})}, nil
}

func (d *KafkaDispatcher) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started || d.stopped {
		return
	}
	d.started = true
	for i, queue := range d.queues {
		d.wg.Add(1)
		go d.worker(i, queue)
	}
}

func (d *KafkaDispatcher) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.stopped {
		d.stopped = true
		for _, queue := range d.queues {
			close(queue)
		}
		go func() {
			d.wg.Wait()
			close(d.stopDone)
		}()
	}
	d.mu.Unlock()
	select {
	case <-d.stopDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *KafkaDispatcher) Publish(ctx context.Context, msg model.Message) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.started || d.stopped {
		return ErrDispatcherStopped
	}

	queue := d.queues[uint64(msg.LessonID)%uint64(len(d.queues))]
	timer := time.NewTimer(d.cfg.EnqueueTimeout)
	defer timer.Stop()
	select {
	case queue <- msg:
		chatPublishQueueDepth.Inc()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		chatPublishQueueFullTotal.Inc()
		return ErrPublishQueueFull
	}
}

func (d *KafkaDispatcher) worker(id int, queue <-chan model.Message) {
	defer d.wg.Done()
	for msg := range queue {
		chatPublishQueueDepth.Dec()
		if err := d.writeWithRetry(context.Background(), msg); err != nil {
			log.Printf("[KafkaDispatcher] write failed after retries: worker=%d lesson=%d sender=%d message_id=%s err=%v",
				id, msg.LessonID, msg.SenderID, msg.MessageID, err)
		}
	}
}

func (d *KafkaDispatcher) writeWithRetry(ctx context.Context, msg model.Message) error {
	var lastErr error
	for attempt := 1; attempt <= d.cfg.RetryAttempts; attempt++ {
		if err := d.writeOnce(ctx, msg); err != nil {
			lastErr = err
			if attempt == d.cfg.RetryAttempts {
				break
			}
			base := d.cfg.RetryBaseBackoff << (attempt - 1)
			jitter := time.Duration(rand.Int63n(int64(base/2) + 1))
			timer := time.NewTimer(base + jitter)
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
	return fmt.Errorf("publish failed after %d attempts: %w", d.cfg.RetryAttempts, lastErr)
}

func (d *KafkaDispatcher) writeOnce(ctx context.Context, msg model.Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, d.cfg.WriteTimeout)
	defer cancel()

	started := time.Now()
	err = d.writer.WriteMessages(writeCtx, kafka.Message{
		Key:   []byte(strconv.FormatInt(msg.LessonID, 10)),
		Value: msgBytes,
	})
	chatPublishLatency.Observe(time.Since(started).Seconds())
	if err != nil {
		chatPublishErrorsTotal.Inc()
	}
	return err
}
