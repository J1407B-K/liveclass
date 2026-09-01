// Command webrtcbench drives the public Kitex signaling API with real Pion
// PeerConnections. It measures one-publisher/N-viewer SFU behavior without
// importing the server implementation.
package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
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

	"github.com/cloudwego/kitex/client"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"

	"liveclass/idl/kitex_gen/common"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/idl/kitex_gen/webrtc_live/webrtclive"
)

type config struct {
	Endpoint          string
	MetricsEndpoint   string
	TeacherID         int64
	ViewerID          int64
	LessonID          int64
	Viewers           int
	ConnectWorkers    int
	Duration          time.Duration
	Warmup            time.Duration
	Drain             time.Duration
	ConnectTimeout    time.Duration
	FPS               int
	PayloadBytes      int
	Scenario          string
	Output            string
	Environment       string
	CPUModel          string
	Memory            string
	DisconnectViewers int
	AudioDelay        time.Duration
}

type environment struct {
	Label     string `json:"label,omitempty"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	CPUCount  int    `json:"cpu_count"`
	CPUModel  string `json:"cpu_model,omitempty"`
	Memory    string `json:"memory,omitempty"`
}

type workload struct {
	Scenario          string `json:"scenario"`
	Endpoint          string `json:"endpoint"`
	LessonID          int64  `json:"lesson_id"`
	Viewers           int    `json:"viewers"`
	ConnectWorkers    int    `json:"connect_workers"`
	FPS               int    `json:"fps"`
	PayloadBytes      int    `json:"payload_bytes"`
	ApproxBitrate     int64  `json:"publisher_approx_bitrate_bps"`
	Warmup            string `json:"warmup"`
	Duration          string `json:"duration"`
	Drain             string `json:"drain"`
	DisconnectViewers int    `json:"viewers_disconnected_before_measurement,omitempty"`
	AudioDelay        string `json:"publisher_audio_delay,omitempty"`
}

type latencySummary struct {
	Samples int     `json:"samples"`
	P50     float64 `json:"p50_ms"`
	P95     float64 `json:"p95_ms"`
	P99     float64 `json:"p99_ms"`
	Max     float64 `json:"max_ms"`
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

type result struct {
	StartedAt        time.Time      `json:"started_at"`
	FinishedAt       time.Time      `json:"finished_at"`
	Environment      environment    `json:"benchmark_environment"`
	Workload         workload       `json:"workload"`
	PublisherOpened  bool           `json:"publisher_opened"`
	ViewersOpened    int64          `json:"viewers_opened"`
	ViewersMeasured  int64          `json:"viewers_measured"`
	ViewersWithAudio int64          `json:"viewers_with_audio,omitempty"`
	ConnectionErrors int64          `json:"connection_errors"`
	ConnectionSetup  latencySummary `json:"viewer_connection_setup_latency"`
	PacketsSent      int64          `json:"publisher_packets_sent"`
	PacketsReceived  int64          `json:"viewer_packets_received"`
	PacketsExpected  int64          `json:"viewer_packets_expected"`
	DeliveryRate     float64        `json:"packet_delivery_rate_percent"`
	FanoutLatency    latencySummary `json:"rtp_fanout_latency"`
	FanoutPerSecond  float64        `json:"fanout_packets_per_second"`
	BytesReceived    int64          `json:"viewer_payload_bytes_received"`
	ReceiveBitrate   float64        `json:"aggregate_receive_bitrate_bps"`
	Metrics          metricRun      `json:"prometheus_samples,omitempty"`
}

type viewer struct {
	pc           *webrtc.PeerConnection
	packets      atomic.Int64
	bytes        atomic.Int64
	audioPackets atomic.Int64
	measure      atomic.Bool
	mu           sync.Mutex
	seen         map[uint16]struct{}
	latency      []time.Duration
	firstSeq     uint16
	lastSeq      uint16
	hasSeq       bool
}

type metricCollector struct {
	mu      sync.Mutex
	samples []map[string]float64
}

var selectedMetrics = map[string]struct{}{
	"go_goroutines": {}, "go_memstats_heap_alloc_bytes": {}, "go_memstats_alloc_bytes": {},
	"go_memstats_heap_objects": {}, "process_cpu_seconds_total": {},
	"process_resident_memory_bytes": {}, "process_open_fds": {},
	"webrtc_active_peer_connections": {}, "webrtc_rtp_packets_in_total": {},
	"webrtc_rtp_packets_out_total": {}, "webrtc_rtp_write_errors_total": {},
	"webrtc_nack_received_total": {}, "webrtc_pli_received_total": {},
	"webrtc_pli_forwarded_total": {}, "webrtc_pli_suppressed_total": {},
	"webrtc_rtp_injected_drops_total":  {},
	"webrtc_track_ready_timeout_total": {},
}

func main() {
	cfg := parseFlags()
	if err := validate(cfg); err != nil {
		fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	got, err := run(ctx, cfg)
	if err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		fatal(err)
	}
	data = append(data, '\n')
	if cfg.Output != "" {
		if err := os.WriteFile(cfg.Output, data, 0o644); err != nil {
			fatal(err)
		}
	}
	_, _ = os.Stdout.Write(data)
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.Endpoint, "endpoint", "127.0.0.1:9001", "WebRTC Kitex service endpoint")
	flag.StringVar(&cfg.MetricsEndpoint, "metrics", "http://127.0.0.1:10004/metrics", "WebRTC Prometheus endpoint")
	flag.Int64Var(&cfg.TeacherID, "teacher-id", 0, "teacher user ID")
	flag.Int64Var(&cfg.ViewerID, "viewer-id", 0, "student user ID, reused by synthetic viewers")
	flag.Int64Var(&cfg.LessonID, "lesson-id", 0, "lesson ID")
	flag.IntVar(&cfg.Viewers, "viewers", 100, "viewer PeerConnections")
	flag.IntVar(&cfg.ConnectWorkers, "connect-workers", 20, "parallel signaling workers")
	flag.DurationVar(&cfg.Duration, "duration", 15*time.Second, "measured media duration")
	flag.DurationVar(&cfg.Warmup, "warmup", 2*time.Second, "media warmup")
	flag.DurationVar(&cfg.Drain, "drain", 2*time.Second, "resource cleanup observation window")
	flag.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 15*time.Second, "per PeerConnection setup timeout")
	flag.IntVar(&cfg.FPS, "fps", 30, "synthetic RTP packets per second")
	flag.IntVar(&cfg.PayloadBytes, "payload-bytes", 1000, "RTP payload bytes")
	flag.StringVar(&cfg.Scenario, "scenario", "single-publisher", "scenario label")
	flag.StringVar(&cfg.Output, "output", "", "write JSON result")
	flag.StringVar(&cfg.Environment, "environment", "", "environment label")
	flag.StringVar(&cfg.CPUModel, "cpu-model", "", "CPU model")
	flag.StringVar(&cfg.Memory, "memory", "", "memory size")
	flag.IntVar(&cfg.DisconnectViewers, "disconnect-viewers", 0, "close this many viewers after warmup, before measurement")
	flag.DurationVar(&cfg.AudioDelay, "audio-delay", -1, "publish Opus audio after this delay; negative disables audio")
	flag.Parse()
	return cfg
}

func validate(cfg config) error {
	if cfg.TeacherID <= 0 || cfg.ViewerID <= 0 || cfg.LessonID <= 0 {
		return errors.New("teacher-id, viewer-id and lesson-id are required")
	}
	if cfg.Viewers <= 0 || cfg.ConnectWorkers <= 0 || cfg.FPS <= 0 || cfg.PayloadBytes < 16 {
		return errors.New("viewers, workers and fps must be positive; payload must be at least 16 bytes")
	}
	if cfg.Duration <= 0 || cfg.ConnectTimeout <= 0 {
		return errors.New("durations must be positive")
	}
	if cfg.DisconnectViewers < 0 || cfg.DisconnectViewers >= cfg.Viewers {
		return errors.New("disconnect-viewers must be between 0 and viewers-1")
	}
	return nil
}

func run(ctx context.Context, cfg config) (out result, runErr error) {
	started := time.Now()
	out = result{StartedAt: started}
	out.Environment = environment{Label: cfg.Environment, OS: runtime.GOOS, Arch: runtime.GOARCH,
		GoVersion: runtime.Version(), CPUCount: runtime.NumCPU(), CPUModel: cfg.CPUModel, Memory: cfg.Memory}
	out.Workload = workload{Scenario: cfg.Scenario, Endpoint: cfg.Endpoint, LessonID: cfg.LessonID,
		Viewers: cfg.Viewers, ConnectWorkers: cfg.ConnectWorkers, FPS: cfg.FPS, PayloadBytes: cfg.PayloadBytes,
		ApproxBitrate: int64(cfg.FPS * cfg.PayloadBytes * 8), Warmup: cfg.Warmup.String(), Duration: cfg.Duration.String()}
	out.Workload.Drain = cfg.Drain.String()
	out.Workload.DisconnectViewers = cfg.DisconnectViewers
	if cfg.AudioDelay >= 0 {
		out.Workload.AudioDelay = cfg.AudioDelay.String()
	}

	rpcClient, err := webrtclive.NewClient("webrtc_liveservice",
		client.WithHostPorts(cfg.Endpoint), client.WithRPCTimeout(cfg.ConnectTimeout))
	if err != nil {
		return out, err
	}
	metricsCtx, cancelMetrics := context.WithCancel(ctx)
	collector := &metricCollector{}
	go collector.collect(metricsCtx, cfg.MetricsEndpoint)
	defer func() { cancelMetrics(); out.Metrics = collector.summary(time.Since(started)) }()

	publisher, videoTrack, audioTrack, err := openPublisher(ctx, rpcClient, cfg)
	if err != nil {
		return out, fmt.Errorf("open publisher: %w", err)
	}
	out.PublisherOpened = true
	defer publisher.Close()

	sendCtx, stopSending := context.WithCancel(ctx)
	var sent atomic.Int64
	go sendRTP(sendCtx, videoTrack, cfg, &sent)
	if audioTrack != nil {
		go sendAudioRTP(sendCtx, audioTrack, cfg.AudioDelay)
	}
	defer stopSending()

	viewers := make([]*viewer, cfg.Viewers)
	setupLatencies := make([]time.Duration, 0, cfg.Viewers)
	var setupMu sync.Mutex
	var opened, connectionErrors atomic.Int64
	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := min(cfg.ConnectWorkers, cfg.Viewers)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				setupStart := time.Now()
				v, openErr := openViewer(ctx, rpcClient, cfg)
				if openErr != nil {
					connectionErrors.Add(1)
					continue
				}
				viewers[idx] = v
				opened.Add(1)
				setupMu.Lock()
				setupLatencies = append(setupLatencies, time.Since(setupStart))
				setupMu.Unlock()
			}
		}()
	}
	for i := 0; i < cfg.Viewers; i++ {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return out, ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	out.ViewersOpened, out.ConnectionErrors = opened.Load(), connectionErrors.Load()
	out.ConnectionSetup = summarizeDurations(setupLatencies)
	for _, v := range viewers {
		if v != nil && v.audioPackets.Load() > 0 {
			out.ViewersWithAudio++
		}
	}
	if opened.Load() == 0 {
		return out, errors.New("no viewers connected")
	}
	defer func() {
		for _, v := range viewers {
			if v != nil {
				_ = v.pc.Close()
			}
		}
	}()

	if err := waitContext(ctx, cfg.Warmup); err != nil {
		return out, err
	}
	for i := 0; i < cfg.DisconnectViewers; i++ {
		if viewers[i] != nil {
			_ = viewers[i].pc.Close()
			viewers[i] = nil
		}
	}
	if cfg.DisconnectViewers > 0 {
		if err := waitContext(ctx, 500*time.Millisecond); err != nil {
			return out, err
		}
	}
	startSent := sent.Load()
	for _, v := range viewers {
		if v != nil {
			out.ViewersMeasured++
			v.startMeasurement()
		}
	}
	measureStart := time.Now()
	if err := waitContext(ctx, cfg.Duration); err != nil {
		return out, err
	}
	measureElapsed := time.Since(measureStart)
	out.PacketsSent = sent.Load() - startSent

	latencies := make([]time.Duration, 0)
	for _, v := range viewers {
		if v == nil {
			continue
		}
		packets, expected, bytes, values := v.snapshot()
		out.PacketsReceived += packets
		out.PacketsExpected += expected
		out.BytesReceived += bytes
		latencies = append(latencies, values...)
	}
	if out.PacketsExpected == 0 {
		out.PacketsExpected = out.PacketsSent * out.ViewersOpened
	}
	if out.PacketsExpected > 0 {
		out.DeliveryRate = float64(out.PacketsReceived) / float64(out.PacketsExpected) * 100
	}
	out.FanoutLatency = summarizeDurations(latencies)
	out.FanoutPerSecond = float64(out.PacketsReceived) / measureElapsed.Seconds()
	out.ReceiveBitrate = float64(out.BytesReceived*8) / measureElapsed.Seconds()
	stopSending()
	for _, v := range viewers {
		if v != nil {
			_ = v.pc.Close()
		}
	}
	_ = publisher.Close()
	if err := waitContext(ctx, cfg.Drain); err != nil {
		return out, err
	}
	out.FinishedAt = time.Now()
	return out, nil
}

func openPublisher(ctx context.Context, cli webrtclive.Client, cfg config) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, *webrtc.TrackLocalStaticRTP, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, nil, nil, err
	}
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "bench-video", "bench-stream")
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	sender, err := pc.AddTrack(track)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	go drainRTCP(sender)
	var audioTrack *webrtc.TrackLocalStaticRTP
	if cfg.AudioDelay >= 0 {
		audioTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2}, "bench-audio", "bench-stream")
		if err != nil {
			_ = pc.Close()
			return nil, nil, nil, err
		}
		audioSender, addErr := pc.AddTrack(audioTrack)
		if addErr != nil {
			_ = pc.Close()
			return nil, nil, nil, addErr
		}
		go drainRTCP(audioSender)
	}
	offer, err := createOffer(pc)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	resp, err := cli.Broadcast(callCtx, &webrtc_live.BroadcastReq{Userid: cfg.TeacherID, LessonId: cfg.LessonID, B64offer: offer})
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	answer, err := responseSDP(resp.GetResp())
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	if err = pc.SetRemoteDescription(answer); err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	if err = waitConnected(callCtx, pc); err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	return pc, track, audioTrack, nil
}

func openViewer(ctx context.Context, cli webrtclive.Client, cfg config) (*viewer, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}
	v := &viewer{pc: pc, seen: make(map[uint16]struct{})}
	videoReady := make(chan struct{})
	audioReady := make(chan struct{})
	var videoOnce, audioOnce sync.Once
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			videoOnce.Do(func() { close(videoReady) })
		} else if track.Kind() == webrtc.RTPCodecTypeAudio {
			audioOnce.Do(func() { close(audioReady) })
		}
		for {
			packet, _, readErr := track.ReadRTP()
			if readErr != nil {
				return
			}
			if track.Kind() == webrtc.RTPCodecTypeVideo {
				v.observe(packet)
			} else {
				v.audioPackets.Add(1)
			}
		}
	})
	if _, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		_ = pc.Close()
		return nil, err
	}
	if cfg.AudioDelay >= 0 {
		if _, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
			_ = pc.Close()
			return nil, err
		}
	}
	offer, err := createOffer(pc)
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	resp, err := cli.View(callCtx, &webrtc_live.ViewReq{Userid: cfg.ViewerID, LessonId: cfg.LessonID, B64offer: offer})
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	answer, err := responseSDP(resp.GetResp())
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if err = pc.SetRemoteDescription(answer); err != nil {
		_ = pc.Close()
		return nil, err
	}
	if err = waitConnected(callCtx, pc); err != nil {
		_ = pc.Close()
		return nil, err
	}
	select {
	case <-videoReady:
	case <-callCtx.Done():
		_ = pc.Close()
		return nil, callCtx.Err()
	}
	if cfg.AudioDelay >= 0 {
		select {
		case <-audioReady:
		case <-callCtx.Done():
			_ = pc.Close()
			return nil, callCtx.Err()
		}
	}
	return v, nil
}

func createOffer(pc *webrtc.PeerConnection) (string, error) {
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	gather := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(offer); err != nil {
		return "", err
	}
	<-gather
	return encodeSDP(pc.LocalDescription())
}

func encodeSDP(description *webrtc.SessionDescription) (string, error) {
	data, err := json.Marshal(description)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func responseSDP(resp *common.Resp) (webrtc.SessionDescription, error) {
	if resp == nil || resp.GetData() == nil || strings.TrimSpace(resp.GetData().GetSdp()) == "" {
		return webrtc.SessionDescription{}, errors.New("signaling response does not contain SDP")
	}
	raw, err := base64.StdEncoding.DecodeString(resp.GetData().GetSdp())
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	var description webrtc.SessionDescription
	if err = json.Unmarshal(raw, &description); err != nil {
		return webrtc.SessionDescription{}, err
	}
	return description, nil
}

func waitConnected(ctx context.Context, pc *webrtc.PeerConnection) error {
	if pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
		return nil
	}
	state := make(chan webrtc.PeerConnectionState, 1)
	pc.OnConnectionStateChange(func(value webrtc.PeerConnectionState) {
		select {
		case state <- value:
		default:
		}
	})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case value := <-state:
			switch value {
			case webrtc.PeerConnectionStateConnected:
				return nil
			case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
				return fmt.Errorf("peer connection state %s", value)
			}
		}
	}
}

func sendRTP(ctx context.Context, track *webrtc.TrackLocalStaticRTP, cfg config, sent *atomic.Int64) {
	interval := time.Second / time.Duration(cfg.FPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	packet := &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 96, SSRC: 424242}, Payload: make([]byte, cfg.PayloadBytes)}
	var sequence uint16
	var timestamp uint32
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			sequence++
			timestamp += uint32(90000 / cfg.FPS)
			packet.SequenceNumber, packet.Timestamp = sequence, timestamp
			binary.BigEndian.PutUint64(packet.Payload[:8], uint64(now.UnixNano()))
			if err := track.WriteRTP(packet); err == nil {
				sent.Add(1)
			}
		}
	}
}

func sendAudioRTP(ctx context.Context, track *webrtc.TrackLocalStaticRTP, delay time.Duration) {
	if err := waitContext(ctx, delay); err != nil {
		return
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	packet := &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SSRC: 434343}, Payload: make([]byte, 160)}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			packet.SequenceNumber++
			packet.Timestamp += 960
			_ = track.WriteRTP(packet)
		}
	}
}

func drainRTCP(sender *webrtc.RTPSender) {
	buffer := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buffer); err != nil {
			return
		}
	}
}

func (v *viewer) startMeasurement() {
	v.mu.Lock()
	v.seen = make(map[uint16]struct{})
	v.latency = nil
	v.hasSeq = false
	v.packets.Store(0)
	v.bytes.Store(0)
	v.measure.Store(true)
	v.mu.Unlock()
}

func (v *viewer) observe(packet *rtp.Packet) {
	if !v.measure.Load() {
		return
	}
	v.packets.Add(1)
	v.bytes.Add(int64(len(packet.Payload)))
	v.mu.Lock()
	if _, exists := v.seen[packet.SequenceNumber]; !exists {
		v.seen[packet.SequenceNumber] = struct{}{}
		if !v.hasSeq {
			v.firstSeq = packet.SequenceNumber
			v.hasSeq = true
		}
		v.lastSeq = packet.SequenceNumber
	}
	if len(packet.Payload) >= 8 {
		sentAt := int64(binary.BigEndian.Uint64(packet.Payload[:8]))
		if sentAt > 0 {
			latency := time.Since(time.Unix(0, sentAt))
			if latency >= 0 && latency < time.Minute {
				v.latency = append(v.latency, latency)
			}
		}
	}
	v.mu.Unlock()
}

func (v *viewer) snapshot() (packets, expected, bytes int64, latency []time.Duration) {
	v.measure.Store(false)
	v.mu.Lock()
	defer v.mu.Unlock()
	packets, bytes = int64(len(v.seen)), v.bytes.Load()
	if v.hasSeq {
		expected = int64(uint16(v.lastSeq-v.firstSeq)) + 1
	}
	latency = append([]time.Duration(nil), v.latency...)
	return
}

func summarizeDurations(values []time.Duration) latencySummary {
	if len(values) == 0 {
		return latencySummary{}
	}
	copyValues := append([]time.Duration(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	return latencySummary{Samples: len(copyValues), P50: durationPercentile(copyValues, .50), P95: durationPercentile(copyValues, .95), P99: durationPercentile(copyValues, .99), Max: float64(copyValues[len(copyValues)-1].Microseconds()) / 1000}
}

func durationPercentile(values []time.Duration, q float64) float64 {
	idx := int(float64(len(values)-1) * q)
	return float64(values[idx].Microseconds()) / 1000
}

func (m *metricCollector) collect(ctx context.Context, endpoint string) {
	if strings.TrimSpace(endpoint) == "" {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if values, err := scrape(ctx, endpoint); err == nil {
			m.mu.Lock()
			m.samples = append(m.samples, values)
			m.mu.Unlock()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func scrape(ctx context.Context, endpoint string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics HTTP %d", resp.StatusCode)
	}
	return parseMetrics(resp.Body)
}

func parseMetrics(reader io.Reader) (map[string]float64, error) {
	values := make(map[string]float64)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		if idx := strings.IndexByte(name, '{'); idx >= 0 {
			name = name[:idx]
		}
		if _, ok := selectedMetrics[name]; !ok {
			continue
		}
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err == nil {
			values[name] += value
		}
	}
	return values, scanner.Err()
}

func (m *metricCollector) summary(window time.Duration) metricRun {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.samples) == 0 {
		return metricRun{}
	}
	first, last := cloneMap(m.samples[0]), cloneMap(m.samples[len(m.samples)-1])
	peak := make(map[string]float64)
	for _, sample := range m.samples {
		for name, value := range sample {
			if value > peak[name] {
				peak[name] = value
			}
		}
	}
	cpuDelta := last["process_cpu_seconds_total"] - first["process_cpu_seconds_total"]
	return metricRun{Samples: len(m.samples), WindowSeconds: window.Seconds(), ProcessCPUSecondsDelta: cpuDelta,
		AverageProcessCPUPercent: cpuDelta / window.Seconds() * 100, First: first, Last: last, Peak: peak}
}

func cloneMap(source map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func waitContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "webrtcbench:", err); os.Exit(1) }
