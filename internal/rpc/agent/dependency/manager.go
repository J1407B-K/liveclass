package dependency

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/resilience"
	"liveclass/internal/rpc/agent/config"
	agenttrace "liveclass/internal/rpc/agent/trace"
	"net"
	"net/http"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	MainLLM        = "main_llm"
	AdvisorLLM     = "advisor_llm"
	ProfileLLM     = "profile_llm"
	Embedding      = "embedding"
	Qdrant         = "qdrant"
	Elasticsearch  = "elasticsearch"
	Reranker       = "reranker"
	WebSearch      = "web_search"
	PostgresRead   = "postgres_read"
	PostgresWrite  = "postgres_write"
	InternalRPC    = "internal_rpc"
	LongTermMemory = "long_term_memory"
	Profile        = "profile"
)

type Manager struct {
	mu       sync.RWMutex
	policies map[string]config.DependencyPolicyConfig
	breakers map[string]*resilience.CircuitBreaker
}

var Default = &Manager{policies: make(map[string]config.DependencyPolicyConfig), breakers: make(map[string]*resilience.CircuitBreaker)}

type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("dependency returned HTTP %d: %s", e.StatusCode, e.Body)
}

func Configure(cfg config.ResilienceConfig) error {
	return Default.Configure(cfg)
}

func (m *Manager) Configure(cfg config.ResilienceConfig) error {
	policies := map[string]config.DependencyPolicyConfig{
		MainLLM: cfg.MainLLM, AdvisorLLM: cfg.AdvisorLLM, ProfileLLM: cfg.ProfileLLM, Embedding: cfg.Embedding,
		Qdrant: cfg.Qdrant, Elasticsearch: cfg.Elasticsearch, Reranker: cfg.Reranker,
		WebSearch: cfg.WebSearch, PostgresRead: cfg.PostgresRead,
		PostgresWrite: cfg.PostgresWrite, InternalRPC: cfg.InternalRPC,
	}
	breakers := make(map[string]*resilience.CircuitBreaker)
	for name, policy := range policies {
		if policy.Timeout <= 0 || policy.Attempts <= 0 {
			return fmt.Errorf("invalid resilience policy for %s", name)
		}
		if !policy.Breaker.Enabled {
			continue
		}
		breaker, err := resilience.NewCircuitBreaker(resilience.BreakerConfig{
			Dependency: name, RollingWindow: policy.Breaker.RollingWindow,
			MinimumRequests: policy.Breaker.MinimumRequests, FailureThreshold: policy.Breaker.FailureThreshold,
			OpenDuration: policy.Breaker.OpenDuration, HalfOpenProbes: policy.Breaker.HalfOpenProbes,
		})
		if err != nil {
			return fmt.Errorf("configure %s breaker: %w", name, err)
		}
		breakers[name] = breaker
	}
	m.mu.Lock()
	m.policies = policies
	m.breakers = breakers
	m.mu.Unlock()
	return nil
}

func Do[T any](ctx context.Context, name, operation string, call func(context.Context) (T, error)) (T, error) {
	Default.mu.RLock()
	cfg, ok := Default.policies[name]
	breaker := Default.breakers[name]
	Default.mu.RUnlock()
	if !ok {
		var zero T
		return zero, fmt.Errorf("resilience policy %s is not configured", name)
	}
	return resilience.Do(ctx, resilience.Policy{
		Dependency: name, Operation: operation, Timeout: cfg.Timeout,
		Attempts: cfg.Attempts, Backoff: cfg.Backoff, MaxBackoff: cfg.MaxBackoff,
		RetryIf: IsTransient, Breaker: breaker,
	}, call)
}

func Fallback(name, operation string) {
	resilience.RecordFallback(name, operation)
}

func FallbackContext(ctx context.Context, name, operation string) {
	Fallback(name, operation)
	if run := agenttrace.FromContext(ctx); run != nil {
		run.Record(ctx, "fallback", name, "used", 0, map[string]any{"operation": operation})
	}
}

func IsTransient(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusTooManyRequests || httpErr.StatusCode >= 500
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
