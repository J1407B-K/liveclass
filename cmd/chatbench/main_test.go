package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMarkerTimestamp(t *testing.T) {
	want := time.Unix(0, 123456789)
	got, ok := markerTimestamp("chatbench run1 7 123456789", "run1")
	if !ok || !got.Equal(want) {
		t.Fatalf("got %v, %v; want %v, true", got, ok, want)
	}
	if _, ok := markerTimestamp("chatbench another 7 123456789", "run1"); ok {
		t.Fatal("accepted another run")
	}
}

func TestMultipleUsersAndDuplicateAccounting(t *testing.T) {
	var mu sync.Mutex
	users := make(map[string]int)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var msg map[string]string
			if conn.ReadJSON(&msg) != nil {
				return
			}
			mu.Lock()
			users[r.URL.Query().Get("token")]++
			mu.Unlock()
			// Two frames representing the same delivery must count only once.
			for i := 0; i < 2; i++ {
				if conn.WriteJSON(msg) != nil {
					return
				}
			}
		}
	}))
	defer server.Close()
	cfg := config{URL: "ws" + strings.TrimPrefix(server.URL, "http"), Tokens: []string{"user1", "user2"}, LessonIDs: []int64{1}, Connections: 2, ConnectWorkers: 2, ConnectTimeout: time.Second, QPS: 40, Duration: 250 * time.Millisecond, Drain: 100 * time.Millisecond, MessageBytes: 128}
	result, err := run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if users["user1"] == 0 || users["user2"] == 0 {
		t.Fatalf("users not rotated: %v", users)
	}
	if result.MessagesSent == 0 || result.MessagesReceived != result.MessagesSent || result.Duplicates != result.MessagesSent {
		t.Fatalf("bad accounting: %+v", result)
	}
	if result.Latency.Samples != int(result.MessagesReceived) || result.ReadErrors != 0 {
		t.Fatalf("bad readings: %+v", result)
	}
}

func TestSummarize(t *testing.T) {
	got := summarize([]float64{100, 1, 50, 99, 10})
	if got.Samples != 5 || got.P50 != 50 || got.P95 != 99 || got.P99 != 99 || got.Max != 100 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestParseMetrics(t *testing.T) {
	input := `# HELP go_goroutines Number of goroutines.
go_goroutines 42
go_memstats_heap_alloc_bytes 1024
ignored_metric 7
request_total{method="GET"} 10
chat_accepted_total{delivery_status="queued"} 3
chat_accepted_total{delivery_status="duplicate"} 2
`
	got, err := parseMetrics(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 || got["go_goroutines"] != 42 || got["go_memstats_heap_alloc_bytes"] != 1024 ||
		got["chat_accepted_total"] != 5 ||
		got[`chat_accepted_total{delivery_status="queued"}`] != 3 ||
		got[`chat_accepted_total{delivery_status="duplicate"}`] != 2 {
		t.Fatalf("unexpected metrics: %#v", got)
	}
}
