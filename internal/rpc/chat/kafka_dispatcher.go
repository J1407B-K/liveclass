package main

import (
	"context"
	"encoding/json"
	"errors"
	"liveclass/internal/rpc/chat/model"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	kafkaDispatchQueueSize = 1024
	kafkaDispatchWorkers   = 4
	kafkaDispatchRetries   = 3
	kafkaWriteTimeout      = 3 * time.Second
)

type KafkaDispatcher struct {
	writer *kafka.Writer
	queue  chan model.Message
	wg     sync.WaitGroup
}

func NewKafkaDispatcher(writer *kafka.Writer) *KafkaDispatcher {
	return &KafkaDispatcher{
		writer: writer,
		queue:  make(chan model.Message, kafkaDispatchQueueSize),
	}
}

func (d *KafkaDispatcher) Start() {
	if d == nil || d.writer == nil {
		return
	}
	for i := 0; i < kafkaDispatchWorkers; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}
}

func (d *KafkaDispatcher) Stop() {
	if d == nil {
		return
	}
	close(d.queue)
	d.wg.Wait()
}

func (d *KafkaDispatcher) Publish(ctx context.Context, msg model.Message) error {
	if d == nil || d.writer == nil {
		return errors.New("nil kafka dispatcher")
	}

	select {
	case d.queue <- msg:
		return nil
	default:
		log.Printf("[KafkaDispatcher] queue full, fallback to direct write: lesson=%d sender=%d", msg.LessonID, msg.Sender)
		return d.writeWithRetry(ctx, msg)
	}
}

func (d *KafkaDispatcher) worker(id int) {
	defer d.wg.Done()

	for msg := range d.queue {
		if err := d.writeWithRetry(context.Background(), msg); err != nil {
			log.Printf("[KafkaDispatcher] write failed after retries: worker=%d lesson=%d sender=%d err=%v",
				id, msg.LessonID, msg.Sender, err)
		}
	}
}

func (d *KafkaDispatcher) writeWithRetry(ctx context.Context, msg model.Message) error {
	var lastErr error
	for attempt := 1; attempt <= kafkaDispatchRetries; attempt++ {
		if err := d.writeOnce(ctx, msg); err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
			continue
		}
		return nil
	}
	return lastErr
}

func (d *KafkaDispatcher) writeOnce(ctx context.Context, msg model.Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, kafkaWriteTimeout)
	defer cancel()

	return d.writer.WriteMessages(writeCtx, kafka.Message{
		Key:   []byte(strconv.FormatInt(msg.LessonID, 10)),
		Value: msgBytes,
	})
}
