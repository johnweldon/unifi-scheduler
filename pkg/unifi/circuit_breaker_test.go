package unifi

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:      3,
		ResetTimeout:     time.Second,
		SuccessThreshold: 2,
	})

	// Initially should be closed
	if cb.State() != CircuitBreakerClosed {
		t.Errorf("expected circuit breaker to be closed, got %v", cb.State())
	}

	// Successful execution should work
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != CircuitBreakerClosed {
		t.Errorf("expected circuit breaker to remain closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_TransitionToOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:      3,
		ResetTimeout:     time.Second,
		SuccessThreshold: 2,
	})

	// Fail 3 times to trigger open state
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return testErr
		})
		if !errors.Is(err, testErr) {
			t.Errorf("expected test error, got %v", err)
		}
	}

	// Should now be open
	if cb.State() != CircuitBreakerOpen {
		t.Errorf("expected circuit breaker to be open, got %v", cb.State())
	}

	// Next call should fail immediately
	err := cb.Execute(func() error {
		t.Error("function should not be called when circuit breaker is open")
		return nil
	})
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Errorf("expected circuit breaker open error, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:      2,
		ResetTimeout:     100 * time.Millisecond,
		SuccessThreshold: 2,
	})

	// Fail to open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return testErr
		})
	}

	// Should be open
	if cb.State() != CircuitBreakerOpen {
		t.Errorf("expected circuit breaker to be open, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// First success should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != CircuitBreakerHalfOpen {
		t.Errorf("expected circuit breaker to be half-open, got %v", cb.State())
	}

	// Another success should close the circuit
	err = cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != CircuitBreakerClosed {
		t.Errorf("expected circuit breaker to be closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:      1,
		ResetTimeout:     100 * time.Millisecond,
		SuccessThreshold: 2,
	})

	// Fail to open the circuit
	testErr := errors.New("test error")
	cb.Execute(func() error {
		return testErr
	})

	// Should be open
	if cb.State() != CircuitBreakerOpen {
		t.Errorf("expected circuit breaker to be open, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Failure in half-open should return to open
	err := cb.Execute(func() error {
		return testErr
	})
	if !errors.Is(err, testErr) {
		t.Errorf("expected test error, got %v", err)
	}

	if cb.State() != CircuitBreakerOpen {
		t.Errorf("expected circuit breaker to return to open, got %v", cb.State())
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:      2,
		ResetTimeout:     time.Second,
		SuccessThreshold: 2,
	})

	stats := cb.Stats()
	if stats.State != CircuitBreakerClosed {
		t.Errorf("expected closed state, got %v", stats.State)
	}
	if stats.Failures != 0 {
		t.Errorf("expected 0 failures, got %d", stats.Failures)
	}

	// Add some failures
	testErr := errors.New("test error")
	cb.Execute(func() error {
		return testErr
	})

	stats = cb.Stats()
	if stats.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.Failures)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:      1,
		ResetTimeout:     time.Hour, // Long timeout
		SuccessThreshold: 2,
	})

	// Fail to open the circuit
	testErr := errors.New("test error")
	cb.Execute(func() error {
		return testErr
	})

	// Should be open
	if cb.State() != CircuitBreakerOpen {
		t.Errorf("expected circuit breaker to be open, got %v", cb.State())
	}

	// Reset should close it
	cb.Reset()

	if cb.State() != CircuitBreakerClosed {
		t.Errorf("expected circuit breaker to be closed after reset, got %v", cb.State())
	}

	// Should work normally now
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error after reset, got %v", err)
	}
}

func TestCircuitBreakerStats_String(t *testing.T) {
	stats := CircuitBreakerStats{
		State:        CircuitBreakerOpen,
		Failures:     5,
		Successes:    2,
		LastFailTime: time.Now(),
	}

	str := stats.String()
	if str == "" {
		t.Error("expected non-empty string representation")
	}

	// Should contain state information
	if !contains(str, "OPEN") {
		t.Errorf("expected string to contain state, got %s", str)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) <= len(s) && s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}
