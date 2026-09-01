package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"liveclass/internal/resilience"
)

type config struct {
	Requests        int           `json:"requests"`
	Concurrency     int           `json:"concurrency"`
	DependencyDelay time.Duration `json:"dependency_delay"`
	Timeout         time.Duration `json:"timeout"`
	Attempts        int           `json:"attempts"`
	Backoff         time.Duration `json:"backoff"`
	OpenDuration    time.Duration `json:"open_duration"`
	Trials          int           `json:"trials"`
}

type trialResult struct {
	WallMS             float64 `json:"wall_ms"`
	P50MS              float64 `json:"p50_ms"`
	P95MS              float64 `json:"p95_ms"`
	P99MS              float64 `json:"p99_ms"`
	CompletionPerSec   float64 `json:"completion_per_sec"`
	UpstreamCalls      int64   `json:"upstream_calls"`
	DependencyErrors   int64   `json:"dependency_errors"`
	Fallbacks          int64   `json:"fallbacks"`
	BusinessErrors     int64   `json:"business_errors"`
	PeakGoroutines     int     `json:"peak_goroutines"`
	EndGoroutineDelta  int     `json:"end_goroutine_delta"`
	PeakHeapDeltaBytes uint64  `json:"peak_heap_delta_bytes"`
	ProcessCPUMS       float64 `json:"process_cpu_ms"`
	BreakerAfterFault  string  `json:"breaker_after_fault"`
	RecoveryProbeCalls int64   `json:"recovery_probe_calls,omitempty"`
	BreakerAfterProbe  string  `json:"breaker_after_probe,omitempty"`
}

type summary struct {
	WallMS             float64 `json:"wall_ms_mean"`
	P50MS              float64 `json:"p50_ms_mean"`
	P95MS              float64 `json:"p95_ms_mean"`
	P99MS              float64 `json:"p99_ms_mean"`
	CompletionPerSec   float64 `json:"completion_per_sec_mean"`
	UpstreamCalls      float64 `json:"upstream_calls_mean"`
	BusinessErrors     float64 `json:"business_errors_mean"`
	PeakGoroutines     float64 `json:"peak_goroutines_mean"`
	PeakHeapDeltaBytes float64 `json:"peak_heap_delta_bytes_mean"`
	ProcessCPUMS       float64 `json:"process_cpu_ms_mean"`
}

type report struct {
	GeneratedAt string `json:"generated_at"`
	Environment struct {
		OS          string `json:"os"`
		Arch        string `json:"arch"`
		GoVersion   string `json:"go_version"`
		LogicalCPUs int    `json:"logical_cpus"`
		CPUModel    string `json:"cpu_model"`
		Memory      string `json:"memory"`
	} `json:"environment"`
	Workload struct {
		Description string `json:"description"`
		Config      config `json:"config"`
	} `json:"workload"`
	Before struct {
		Trials  []trialResult `json:"trials"`
		Summary summary       `json:"summary"`
	} `json:"before_no_protection"`
	After struct {
		Trials  []trialResult `json:"trials"`
		Summary summary       `json:"summary"`
	} `json:"after_timeout_retry_breaker_fallback"`
	Improvement struct {
		P99ReductionPercent          float64 `json:"p99_reduction_percent"`
		WallTimeReductionPercent     float64 `json:"wall_time_reduction_percent"`
		UpstreamCallReductionPercent float64 `json:"upstream_call_reduction_percent"`
	} `json:"improvement"`
	Semantics []string `json:"semantics"`
}

func main() {
	var cfg config
	var output, cpuModel, memory string
	flag.IntVar(&cfg.Requests, "requests", 200, "logical requests per trial")
	flag.IntVar(&cfg.Concurrency, "concurrency", 10, "worker count")
	flag.DurationVar(&cfg.DependencyDelay, "dependency-delay", 100*time.Millisecond, "injected dependency latency")
	flag.DurationVar(&cfg.Timeout, "timeout", 20*time.Millisecond, "protected per-attempt timeout")
	flag.IntVar(&cfg.Attempts, "attempts", 2, "maximum attempts")
	flag.DurationVar(&cfg.Backoff, "backoff", 5*time.Millisecond, "base retry backoff")
	flag.DurationVar(&cfg.OpenDuration, "open-duration", 100*time.Millisecond, "breaker open duration")
	flag.IntVar(&cfg.Trials, "trials", 3, "number of before/after trials")
	flag.StringVar(&output, "output", "benchmark-results/resilience-fault-injection.json", "JSON result path")
	flag.StringVar(&cpuModel, "cpu-model", "unknown", "benchmark CPU model")
	flag.StringVar(&memory, "memory", "unknown", "benchmark memory")
	flag.Parse()
	if cfg.Requests <= 0 || cfg.Concurrency <= 0 || cfg.Trials <= 0 || cfg.Timeout <= 0 || cfg.DependencyDelay <= 0 {
		fatal(errors.New("requests, concurrency, trials and durations must be positive"))
	}

	var out report
	out.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	out.Environment.OS, out.Environment.Arch = runtime.GOOS, runtime.GOARCH
	out.Environment.GoVersion, out.Environment.LogicalCPUs = runtime.Version(), runtime.NumCPU()
	out.Environment.CPUModel, out.Environment.Memory = cpuModel, memory
	out.Workload.Description = "Always-slow dependency; business layer converts dependency failures to successful fallback responses"
	out.Workload.Config = cfg
	for i := 0; i < cfg.Trials; i++ {
		out.Before.Trials = append(out.Before.Trials, runTrial(cfg, false, i))
		out.After.Trials = append(out.After.Trials, runTrial(cfg, true, i))
	}
	out.Before.Summary = summarize(out.Before.Trials)
	out.After.Summary = summarize(out.After.Trials)
	out.Improvement.P99ReductionPercent = reduction(out.Before.Summary.P99MS, out.After.Summary.P99MS)
	out.Improvement.WallTimeReductionPercent = reduction(out.Before.Summary.WallMS, out.After.Summary.WallMS)
	out.Improvement.UpstreamCallReductionPercent = reduction(out.Before.Summary.UpstreamCalls, out.After.Summary.UpstreamCalls)
	out.Semantics = []string{
		"Before waits for every slow dependency call, then applies the same business fallback.",
		"After uses per-attempt timeout, at most two attempts, exponential backoff with jitter, a rolling-window breaker, and the same fallback.",
		"Completion/s includes fast fallback responses and is not dependency throughput.",
		"This synthetic benchmark proves failure-latency bounds and retry-storm protection; it is not a production SLA.",
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(output, append(data, '\n'), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %s\n", output)
}

func runTrial(cfg config, protected bool, trial int) trialResult {
	runtime.GC()
	startG := runtime.NumGoroutine()
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	startCPU := cpuTime()

	var calls, depErrors, fallbacks, businessErrors atomic.Int64
	latencies := make([]time.Duration, cfg.Requests)
	jobs := make(chan int)
	stopMonitor := make(chan struct{})
	var peakG atomic.Int64
	peakG.Store(int64(startG))
	var peakHeap atomic.Uint64
	peakHeap.Store(startMem.HeapAlloc)
	go monitor(stopMonitor, &peakG, &peakHeap)

	var breaker *resilience.CircuitBreaker
	if protected {
		breaker, _ = resilience.NewCircuitBreaker(resilience.BreakerConfig{
			Dependency: fmt.Sprintf("fault_injection_%d", trial), RollingWindow: time.Second,
			MinimumRequests: cfg.Concurrency, FailureThreshold: 0.5,
			OpenDuration: cfg.OpenDuration, HalfOpenProbes: 2,
		})
	}

	started := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < cfg.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				requestStart := time.Now()
				var err error
				if protected {
					_, err = resilience.Do(context.Background(), resilience.Policy{
						Dependency: "fault_injection", Operation: "slow_read", Timeout: cfg.Timeout,
						Attempts: cfg.Attempts, Backoff: cfg.Backoff, MaxBackoff: 4 * cfg.Backoff,
						RetryIf: func(err error) bool { return errors.Is(err, context.DeadlineExceeded) }, Breaker: breaker,
					}, func(ctx context.Context) (struct{}, error) {
						calls.Add(1)
						return slowCall(ctx, cfg.DependencyDelay)
					})
				} else {
					calls.Add(1)
					_, err = slowCall(context.Background(), cfg.DependencyDelay)
					if err == nil {
						err = errors.New("dependency result is unusably slow")
					}
				}
				if err != nil {
					depErrors.Add(1)
					fallbacks.Add(1) // fallback is a successful business response
				}
				latencies[idx] = time.Since(requestStart)
			}
		}()
	}
	for i := 0; i < cfg.Requests; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	elapsed := time.Since(started)
	close(stopMonitor)

	result := trialResult{
		WallMS: float64(elapsed.Microseconds()) / 1000, P50MS: percentile(latencies, 0.50),
		P95MS: percentile(latencies, 0.95), P99MS: percentile(latencies, 0.99),
		CompletionPerSec: float64(cfg.Requests) / elapsed.Seconds(), UpstreamCalls: calls.Load(),
		DependencyErrors: depErrors.Load(), Fallbacks: fallbacks.Load(), BusinessErrors: businessErrors.Load(),
		PeakGoroutines: int(peakG.Load()), EndGoroutineDelta: runtime.NumGoroutine() - startG,
		PeakHeapDeltaBytes: positiveDelta(peakHeap.Load(), startMem.HeapAlloc), ProcessCPUMS: (cpuTime() - startCPU) * 1000,
		BreakerAfterFault: "disabled",
	}
	if protected {
		result.BreakerAfterFault = stateName(breaker.State())
		time.Sleep(cfg.OpenDuration + 10*time.Millisecond)
		beforeRecovery := calls.Load()
		for i := 0; i < 2; i++ {
			_, _ = resilience.Do(context.Background(), resilience.Policy{
				Dependency: "fault_injection", Operation: "recovery_probe", Timeout: cfg.Timeout,
				Attempts: 1, Breaker: breaker,
			}, func(context.Context) (struct{}, error) { calls.Add(1); return struct{}{}, nil })
		}
		result.RecoveryProbeCalls = calls.Load() - beforeRecovery
		result.BreakerAfterProbe = stateName(breaker.State())
	}
	return result
}

func slowCall(ctx context.Context, delay time.Duration) (struct{}, error) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return struct{}{}, ctx.Err()
	case <-timer.C:
		return struct{}{}, nil
	}
}

func monitor(stop <-chan struct{}, peakG *atomic.Int64, peakHeap *atomic.Uint64) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			g := int64(runtime.NumGoroutine())
			if g > peakG.Load() {
				peakG.Store(g)
			}
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			if mem.HeapAlloc > peakHeap.Load() {
				peakHeap.Store(mem.HeapAlloc)
			}
		}
	}
}

func percentile(values []time.Duration, q float64) float64 {
	copyValues := append([]time.Duration(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	idx := int(float64(len(copyValues)-1) * q)
	return float64(copyValues[idx].Microseconds()) / 1000
}

func summarize(trials []trialResult) summary {
	var s summary
	for _, t := range trials {
		s.WallMS += t.WallMS
		s.P50MS += t.P50MS
		s.P95MS += t.P95MS
		s.P99MS += t.P99MS
		s.CompletionPerSec += t.CompletionPerSec
		s.UpstreamCalls += float64(t.UpstreamCalls)
		s.BusinessErrors += float64(t.BusinessErrors)
		s.PeakGoroutines += float64(t.PeakGoroutines)
		s.PeakHeapDeltaBytes += float64(t.PeakHeapDeltaBytes)
		s.ProcessCPUMS += t.ProcessCPUMS
	}
	n := float64(len(trials))
	s.WallMS /= n
	s.P50MS /= n
	s.P95MS /= n
	s.P99MS /= n
	s.CompletionPerSec /= n
	s.UpstreamCalls /= n
	s.BusinessErrors /= n
	s.PeakGoroutines /= n
	s.PeakHeapDeltaBytes /= n
	s.ProcessCPUMS /= n
	return s
}

func reduction(before, after float64) float64 {
	if before == 0 {
		return 0
	}
	return (before - after) / before * 100
}

func stateName(state resilience.State) string {
	switch state {
	case resilience.StateClosed:
		return "closed"
	case resilience.StateOpen:
		return "open"
	default:
		return "half-open"
	}
}

func positiveDelta(after, before uint64) uint64 {
	if after > before {
		return after - before
	}
	return 0
}

func cpuTime() float64 {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) != nil {
		return 0
	}
	return float64(usage.Utime.Sec+usage.Stime.Sec) + float64(usage.Utime.Usec+usage.Stime.Usec)/1e6
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
