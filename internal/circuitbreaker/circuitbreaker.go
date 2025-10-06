package circuitbreaker

import (
	"fmt"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"github.com/zakirkun/isekai/internal/metrics"
	"github.com/zakirkun/isekai/pkg/logger"
)

// CircuitBreaker manages circuit breakers for different targets
type CircuitBreaker struct {
	breakers map[string]*gobreaker.CircuitBreaker
	mu       sync.RWMutex
	settings gobreaker.Settings
	log      *logger.Logger
	metrics  *metrics.Metrics
}

// New creates a new circuit breaker manager
func New(log *logger.Logger, metrics *metrics.Metrics) *CircuitBreaker {
	return &CircuitBreaker{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		settings: gobreaker.Settings{
			Name:        "DefaultCircuitBreaker",
			MaxRequests: 3,
			Interval:    time.Second * 10,
			Timeout:     time.Second * 60,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 3 && failureRatio >= 0.6
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				log.Infof("Circuit breaker '%s' state changed from %s to %s", name, from, to)
			},
		},
		log:     log,
		metrics: metrics,
	}
}

// GetBreaker returns or creates a circuit breaker for the target
func (cb *CircuitBreaker) GetBreaker(target string) *gobreaker.CircuitBreaker {
	cb.mu.RLock()
	breaker, exists := cb.breakers[target]
	cb.mu.RUnlock()

	if exists {
		return breaker
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := cb.breakers[target]; exists {
		return breaker
	}

	settings := cb.settings
	settings.Name = target
	settings.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
		cb.log.Infof("Circuit breaker '%s' state changed from %s to %s", name, from, to)

		// Update metrics
		var stateValue float64
		switch to {
		case gobreaker.StateClosed:
			stateValue = 0
		case gobreaker.StateHalfOpen:
			stateValue = 1
		case gobreaker.StateOpen:
			stateValue = 2
		}
		if cb.metrics != nil {
			cb.metrics.CircuitBreakerState.WithLabelValues(name).Set(stateValue)
		}
	}

	breaker = gobreaker.NewCircuitBreaker(settings)
	cb.breakers[target] = breaker

	return breaker
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(target string, fn func() (interface{}, error)) (interface{}, error) {
	breaker := cb.GetBreaker(target)
	result, err := breaker.Execute(fn)

	if err != nil {
		if err == gobreaker.ErrOpenState {
			cb.log.Warnf("Circuit breaker '%s' is open", target)
		}
		return nil, fmt.Errorf("circuit breaker error for %s: %w", target, err)
	}

	return result, nil
}

// GetState returns the current state of a circuit breaker
func (cb *CircuitBreaker) GetState(target string) gobreaker.State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	breaker, exists := cb.breakers[target]
	if !exists {
		return gobreaker.StateClosed
	}

	return breaker.State()
}

// GetAllStates returns the states of all circuit breakers
func (cb *CircuitBreaker) GetAllStates() map[string]gobreaker.State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	states := make(map[string]gobreaker.State)
	for name, breaker := range cb.breakers {
		states[name] = breaker.State()
	}

	return states
}
