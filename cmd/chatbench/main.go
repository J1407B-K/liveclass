// Command chatbench runs a black-box baseline workload against LiveClass chat.
// It deliberately uses only the public WebSocket endpoint, so the result covers
// authentication, permission checks, rate limiting, RPC, persistence, Kafka and
// WebSocket fanout.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type config struct {
	URL                   string
	Token                 string
	LessonID              int64
	LessonIDs             []int64
	LessonIDsRaw          string
	Connections           int
	SlowConsumers         int
	DisconnectBeforeLoad  int
	MessageBytes          int
	Scenario              string
	QPS                   float64
	Duration              time.Duration
	Warmup                time.Duration
	Drain                 time.Duration
	ConnectTimeout        time.Duration
	ConnectWorkers        int
	MetricsEndpoints      string
	Output                string
	Environment           string
	CPUModel              string
	Memory                string
	RedisDeployment       string
	MongoDeployment       string
	KafkaDeployment       string
	RepeatClientMessageID bool
}

type receivedMessage struct {
	Type           string `json:"type"`
	Content        string `json:"content"`
	DeliveryStatus string `json:"delivery_status"`
}

type openedConnection struct {
	conn     *websocket.Conn
	lessonID int64
}

type workloadResult struct {
	StartedAt              time.Time            `json:"started_at"`
	FinishedAt             time.Time            `json:"finished_at"`
	Environment            environment          `json:"benchmark_environment"`
	Workload               workload             `json:"workload"`
	ConnectionsOpened      int64                `json:"connections_opened"`
	ConnectionsActive      int64                `json:"connections_active_during_load"`
	HealthyReaders         int64                `json:"healthy_readers"`
	SlowConsumers          int64                `json:"slow_consumers"`
	DisconnectedBeforeLoad int64                `json:"disconnected_before_load"`
	ConnectionErrors       int64                `json:"connection_errors"`
	MessagesSent           int64                `json:"messages_sent"`
	MessagesReceived       int64                `json:"fanout_deliveries_received"`
	ExpectedDeliveries     int64                `json:"fanout_deliveries_expected"`
	SendErrors             int64                `json:"send_errors"`
	ReadErrors             int64                `json:"read_errors"`
	ErrorRate              float64              `json:"delivery_error_rate_percent"`
	Throughput             float64              `json:"fanout_deliveries_per_second"`
	Latency                latencySummary       `json:"fanout_latency_ms"`
	Acknowledgements       int64                `json:"acknowledgements"`
	AckStatuses            map[string]int64     `json:"ack_statuses,omitempty"`
	Metrics                map[string]metricRun `json:"prometheus_samples,omitempty"`
}

type environment struct {
	Label      string `json:"label,omitempty"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	CPUCount   int    `json:"cpu_count"`
	GoVersion  string `json:"go_version"`
	ClientHost string `json:"client_host,omitempty"`
	CPUModel   string `json:"cpu_model,omitempty"`
	Memory     string `json:"memory,omitempty"`
	Redis      string `json:"redis_deployment,omitempty"`
	MongoDB    string `json:"mongodb_deployment,omitempty"`
	Kafka      string `json:"kafka_deployment,omitempty"`
}

type workload struct {
	URL                   string  `json:"url"`
	LessonID              int64   `json:"lesson_id"`
	LessonIDs             []int64 `json:"lesson_ids,omitempty"`
	Connections           int     `json:"connections"`
	Scenario              string  `json:"scenario"`
	SlowConsumers         int     `json:"slow_consumers"`
	DisconnectBeforeLoad  int     `json:"disconnect_before_load"`
	MessageBytes          int     `json:"message_bytes"`
	ConnectWorkers        int     `json:"connect_workers"`
	QPS                   float64 `json:"messages_per_second"`
	Duration              string  `json:"duration"`
	Warmup                string  `json:"warmup"`
	Drain                 string  `json:"drain"`
	RepeatClientMessageID bool    `json:"repeat_client_message_id"`
}

type latencySummary struct {
	Samples int     `json:"samples"`
	P50     float64 `json:"p50"`
	P95     float64 `json:"p95"`
	P99     float64 `json:"p99"`
	Max     float64 `json:"max"`
}

type metricRun struct {
	Samples                  int                `json:"samples"`
	WindowSeconds            float64            `json:"window_seconds"`
	ProcessCPUSecondsDelta   float64            `json:"process_cpu_seconds_delta,omitempty"`
	AverageProcessCPUPercent float64            `json:"average_process_cpu_percent,omitempty"`
	First                    map[string]float64 `json:"first,omitempty"`
	Last                     map[string]float64 `json:"last,omitempty"`
	Peak                     map[string]float64 `json:"peak,omitempty"`
}

type metricCollector struct {
	mu      sync.Mutex
	samples map[string][]map[string]float64
}

var selectedMetrics = map[string]struct{}{
	"go_goroutines":                               {},
	"go_memstats_heap_alloc_bytes":                {},
	"go_memstats_alloc_bytes":                     {},
	"go_memstats_heap_objects":                    {},
	"go_memstats_gc_cpu_fraction":                 {},
	"process_cpu_seconds_total":                   {},
	"process_resident_memory_bytes":               {},
	"process_virtual_memory_bytes":                {},
	"process_open_fds":                            {},
	"process_max_fds":                             {},
	"active_websocket_connections":                {},
	"chat_queue_depth":                            {},
	"slow_consumer_total":                         {},
	"dropped_messages_total":                      {},
	"subscriber_reconnect_total":                  {},
	"chat_subscriber_connected":                   {},
	"chat_publish_queue_depth":                    {},
	"chat_publish_queue_full_total":               {},
	"chat_accepted_total":                         {},
	"chat_outbox_pending":                         {},
	"chat_outbox_claimed_total":                   {},
	"chat_outbox_published_total":                 {},
	"chat_outbox_retry_total":                     {},
	"chat_duplicate_deliveries_suppressed_total":  {},
	"chat_messages_total":                         {},
	"websocket_write_errors_total":                {},
	"chat_publish_errors_total":                   {},
	"chat_redis_rate_limit_latency_seconds_sum":   {},
	"chat_redis_rate_limit_latency_seconds_count": {},
	"chat_rpc_latency_seconds_sum":                {},
	"chat_rpc_latency_seconds_count":              {},
	"chat_fanout_latency_seconds_sum":             {},
	"chat_fanout_latency_seconds_count":           {},
	"chat_mongo_latency_seconds_sum":              {},
	"chat_mongo_latency_seconds_count":            {},
	"chat_publish_latency_seconds_sum":            {},
	"chat_publish_latency_seconds_count":          {},
}

func main() {
	cfg := parseFlags()
	if err := validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "chatbench:", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := run(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chatbench:", err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "chatbench:", err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if cfg.Output != "" {
		if err := os.WriteFile(cfg.Output, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "chatbench:", err)
			os.Exit(1)
		}
	}
	_, _ = os.Stdout.Write(data)
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.URL, "url", "ws://127.0.0.1:8080/ws/live_chat", "chat WebSocket URL")
	flag.StringVar(&cfg.Token, "token", "", "valid access token (or set LIVECLASS_CHAT_TOKEN)")
	flag.Int64Var(&cfg.LessonID, "lesson-id", 0, "lesson containing the token user")
	flag.StringVar(&cfg.LessonIDsRaw, "lesson-ids", "", "comma-separated lesson IDs; connections and sends are distributed round-robin")
	flag.IntVar(&cfg.Connections, "connections", 100, "number of WebSocket clients")
	flag.IntVar(&cfg.SlowConsumers, "slow-consumers", 0, "clients that never read WebSocket frames")
	flag.IntVar(&cfg.DisconnectBeforeLoad, "disconnect-before-load", 0, "connections closed without a close frame before sending")
	flag.IntVar(&cfg.MessageBytes, "message-bytes", 128, "approximate chat content size in bytes")
	flag.StringVar(&cfg.Scenario, "scenario", "hot-room", "scenario label recorded in the result")
	flag.Float64Var(&cfg.QPS, "qps", 5, "aggregate messages sent per second")
	flag.DurationVar(&cfg.Duration, "duration", 30*time.Second, "measured workload duration")
	flag.DurationVar(&cfg.Warmup, "warmup", 5*time.Second, "warmup duration")
	flag.DurationVar(&cfg.Drain, "drain", 3*time.Second, "time to wait for final fanout deliveries")
	flag.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "timeout per WebSocket handshake")
	flag.IntVar(&cfg.ConnectWorkers, "connect-workers", 50, "maximum concurrent WebSocket handshakes")
	flag.StringVar(&cfg.MetricsEndpoints, "metrics", "http://127.0.0.1:10001/metrics,http://127.0.0.1:10005/metrics", "comma-separated Prometheus endpoints")
	flag.StringVar(&cfg.Output, "output", "", "optional JSON result path")
	flag.StringVar(&cfg.Environment, "environment", "", "deployment label, e.g. local-docker-m2-16gb")
	flag.StringVar(&cfg.CPUModel, "cpu-model", "", "CPU model recorded in the result")
	flag.StringVar(&cfg.Memory, "memory", "", "host memory recorded in the result, e.g. 16GB")
	flag.StringVar(&cfg.RedisDeployment, "redis", "", "Redis version and deployment description")
	flag.StringVar(&cfg.MongoDeployment, "mongo", "", "MongoDB version and deployment description")
	flag.StringVar(&cfg.KafkaDeployment, "kafka", "", "Kafka version and deployment description")
	flag.BoolVar(&cfg.RepeatClientMessageID, "repeat-client-message-id", false, "reuse one client_message_id and payload to verify idempotency")
	flag.Parse()
	if cfg.Token == "" {
		cfg.Token = os.Getenv("LIVECLASS_CHAT_TOKEN")
	}
	if cfg.LessonIDsRaw != "" {
		for _, value := range splitNonEmpty(cfg.LessonIDsRaw) {
			lessonID, err := strconv.ParseInt(value, 10, 64)
			if err != nil || lessonID <= 0 {
				cfg.LessonIDs = nil
				break
			}
			cfg.LessonIDs = append(cfg.LessonIDs, lessonID)
		}
	} else if cfg.LessonID > 0 {
		cfg.LessonIDs = []int64{cfg.LessonID}
	}
	return cfg
}

func validate(cfg config) error {
	if cfg.Token == "" {
		return errors.New("-token or LIVECLASS_CHAT_TOKEN is required")
	}
	if len(cfg.LessonIDs) == 0 || cfg.Connections <= 0 || cfg.ConnectWorkers <= 0 || cfg.QPS <= 0 || cfg.Duration <= 0 {
		return errors.New("lesson-id, connections, connect-workers, qps and duration must be positive")
	}
	if cfg.Warmup < 0 || cfg.Drain < 0 || cfg.ConnectTimeout <= 0 {
		return errors.New("durations must not be negative and connect-timeout must be positive")
	}
	if cfg.DisconnectBeforeLoad < 0 || cfg.DisconnectBeforeLoad >= cfg.Connections {
		return errors.New("disconnect-before-load must be between 0 and connections-1")
	}
	if cfg.SlowConsumers < 0 || cfg.SlowConsumers >= cfg.Connections-cfg.DisconnectBeforeLoad {
		return errors.New("slow-consumers must leave at least one healthy reader")
	}
	if cfg.MessageBytes < 64 || cfg.MessageBytes > 3800 {
		return errors.New("message-bytes must be between 64 and 3800")
	}
	return nil
}

func run(ctx context.Context, cfg config) (workloadResult, error) {
	started := time.Now()
	host, _ := os.Hostname()
	result := workloadResult{
		StartedAt: started,
		Environment: environment{Label: cfg.Environment, OS: runtime.GOOS, Arch: runtime.GOARCH,
			CPUCount: runtime.NumCPU(), GoVersion: runtime.Version(), ClientHost: host,
			CPUModel: cfg.CPUModel, Memory: cfg.Memory, Redis: cfg.RedisDeployment,
			MongoDB: cfg.MongoDeployment, Kafka: cfg.KafkaDeployment},
		Workload: workload{URL: cfg.URL, LessonID: cfg.LessonIDs[0], LessonIDs: cfg.LessonIDs, Connections: cfg.Connections, Scenario: cfg.Scenario,
			SlowConsumers: cfg.SlowConsumers, DisconnectBeforeLoad: cfg.DisconnectBeforeLoad, MessageBytes: cfg.MessageBytes, ConnectWorkers: cfg.ConnectWorkers,
			QPS: cfg.QPS, Duration: cfg.Duration.String(), Warmup: cfg.Warmup.String(), Drain: cfg.Drain.String(), RepeatClientMessageID: cfg.RepeatClientMessageID},
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	metricsCtx, cancelMetrics := context.WithCancel(ctx)
	defer cancelMetrics()
	collector := &metricCollector{samples: make(map[string][]map[string]float64)}
	metricsDone := make(chan struct{})
	endpoints := splitNonEmpty(cfg.MetricsEndpoints)
	go func() {
		defer close(metricsDone)
		collector.collect(metricsCtx, endpoints)
	}()

	conns, connectionErrors := openConnections(runCtx, cfg)
	result.ConnectionErrors = int64(connectionErrors)
	result.ConnectionsOpened = int64(len(conns))
	if len(conns) == 0 {
		return result, errors.New("no WebSocket connection opened")
	}
	if cfg.DisconnectBeforeLoad >= len(conns) {
		closeAll(conns)
		return result, errors.New("too few connections opened for disconnect-before-load")
	}
	for _, conn := range conns[len(conns)-cfg.DisconnectBeforeLoad:] {
		_ = conn.conn.Close()
	}
	conns = conns[:len(conns)-cfg.DisconnectBeforeLoad]
	result.DisconnectedBeforeLoad = int64(cfg.DisconnectBeforeLoad)
	result.ConnectionsActive = int64(len(conns))
	if cfg.SlowConsumers >= len(conns) {
		closeAll(conns)
		return result, errors.New("too few connections opened to leave a healthy reader")
	}
	healthyCount := len(conns) - cfg.SlowConsumers
	result.HealthyReaders = int64(healthyCount)
	result.SlowConsumers = int64(cfg.SlowConsumers)
	for _, conn := range conns[healthyCount:] {
		if tcp, ok := conn.conn.UnderlyingConn().(*net.TCPConn); ok {
			_ = tcp.SetReadBuffer(1024)
		}
	}

	runID := strconv.FormatInt(time.Now().UnixNano(), 36)
	var sent, received, expected, sendErrors, readErrors, acknowledgements atomic.Int64
	ackStatuses := make(map[string]int64)
	var ackMu sync.Mutex
	var latencyMu sync.Mutex
	latencies := make([]float64, 0, len(conns)*int(cfg.QPS*cfg.Duration.Seconds()))
	var readers sync.WaitGroup
	for _, conn := range conns[:healthyCount] {
		readers.Add(1)
		go func(c *websocket.Conn) {
			defer readers.Done()
			for {
				var msg receivedMessage
				if err := c.ReadJSON(&msg); err != nil {
					if runCtx.Err() == nil {
						readErrors.Add(1)
					}
					return
				}
				if msg.Type == "chat_ack" {
					acknowledgements.Add(1)
					ackMu.Lock()
					ackStatuses[msg.DeliveryStatus]++
					ackMu.Unlock()
					continue
				}
				sentAt, ok := markerTimestamp(msg.Content, runID)
				if !ok {
					continue
				}
				received.Add(1)
				ms := float64(time.Since(sentAt).Nanoseconds()) / float64(time.Millisecond)
				latencyMu.Lock()
				latencies = append(latencies, ms)
				latencyMu.Unlock()
			}
		}(conn.conn)
	}
	healthyPerLesson := make(map[int64]int64)
	senders := make(map[int64]*websocket.Conn)
	for _, conn := range conns[:healthyCount] {
		healthyPerLesson[conn.lessonID]++
		if senders[conn.lessonID] == nil {
			senders[conn.lessonID] = conn.conn
		}
	}
	for _, lessonID := range cfg.LessonIDs {
		if senders[lessonID] == nil {
			closeAll(conns)
			return result, fmt.Errorf("lesson %d has no healthy connection", lessonID)
		}
	}

	if err := sleepContext(runCtx, cfg.Warmup); err != nil {
		closeAll(conns)
		return result, err
	}
	interval := time.Duration(float64(time.Second) / cfg.QPS)
	ticker := time.NewTicker(interval)
	deadline := time.NewTimer(cfg.Duration)
	seq := int64(0)
	repeatedContent := makeContent(runID, 1, time.Now(), cfg.MessageBytes)
sendLoop:
	for {
		select {
		case <-runCtx.Done():
			break sendLoop
		case <-deadline.C:
			break sendLoop
		case now := <-ticker.C:
			seq++
			lessonID := cfg.LessonIDs[(seq-1)%int64(len(cfg.LessonIDs))]
			content := makeContent(runID, seq, now, cfg.MessageBytes)
			clientMessageID := fmt.Sprintf("%s-%d", runID, seq)
			if cfg.RepeatClientMessageID {
				content = repeatedContent
				clientMessageID = runID + "-idempotency"
			}
			if err := senders[lessonID].WriteJSON(map[string]string{"content": content, "client_message_id": clientMessageID}); err != nil {
				sendErrors.Add(1)
				continue
			}
			sent.Add(1)
			if !cfg.RepeatClientMessageID || seq == 1 {
				expected.Add(healthyPerLesson[lessonID])
			}
		}
	}
	ticker.Stop()
	if !deadline.Stop() {
		select {
		case <-deadline.C:
		default:
		}
	}
	_ = sleepContext(runCtx, cfg.Drain)
	cancelRun()
	closeAll(conns)
	readers.Wait()
	cancelMetrics()
	<-metricsDone
	_ = sleepContext(context.Background(), 500*time.Millisecond)
	finalCtx, finalCancel := context.WithTimeout(context.Background(), 3*time.Second)
	collector.sample(finalCtx, endpoints)
	finalCancel()

	result.FinishedAt = time.Now()
	result.MessagesSent = sent.Load()
	result.MessagesReceived = received.Load()
	result.SendErrors = sendErrors.Load()
	result.ReadErrors = readErrors.Load()
	result.Acknowledgements = acknowledgements.Load()
	result.AckStatuses = ackStatuses
	result.ExpectedDeliveries = expected.Load()
	if result.ExpectedDeliveries > 0 {
		missing := result.ExpectedDeliveries - result.MessagesReceived
		if missing < 0 {
			missing = 0
		}
		result.ErrorRate = float64(missing) * 100 / float64(result.ExpectedDeliveries)
	}
	result.Throughput = float64(result.MessagesReceived) / cfg.Duration.Seconds()
	result.Latency = summarize(latencies)
	result.Metrics = collector.summary(result.FinishedAt.Sub(result.StartedAt))
	return result, nil
}

func openConnections(ctx context.Context, cfg config) ([]openedConnection, int) {
	type dialJob struct {
		index    int
		lessonID int64
	}
	type dialResult struct {
		index    int
		lessonID int64
		conn     *websocket.Conn
		err      error
	}
	workers := cfg.ConnectWorkers
	if workers > cfg.Connections {
		workers = cfg.Connections
	}
	jobs := make(chan dialJob)
	results := make(chan dialResult, cfg.Connections)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				conn, _, err := dial(ctx, cfg, job.lessonID)
				results <- dialResult{index: job.index, lessonID: job.lessonID, conn: conn, err: err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for i := 0; i < cfg.Connections; i++ {
			job := dialJob{index: i, lessonID: cfg.LessonIDs[i%len(cfg.LessonIDs)]}
			select {
			case jobs <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]openedConnection, cfg.Connections)
	opened := make([]bool, cfg.Connections)
	errors := 0
	for result := range results {
		if result.err != nil {
			errors++
			continue
		}
		ordered[result.index] = openedConnection{conn: result.conn, lessonID: result.lessonID}
		opened[result.index] = true
	}
	conns := make([]openedConnection, 0, cfg.Connections-errors)
	for i, conn := range ordered {
		if opened[i] {
			conns = append(conns, conn)
		}
	}
	return conns, errors
}

func dial(ctx context.Context, cfg config, lessonID int64) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, nil, err
	}
	q := u.Query()
	q.Set("token", cfg.Token)
	q.Set("lesson_id", strconv.FormatInt(lessonID, 10))
	u.RawQuery = q.Encode()
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = cfg.ConnectTimeout
	return dialer.DialContext(ctx, u.String(), nil)
}

func markerTimestamp(content, runID string) (time.Time, bool) {
	parts := strings.Fields(content)
	if len(parts) < 4 || parts[0] != "chatbench" || parts[1] != runID {
		return time.Time{}, false
	}
	ns, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(0, ns), true
}

func makeContent(runID string, seq int64, now time.Time, size int) string {
	content := fmt.Sprintf("chatbench %s %d %d", runID, seq, now.UnixNano())
	if len(content) < size {
		content += " " + strings.Repeat("x", size-len(content)-1)
	}
	return content
}

func summarize(values []float64) latencySummary {
	if len(values) == 0 {
		return latencySummary{}
	}
	sort.Float64s(values)
	return latencySummary{Samples: len(values), P50: percentile(values, .50), P95: percentile(values, .95),
		P99: percentile(values, .99), Max: values[len(values)-1]}
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	i := int(float64(len(values)-1) * p)
	return values[i]
}

func (c *metricCollector) collect(ctx context.Context, endpoints []string) {
	if len(endpoints) == 0 {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		c.sample(ctx, endpoints)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *metricCollector) sample(ctx context.Context, endpoints []string) {
	for _, endpoint := range endpoints {
		if sample, err := scrape(ctx, endpoint); err == nil {
			c.mu.Lock()
			c.samples[endpoint] = append(c.samples[endpoint], sample)
			c.mu.Unlock()
		}
	}
}

func scrape(ctx context.Context, endpoint string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics status %s", resp.Status)
	}
	return parseMetrics(resp.Body)
}

func parseMetrics(r io.Reader) (map[string]float64, error) {
	values := make(map[string]float64)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		metricToken := fields[0]
		metricName := metricToken
		if labelStart := strings.IndexByte(metricToken, '{'); labelStart >= 0 {
			metricName = metricToken[:labelStart]
		}
		if _, ok := selectedMetrics[metricName]; !ok {
			continue
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err == nil {
			values[metricName] += value
			if metricToken != metricName {
				values[metricToken] = value
			}
		}
	}
	return values, scanner.Err()
}

func (c *metricCollector) summary(window time.Duration) map[string]metricRun {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]metricRun, len(c.samples))
	for endpoint, samples := range c.samples {
		if len(samples) == 0 {
			continue
		}
		peak := make(map[string]float64)
		for _, sample := range samples {
			for name, value := range sample {
				if old, ok := peak[name]; !ok || value > old {
					peak[name] = value
				}
			}
		}
		cpuDelta := samples[len(samples)-1]["process_cpu_seconds_total"] - samples[0]["process_cpu_seconds_total"]
		if cpuDelta < 0 {
			cpuDelta = 0
		}
		averageCPU := 0.0
		if window > 0 {
			averageCPU = cpuDelta * 100 / window.Seconds()
		}
		result[endpoint] = metricRun{
			Samples: len(samples), WindowSeconds: window.Seconds(), ProcessCPUSecondsDelta: cpuDelta,
			AverageProcessCPUPercent: averageCPU, First: samples[0], Last: samples[len(samples)-1], Peak: peak,
		}
	}
	return result
}

func splitNonEmpty(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d == 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func closeAll(conns []openedConnection) {
	for _, conn := range conns {
		_ = conn.conn.Close()
	}
}
