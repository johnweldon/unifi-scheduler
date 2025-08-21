package unifi

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	// CircuitBreakerClosed allows all requests to pass through
	CircuitBreakerClosed CircuitBreakerState = iota
	// CircuitBreakerOpen rejects all requests immediately
	CircuitBreakerOpen
	// CircuitBreakerHalfOpen allows a limited number of requests to test the service
	CircuitBreakerHalfOpen
)

// String returns the string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "CLOSED"
	case CircuitBreakerOpen:
		return "OPEN"
	case CircuitBreakerHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds the configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// MaxFailures is the maximum number of failures allowed before opening the circuit
	MaxFailures uint64
	// ResetTimeout is the time to wait before transitioning from Open to Half-Open
	ResetTimeout time.Duration
	// SuccessThreshold is the number of consecutive successes required in Half-Open state to close the circuit
	SuccessThreshold uint64
}

// DefaultCircuitBreakerConfig returns a sensible default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:      5,
		ResetTimeout:     30 * time.Second,
		SuccessThreshold: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     uint64
	successes    uint64
	lastFailTime time.Time
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
	}
}

// ErrCircuitBreakerOpen is returned when the circuit breaker is open
var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

// Execute wraps a function call with circuit breaker logic
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can execute the request
	if !cb.canExecute() {
		return ErrCircuitBreakerOpen
	}

	// Execute the function
	err := fn()
	// Update state based on result
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// canExecute determines if a request can be executed based on the current state
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if enough time has passed to transition to half-open
		if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
			// Transition to half-open will be done in onSuccess/onFailure
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// onSuccess handles successful function execution
func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		// Reset failure count on success
		cb.failures = 0
	case CircuitBreakerOpen:
		// Transition to half-open if timeout has passed
		if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.successes = 1
			cb.failures = 0
		}
	case CircuitBreakerHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			// Transition back to closed
			cb.state = CircuitBreakerClosed
			cb.successes = 0
			cb.failures = 0
		}
	}
}

// onFailure handles failed function execution
func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.successes = 0
	cb.lastFailTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.state = CircuitBreakerOpen
		}
	case CircuitBreakerHalfOpen:
		// Any failure in half-open state transitions back to open
		cb.state = CircuitBreakerOpen
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns the current statistics of the circuit breaker
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:        cb.state,
		Failures:     cb.failures,
		Successes:    cb.successes,
		LastFailTime: cb.lastFailTime,
	}
}

// CircuitBreakerStats holds statistics about the circuit breaker
type CircuitBreakerStats struct {
	State        CircuitBreakerState
	Failures     uint64
	Successes    uint64
	LastFailTime time.Time
}

// String returns a string representation of the circuit breaker stats
func (s CircuitBreakerStats) String() string {
	return fmt.Sprintf("State: %s, Failures: %d, Successes: %d, LastFailTime: %v",
		s.State, s.Failures, s.Successes, s.LastFailTime)
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitBreakerClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailTime = time.Time{}
}
