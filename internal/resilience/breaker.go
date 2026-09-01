package resilience

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State uint8

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type BreakerConfig struct {
	Dependency       string
	RollingWindow    time.Duration
	MinimumRequests  int
	FailureThreshold float64
	OpenDuration     time.Duration
	HalfOpenProbes   int
}

type outcome struct {
	at      time.Time
	success bool
}

type CircuitBreaker struct {
	cfg BreakerConfig

	mu               sync.Mutex
	state            State
	openedAt         time.Time
	outcomes         []outcome
	halfOpenInFlight int
	halfOpenSuccess  int
	now              func() time.Time
}

type Permit struct {
	breaker *CircuitBreaker
	once    sync.Once
}

func NewCircuitBreaker(cfg BreakerConfig) (*CircuitBreaker, error) {
	if cfg.Dependency == "" {
		return nil, errors.New("breaker dependency is required")
	}
	if cfg.RollingWindow <= 0 || cfg.MinimumRequests <= 0 || cfg.OpenDuration <= 0 || cfg.HalfOpenProbes <= 0 {
		return nil, errors.New("breaker durations, minimum requests and half-open probes must be positive")
	}
	if cfg.FailureThreshold <= 0 || cfg.FailureThreshold > 1 {
		return nil, errors.New("breaker failure threshold must be in (0,1]")
	}
	b := &CircuitBreaker{cfg: cfg, now: time.Now}
	breakerState.WithLabelValues(cfg.Dependency).Set(float64(StateClosed))
	return b, nil
}

func (b *CircuitBreaker) Acquire() (*Permit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()

	if b.state == StateOpen {
		if now.Sub(b.openedAt) < b.cfg.OpenDuration {
			return nil, ErrCircuitOpen
		}
		b.state = StateHalfOpen
		b.halfOpenInFlight = 0
		b.halfOpenSuccess = 0
		breakerState.WithLabelValues(b.cfg.Dependency).Set(float64(StateHalfOpen))
	}
	if b.state == StateHalfOpen {
		if b.halfOpenInFlight >= b.cfg.HalfOpenProbes {
			return nil, ErrCircuitOpen
		}
		b.halfOpenInFlight++
	}
	return &Permit{breaker: b}, nil
}

func (p *Permit) Done(success bool) {
	if p == nil || p.breaker == nil {
		return
	}
	p.once.Do(func() { p.breaker.record(success) })
}

func (b *CircuitBreaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == StateOpen && b.now().Sub(b.openedAt) >= b.cfg.OpenDuration {
		return StateHalfOpen
	}
	return b.state
}

func (b *CircuitBreaker) record(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()

	if b.state == StateHalfOpen {
		if b.halfOpenInFlight > 0 {
			b.halfOpenInFlight--
		}
		if !success {
			b.open(now)
			return
		}
		b.halfOpenSuccess++
		if b.halfOpenSuccess >= b.cfg.HalfOpenProbes {
			b.state = StateClosed
			b.outcomes = nil
			b.halfOpenSuccess = 0
			breakerState.WithLabelValues(b.cfg.Dependency).Set(float64(StateClosed))
		}
		return
	}
	if b.state != StateClosed {
		return
	}

	b.outcomes = append(b.outcomes, outcome{at: now, success: success})
	cutoff := now.Add(-b.cfg.RollingWindow)
	first := 0
	for first < len(b.outcomes) && b.outcomes[first].at.Before(cutoff) {
		first++
	}
	if first > 0 {
		b.outcomes = append([]outcome(nil), b.outcomes[first:]...)
	}
	if len(b.outcomes) < b.cfg.MinimumRequests {
		return
	}
	failures := 0
	for _, result := range b.outcomes {
		if !result.success {
			failures++
		}
	}
	if float64(failures)/float64(len(b.outcomes)) >= b.cfg.FailureThreshold {
		b.open(now)
	}
}

func (b *CircuitBreaker) open(now time.Time) {
	b.state = StateOpen
	b.openedAt = now
	b.halfOpenInFlight = 0
	b.halfOpenSuccess = 0
	breakerOpenTotal.WithLabelValues(b.cfg.Dependency).Inc()
	breakerState.WithLabelValues(b.cfg.Dependency).Set(float64(StateOpen))
}
