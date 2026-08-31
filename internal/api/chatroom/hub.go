package chatroom

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"liveclass/internal/api/observability"

	"github.com/hertz-contrib/websocket"
)

const slowConsumerCloseReason = "slow consumer"

type Config struct {
	SendQueueSize  int
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
}

func (c Config) validate() error {
	if c.SendQueueSize <= 0 {
		return errors.New("chat send queue size must be positive")
	}
	if c.WriteWait <= 0 || c.PongWait <= 0 || c.PingPeriod <= 0 {
		return errors.New("chat websocket timeouts must be positive")
	}
	if c.PingPeriod >= c.PongWait {
		return errors.New("chat ping period must be shorter than pong wait")
	}
	if c.MaxMessageSize <= 0 {
		return errors.New("chat max message size must be positive")
	}
	return nil
}

type Socket interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	SetReadLimit(limit int64)
	SetPongHandler(func(string) error)
	Close() error
}

type Manager struct {
	mu    sync.RWMutex
	rooms map[int64]*Room
	cfg   Config
}

type Room struct {
	lessonID int64
	mu       sync.RWMutex
	clients  map[*Client]struct{}
}

type Client struct {
	manager   *Manager
	lessonID  int64
	conn      Socket
	send      chan []byte
	alive     chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
	stateMu   sync.RWMutex
	closed    bool
	reason    string
}

type MessageHandler func(ctx context.Context, messageType int, payload []byte) error

func NewManager(cfg Config) (*Manager, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Manager{rooms: make(map[int64]*Room), cfg: cfg}, nil
}

func (m *Manager) NewClient(parent context.Context, lessonID int64, conn Socket) *Client {
	ctx, cancel := context.WithCancel(parent)
	return &Client{
		manager: m, lessonID: lessonID, conn: conn,
		send: make(chan []byte, m.cfg.SendQueueSize), alive: make(chan struct{}, 1), ctx: ctx, cancel: cancel,
	}
}

func (m *Manager) BroadcastJSON(lessonID int64, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	m.Broadcast(lessonID, payload)
	return nil
}

func (m *Manager) Broadcast(lessonID int64, payload []byte) {
	started := time.Now()
	defer func() { observability.ChatFanoutLatency.Observe(time.Since(started).Seconds()) }()

	m.mu.RLock()
	room := m.rooms[lessonID]
	m.mu.RUnlock()
	if room == nil {
		return
	}

	room.mu.RLock()
	clients := make([]*Client, 0, len(room.clients))
	for client := range room.clients {
		clients = append(clients, client)
	}
	room.mu.RUnlock()

	for _, client := range clients {
		if !client.Enqueue(payload) {
			if client.close(slowConsumerCloseReason) {
				observability.DroppedMessagesTotal.Inc()
				observability.SlowConsumerTotal.Inc()
				m.unregister(client)
			}
		}
	}
}

func (m *Manager) ConnectionCount(lessonID int64) int {
	m.mu.RLock()
	room := m.rooms[lessonID]
	m.mu.RUnlock()
	if room == nil {
		return 0
	}
	room.mu.RLock()
	count := len(room.clients)
	room.mu.RUnlock()
	return count
}

func (m *Manager) register(client *Client) {
	m.mu.Lock()
	room := m.rooms[client.lessonID]
	if room == nil {
		room = &Room{lessonID: client.lessonID, clients: make(map[*Client]struct{})}
		m.rooms[client.lessonID] = room
	}
	room.mu.Lock()
	room.clients[client] = struct{}{}
	room.mu.Unlock()
	m.mu.Unlock()
	observability.ActiveWebSocketConnections.Inc()
	observability.RoomConnections.WithLabelValues(lessonLabel(client.lessonID)).Inc()
}

func (m *Manager) unregister(client *Client) {
	m.mu.RLock()
	room := m.rooms[client.lessonID]
	m.mu.RUnlock()
	if room == nil {
		return
	}

	removed := false
	room.mu.Lock()
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
		removed = true
	}
	empty := len(room.clients) == 0
	room.mu.Unlock()
	if !removed {
		return
	}

	observability.ActiveWebSocketConnections.Dec()
	observability.RoomConnections.WithLabelValues(lessonLabel(client.lessonID)).Dec()
	if empty {
		m.mu.Lock()
		if current := m.rooms[client.lessonID]; current == room {
			room.mu.RLock()
			stillEmpty := len(room.clients) == 0
			room.mu.RUnlock()
			if stillEmpty {
				delete(m.rooms, client.lessonID)
				observability.RoomConnections.DeleteLabelValues(lessonLabel(client.lessonID))
			}
		}
		m.mu.Unlock()
	}
}

func (c *Client) Context() context.Context { return c.ctx }

func (c *Client) Enqueue(payload []byte) bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- payload:
		observability.ChatQueueDepth.Inc()
		return true
	default:
		return false
	}
}

func (c *Client) Close(reason string) {
	c.close(reason)
}

func (c *Client) close(reason string) bool {
	closedNow := false
	c.closeOnce.Do(func() {
		c.stateMu.Lock()
		c.reason = reason
		c.closed = true
		c.cancel()
		c.stateMu.Unlock()
		closedNow = true
	})
	return closedNow
}

func (c *Client) Serve(handler MessageHandler) error {
	c.manager.register(c)
	defer c.manager.unregister(c)

	c.conn.SetReadLimit(c.manager.cfg.MaxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.markAlive()
		return nil
	})

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		c.writePump()
	}()

	err := c.readPump(handler)
	c.Close("connection closed")
	<-writerDone
	return err
}

func (c *Client) readPump(handler MessageHandler) error {
	for {
		messageType, payload, err := c.conn.ReadMessage()
		if err != nil {
			return err
		}
		c.markAlive()
		if err := handler(c.ctx, messageType, payload); err != nil {
			return err
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(c.manager.cfg.PingPeriod)
	defer ticker.Stop()
	idleTimer := time.NewTimer(c.manager.cfg.PongWait)
	defer idleTimer.Stop()
	defer func() {
		if queued := len(c.send); queued > 0 {
			observability.ChatQueueDepth.Sub(float64(queued))
		}
		_ = c.conn.Close()
	}()

	for {
		select {
		case payload := <-c.send:
			observability.ChatQueueDepth.Dec()
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.manager.cfg.WriteWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				observability.WebSocketWriteErrorsTotal.Inc()
				c.Close("write error")
				return
			}
		case <-ticker.C:
			deadline := time.Now().Add(c.manager.cfg.WriteWait)
			if err := c.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				observability.WebSocketWriteErrorsTotal.Inc()
				c.Close("ping error")
				return
			}
		case <-c.alive:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(c.manager.cfg.PongWait)
		case <-idleTimer.C:
			c.Close("pong timeout")
			return
		case <-c.ctx.Done():
			c.stateMu.RLock()
			reason := c.reason
			c.stateMu.RUnlock()
			if reason == "" {
				reason = "server shutdown"
			}
			_ = c.conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
				time.Now().Add(c.manager.cfg.WriteWait))
			return
		}
	}
}

func (c *Client) markAlive() {
	select {
	case c.alive <- struct{}{}:
	default:
	}
}

func lessonLabel(lessonID int64) string {
	return strconv.FormatInt(lessonID, 10)
}
