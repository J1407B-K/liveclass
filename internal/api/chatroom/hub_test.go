package chatroom

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hertz-contrib/websocket"
)

type fakeSocket struct {
	readCh       chan fakeRead
	writes       chan []byte
	writeStarted chan struct{}
	writeGate    chan struct{}
	closed       chan struct{}
	closeOnce    sync.Once
}

func TestResumeBuffersLiveMessagesAndDeduplicatesHistory(t *testing.T) {
	manager, err := NewManager(testConfig(8))
	if err != nil {
		t.Fatal(err)
	}
	socket := newFakeSocket()
	client := manager.NewResumingClient(context.Background(), 21, socket, true)
	done := make(chan error, 1)
	go func() {
		done <- client.ServeWithBootstrap(func(context.Context, int, []byte) error { return nil }, func(_ context.Context, c *Client) error {
			if err := manager.BroadcastChat(21, "m2", map[string]string{"message_id": "m2", "content": "live"}); err != nil {
				return err
			}
			for _, message := range []map[string]string{
				{"message_id": "m1", "content": "missed"},
				{"message_id": "m2", "content": "live"},
			} {
				payload, _ := json.Marshal(message)
				if !c.EnqueueReplay(message["message_id"], payload) {
					return errors.New("replay enqueue failed")
				}
			}
			if !c.FinishResume() {
				return errors.New("resume flush failed")
			}
			return nil
		})
	}()

	for _, want := range []string{"m1", "m2"} {
		select {
		case payload := <-socket.writes:
			var message map[string]string
			if err := json.Unmarshal(payload, &message); err != nil || message["message_id"] != want {
				t.Fatalf("got %s, want message_id=%s", payload, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s", want)
		}
	}
	select {
	case duplicate := <-socket.writes:
		t.Fatalf("duplicate resume delivery: %s", duplicate)
	case <-time.After(20 * time.Millisecond):
	}
	client.Close("test complete")
	<-done
}

type fakeRead struct {
	typeID  int
	payload []byte
	err     error
}

func newFakeSocket() *fakeSocket {
	return &fakeSocket{
		readCh: make(chan fakeRead), writes: make(chan []byte, 16),
		writeStarted: make(chan struct{}, 16), closed: make(chan struct{}),
	}
}

func (f *fakeSocket) ReadMessage() (int, []byte, error) {
	select {
	case message := <-f.readCh:
		return message.typeID, message.payload, message.err
	case <-f.closed:
		return 0, nil, errors.New("closed")
	}
}

func (f *fakeSocket) WriteMessage(_ int, payload []byte) error {
	f.writeStarted <- struct{}{}
	if f.writeGate != nil {
		select {
		case <-f.writeGate:
		case <-f.closed:
			return errors.New("closed")
		}
	}
	f.writes <- append([]byte(nil), payload...)
	return nil
}

func (f *fakeSocket) WriteControl(_ int, _ []byte, _ time.Time) error { return nil }
func (f *fakeSocket) SetReadDeadline(time.Time) error                 { return nil }
func (f *fakeSocket) SetWriteDeadline(time.Time) error                { return nil }
func (f *fakeSocket) SetReadLimit(int64)                              {}
func (f *fakeSocket) SetPongHandler(func(string) error)               {}
func (f *fakeSocket) Close() error {
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}

func testConfig(queueSize int) Config {
	return Config{
		SendQueueSize: queueSize, MessageDedupSize: 16, WriteWait: time.Second,
		PongWait: time.Minute, PingPeriod: 30 * time.Second, MaxMessageSize: 4096,
	}
}

func TestBroadcastPreservesOrder(t *testing.T) {
	manager, err := NewManager(testConfig(4))
	if err != nil {
		t.Fatal(err)
	}
	socket := newFakeSocket()
	client := manager.NewClient(context.Background(), 7, socket)
	done := make(chan error, 1)
	go func() { done <- client.Serve(func(context.Context, int, []byte) error { return nil }) }()
	waitFor(t, func() bool { return manager.ConnectionCount(7) == 1 })

	for _, payload := range []string{"one", "two", "three"} {
		manager.Broadcast(7, []byte(payload))
	}
	for _, want := range []string{"one", "two", "three"} {
		select {
		case got := <-socket.writes:
			if string(got) != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}

	client.Close("test complete")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("client did not stop")
	}
}

func TestBroadcastChatDeduplicatesPerClient(t *testing.T) {
	manager, err := NewManager(testConfig(4))
	if err != nil {
		t.Fatal(err)
	}
	socket := newFakeSocket()
	client := manager.NewClient(context.Background(), 8, socket)
	done := make(chan error, 1)
	go func() { done <- client.Serve(func(context.Context, int, []byte) error { return nil }) }()
	waitFor(t, func() bool { return manager.ConnectionCount(8) == 1 })

	message := map[string]string{"message_id": "m1", "content": "once"}
	if err := manager.BroadcastChat(8, "m1", message); err != nil {
		t.Fatal(err)
	}
	if err := manager.BroadcastChat(8, "m1", message); err != nil {
		t.Fatal(err)
	}

	select {
	case payload := <-socket.writes:
		var got map[string]string
		if err := json.Unmarshal(payload, &got); err != nil || got["message_id"] != "m1" {
			t.Fatalf("unexpected delivery: %s", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first delivery")
	}
	select {
	case duplicate := <-socket.writes:
		t.Fatalf("duplicate live delivery: %s", duplicate)
	case <-time.After(20 * time.Millisecond):
	}

	client.Close("test complete")
	<-done
}

func TestSlowConsumerDoesNotBlockBroadcast(t *testing.T) {
	manager, err := NewManager(testConfig(1))
	if err != nil {
		t.Fatal(err)
	}
	socket := newFakeSocket()
	socket.writeGate = make(chan struct{})
	client := manager.NewClient(context.Background(), 9, socket)
	done := make(chan error, 1)
	go func() { done <- client.Serve(func(context.Context, int, []byte) error { return nil }) }()
	waitFor(t, func() bool { return manager.ConnectionCount(9) == 1 })

	manager.Broadcast(9, []byte("blocked write"))
	select {
	case <-socket.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	manager.Broadcast(9, []byte("queued"))
	started := time.Now()
	manager.Broadcast(9, []byte("must be dropped"))
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("broadcast blocked for %v", elapsed)
	}
	select {
	case <-client.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("slow consumer was not cancelled")
	}
	if got := manager.ConnectionCount(9); got != 0 {
		t.Fatalf("slow consumer remained in room: %d clients", got)
	}

	close(socket.writeGate)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow client did not stop")
	}
	waitFor(t, func() bool { return manager.ConnectionCount(9) == 0 })
}

func TestReadPumpUsesClientContext(t *testing.T) {
	manager, err := NewManager(testConfig(1))
	if err != nil {
		t.Fatal(err)
	}
	socket := newFakeSocket()
	client := manager.NewClient(context.Background(), 11, socket)
	handled := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- client.Serve(func(ctx context.Context, messageType int, payload []byte) error {
			if ctx != client.Context() || messageType != websocket.TextMessage || string(payload) != "hello" {
				return errors.New("unexpected handler input")
			}
			close(handled)
			return nil
		})
	}()
	waitFor(t, func() bool { return manager.ConnectionCount(11) == 1 })
	socket.readCh <- fakeRead{typeID: websocket.TextMessage, payload: []byte("hello")}
	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("message was not handled")
	}
	client.Close("test complete")
	<-done
}

func TestIdleClientIsDisconnectedAfterPongTimeout(t *testing.T) {
	cfg := testConfig(1)
	cfg.PongWait = 50 * time.Millisecond
	cfg.PingPeriod = 20 * time.Millisecond
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	socket := newFakeSocket()
	client := manager.NewClient(context.Background(), 12, socket)
	done := make(chan error, 1)
	go func() { done <- client.Serve(func(context.Context, int, []byte) error { return nil }) }()
	waitFor(t, func() bool { return manager.ConnectionCount(12) == 1 })

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("idle client was not disconnected")
	}
	if got := manager.ConnectionCount(12); got != 0 {
		t.Fatalf("idle client remained in room: %d clients", got)
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition was not met")
		}
		time.Sleep(time.Millisecond)
	}
}
