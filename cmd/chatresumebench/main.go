// Command chatresumebench verifies cursor-based WebSocket recovery through the
// public endpoint. It is intentionally small and records machine-readable data.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type config struct {
	URL         string
	Token       string
	LessonID    int64
	AnchorID    string
	Messages    int
	Observe     time.Duration
	Output      string
	Environment string
}

type frame struct {
	Type            string `json:"type"`
	MessageID       string `json:"message_id"`
	ClientMessageID string `json:"client_message_id"`
	Content         string `json:"content"`
	AfterMessageID  string `json:"after_message_id"`
	Recovered       int    `json:"recovered"`
	Truncated       bool   `json:"truncated"`
	Error           string `json:"error"`
}

type result struct {
	StartedAt           time.Time     `json:"started_at"`
	FinishedAt          time.Time     `json:"finished_at"`
	Environment         string        `json:"environment"`
	LessonID            int64         `json:"lesson_id"`
	AnchorMessageID     string        `json:"anchor_message_id"`
	OfflineMessagesSent int           `json:"offline_messages_sent"`
	AcceptedAcks        int64         `json:"accepted_acks"`
	ResumeReported      int           `json:"resume_reported"`
	ExpectedRecovered   int           `json:"expected_recovered"`
	UniqueRecovered     int           `json:"unique_recovered"`
	DuplicateDeliveries int           `json:"duplicate_deliveries"`
	MissingMessages     int           `json:"missing_messages"`
	ResumeLatency       time.Duration `json:"resume_latency_ns"`
	PostResumeObserve   time.Duration `json:"post_resume_observe_ns"`
	ResumeStatus        string        `json:"resume_status"`
	ResumeError         string        `json:"resume_error,omitempty"`
	Truncated           bool          `json:"truncated"`
	Success             bool          `json:"success"`
}

func main() {
	cfg := parseFlags()
	if err := validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "chatresumebench:", err)
		os.Exit(2)
	}
	res, err := run(context.Background(), cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chatresumebench:", err)
		os.Exit(1)
	}
	payload, _ := json.MarshalIndent(res, "", "  ")
	payload = append(payload, '\n')
	if cfg.Output != "" {
		if err := os.WriteFile(cfg.Output, payload, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "chatresumebench:", err)
			os.Exit(1)
		}
	}
	_, _ = os.Stdout.Write(payload)
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.URL, "url", "ws://127.0.0.1:8080/ws/live_chat", "chat WebSocket URL")
	flag.StringVar(&cfg.Token, "token", os.Getenv("LIVECLASS_CHAT_TOKEN"), "access token or LIVECLASS_CHAT_TOKEN")
	flag.Int64Var(&cfg.LessonID, "lesson-id", 0, "lesson containing the token user")
	flag.StringVar(&cfg.AnchorID, "anchor-message-id", "", "last message observed before disconnect")
	flag.IntVar(&cfg.Messages, "messages", 20, "messages persisted while the observer is offline")
	flag.DurationVar(&cfg.Observe, "post-resume-observe", 8*time.Second, "window for detecting delayed duplicates")
	flag.StringVar(&cfg.Output, "output", "", "JSON result path")
	flag.StringVar(&cfg.Environment, "environment", "", "benchmark environment label")
	flag.Parse()
	return cfg
}

func validate(cfg config) error {
	if cfg.Token == "" || cfg.LessonID <= 0 || cfg.AnchorID == "" {
		return errors.New("token, lesson-id and anchor-message-id are required")
	}
	if cfg.Messages <= 0 || cfg.Messages > 40 || cfg.Observe < 0 {
		return errors.New("messages must be 1..40 and observe must not be negative")
	}
	return nil
}

func run(ctx context.Context, cfg config) (result, error) {
	res := result{StartedAt: time.Now(), Environment: cfg.Environment, LessonID: cfg.LessonID,
		AnchorMessageID: cfg.AnchorID, ExpectedRecovered: cfg.Messages, PostResumeObserve: cfg.Observe}
	sender, err := dial(ctx, cfg, "")
	if err != nil {
		return res, fmt.Errorf("dial sender: %w", err)
	}
	defer sender.Close()
	var acks atomic.Int64
	go func() {
		for {
			var message frame
			if sender.ReadJSON(&message) != nil {
				return
			}
			if message.Type == "chat_ack" && message.Error == "" {
				acks.Add(1)
			}
		}
	}()

	runID := fmt.Sprintf("resume%d", time.Now().UnixNano())
	expected := make(map[string]struct{}, cfg.Messages)
	for i := 0; i < cfg.Messages; i++ {
		clientID := fmt.Sprintf("%s-%d", runID, i)
		content := fmt.Sprintf("%s offline message %d", runID, i)
		expected[content] = struct{}{}
		if err := sender.WriteJSON(map[string]string{"client_message_id": clientID, "content": content}); err != nil {
			return res, fmt.Errorf("send offline message %d: %w", i, err)
		}
	}
	res.OfflineMessagesSent = cfg.Messages
	deadline := time.Now().Add(5 * time.Second)
	for acks.Load() < int64(cfg.Messages) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	res.AcceptedAcks = acks.Load()

	resumeStarted := time.Now()
	observer, err := dial(ctx, cfg, cfg.AnchorID)
	if err != nil {
		return res, fmt.Errorf("dial observer: %w", err)
	}
	defer observer.Close()
	seen := make(map[string]struct{}, cfg.Messages)
	readDeadline := time.Now().Add(10 * time.Second)
	_ = observer.SetReadDeadline(readDeadline)
	for {
		var message frame
		if err := observer.ReadJSON(&message); err != nil {
			return res, fmt.Errorf("read resume: %w", err)
		}
		if message.Type == "resume_complete" || message.Type == "resume_error" {
			res.ResumeStatus, res.ResumeError = message.Type, message.Error
			res.ResumeReported, res.Truncated = message.Recovered, message.Truncated
			res.ResumeLatency = time.Since(resumeStarted)
			break
		}
		if _, belongs := expected[message.Content]; belongs {
			if _, duplicate := seen[message.MessageID]; duplicate {
				res.DuplicateDeliveries++
			} else {
				seen[message.MessageID] = struct{}{}
			}
		}
	}

	observeUntil := time.Now().Add(cfg.Observe)
	_ = observer.SetReadDeadline(observeUntil)
	for time.Now().Before(observeUntil) {
		var message frame
		if err := observer.ReadJSON(&message); err != nil {
			break
		}
		if _, belongs := expected[message.Content]; belongs {
			if _, duplicate := seen[message.MessageID]; duplicate {
				res.DuplicateDeliveries++
			} else {
				seen[message.MessageID] = struct{}{}
			}
		}
	}
	res.UniqueRecovered = len(seen)
	res.MissingMessages = cfg.Messages - len(seen)
	res.Success = res.AcceptedAcks == int64(cfg.Messages) && res.MissingMessages == 0 &&
		res.DuplicateDeliveries == 0 && res.ResumeStatus == "resume_complete" && res.ResumeError == "" && !res.Truncated
	res.FinishedAt = time.Now()
	return res, nil
}

func dial(ctx context.Context, cfg config, lastMessageID string) (*websocket.Conn, error) {
	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set("lesson_id", fmt.Sprint(cfg.LessonID))
	query.Set("token", cfg.Token)
	if strings.TrimSpace(lastMessageID) != "" {
		query.Set("last_message_id", lastMessageID)
	}
	parsed.RawQuery = query.Encode()
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, response, err := websocket.DefaultDialer.DialContext(dialCtx, parsed.String(), http.Header{})
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	return conn, err
}
