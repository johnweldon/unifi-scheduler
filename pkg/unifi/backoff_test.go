package unifi

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestDefaultBackoffConfig(t *testing.T) {
	config := DefaultBackoffConfig()

	if config.Strategy != BackoffExponential {
		t.Errorf("Expected strategy %v, got %v", BackoffExponential, config.Strategy)
	}

	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected initial delay %v, got %v", 100*time.Millisecond, config.InitialDelay)
	}

	if config.MaxRetries != 5 {
		t.Errorf("Expected max retries %d, got %d", 5, config.MaxRetries)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Expected multiplier %f, got %f", 2.0, config.Multiplier)
	}
}

func TestBackoffStrategy_String(t *testing.T) {
	tests := []struct {
		strategy BackoffStrategy
		expected string
	}{
		{BackoffExponential, "EXPONENTIAL"},
		{BackoffLinear, "LINEAR"},
		{BackoffFixed, "FIXED"},
		{BackoffStrategy(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.strategy.String()
			if result != tt.expected {
				t.Errorf("Strategy.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewBackoffRetry(t *testing.T) {
	config := BackoffConfig{
		Strategy:     BackoffExponential,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		MaxRetries:   3,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	retry := NewBackoffRetry(config)
	if retry == nil {
		t.Fatal("NewBackoffRetry returned nil")
	}

	if retry.config.Strategy != config.Strategy {
		t.Errorf("Expected strategy %v, got %v", config.Strategy, retry.config.Strategy)
	}
}

func TestNewBackoffRetry_Defaults(t *testing.T) {
	// Test with invalid/zero values to ensure defaults are applied
	config := BackoffConfig{
		InitialDelay: 0,
		MaxDelay:     0,
		MaxRetries:   0,
		Multiplier:   0,
		Jitter:       -1,
	}

	retry := NewBackoffRetry(config)

	if retry.config.InitialDelay <= 0 {
		t.Error("InitialDelay should have been set to default")
	}
	if retry.config.MaxDelay <= 0 {
		t.Error("MaxDelay should have been set to default")
	}
	if retry.config.MaxRetries <= 0 {
		t.Error("MaxRetries should have been set to default")
	}
	if retry.config.Multiplier <= 0 {
		t.Error("Multiplier should have been set to default")
	}
	if retry.config.Jitter < 0 || retry.config.Jitter > 1 {
		t.Error("Jitter should have been normalized to valid range")
	}
}

func TestBackoffRetry_SuccessfulOperation(t *testing.T) {
	config := DefaultBackoffConfig()
	config.MaxRetries = 2
	retry := NewBackoffRetry(config)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		return nil // Success on first attempt
	}

	ctx := context.Background()
	err := retry.Retry(ctx, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt, got %d", attemptCount)
	}
}

func TestBackoffRetry_FailureAfterRetries(t *testing.T) {
	config := DefaultBackoffConfig()
	config.MaxRetries = 2
	config.InitialDelay = 1 * time.Millisecond // Fast test
	config.RetryableErrors = []error{}         // Allow all errors to be retryable for testing
	retry := NewBackoffRetry(config)

	attemptCount := 0
	testError := errors.New("test error")
	operation := func() error {
		attemptCount++
		return testError
	}

	ctx := context.Background()
	err := retry.Retry(ctx, operation)

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if !errors.Is(err, testError) {
		t.Errorf("Expected wrapped test error, got %v", err)
	}

	expectedAttempts := config.MaxRetries + 1 // Initial attempt + retries
	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
	}
}

func TestBackoffRetry_SuccessAfterFailures(t *testing.T) {
	config := DefaultBackoffConfig()
	config.MaxRetries = 3
	config.InitialDelay = 1 * time.Millisecond // Fast test
	config.RetryableErrors = []error{}         // Allow all errors to be retryable for testing
	retry := NewBackoffRetry(config)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary failure")
		}
		return nil // Success on third attempt
	}

	ctx := context.Background()
	err := retry.Retry(ctx, operation)
	if err != nil {
		t.Errorf("Expected no error after success, got %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestBackoffRetry_ContextCancellation(t *testing.T) {
	config := DefaultBackoffConfig()
	config.MaxRetries = 5
	config.InitialDelay = 100 * time.Millisecond
	config.RetryableErrors = []error{} // Allow all errors to be retryable for testing
	retry := NewBackoffRetry(config)

	operation := func() error {
		return errors.New("always fails")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := retry.Retry(ctx, operation)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded error, got %v", err)
	}
}

func TestBackoffRetry_NonRetryableError(t *testing.T) {
	config := DefaultBackoffConfig()
	config.RetryableErrors = []error{ErrCircuitBreakerOpen}
	retry := NewBackoffRetry(config)

	nonRetryableError := errors.New("non-retryable error")
	attemptCount := 0
	operation := func() error {
		attemptCount++
		return nonRetryableError
	}

	ctx := context.Background()
	err := retry.Retry(ctx, operation)

	if err == nil {
		t.Error("Expected error for non-retryable error")
	}

	if !errors.Is(err, nonRetryableError) {
		t.Errorf("Expected wrapped non-retryable error, got %v", err)
	}

	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt, got %d", attemptCount)
	}
}

func TestBackoffRetry_RetryableError(t *testing.T) {
	config := DefaultBackoffConfig()
	config.MaxRetries = 2
	config.InitialDelay = 1 * time.Millisecond
	config.RetryableErrors = []error{ErrCircuitBreakerOpen}
	retry := NewBackoffRetry(config)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		return ErrCircuitBreakerOpen
	}

	ctx := context.Background()
	err := retry.Retry(ctx, operation)

	if err == nil {
		t.Error("Expected error after max retries")
	}

	expectedAttempts := config.MaxRetries + 1
	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
	}
}

func TestBackoffRetry_WithCallback(t *testing.T) {
	config := DefaultBackoffConfig()
	config.MaxRetries = 2
	config.InitialDelay = 1 * time.Millisecond
	config.RetryableErrors = []error{} // Allow all errors to be retryable for testing
	retry := NewBackoffRetry(config)

	attemptCount := 0
	callbackCount := 0
	var callbackAttempts []int
	var callbackErrors []error

	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	callback := func(attempt int, err error, delay time.Duration) {
		callbackCount++
		callbackAttempts = append(callbackAttempts, attempt)
		callbackErrors = append(callbackErrors, err)
	}

	ctx := context.Background()
	err := retry.RetryWithCallback(ctx, operation, callback)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callbackCount != 3 {
		t.Errorf("Expected 3 callback calls, got %d", callbackCount)
	}

	// Check that callback was called for each attempt
	if len(callbackAttempts) != 3 {
		t.Errorf("Expected 3 callback attempts, got %d", len(callbackAttempts))
	}

	// Last callback should have nil error (success)
	if callbackErrors[len(callbackErrors)-1] != nil {
		t.Error("Expected last callback to have nil error")
	}
}

func TestCalculateDelay_ExponentialBackoff(t *testing.T) {
	config := BackoffConfig{
		Strategy:     BackoffExponential,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0, // No jitter for predictable testing
	}
	retry := NewBackoffRetry(config)

	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
		description string
	}{
		{0, 100 * time.Millisecond, 100 * time.Millisecond, "first attempt"},
		{1, 200 * time.Millisecond, 200 * time.Millisecond, "second attempt"},
		{2, 400 * time.Millisecond, 400 * time.Millisecond, "third attempt"},
		{3, 800 * time.Millisecond, 800 * time.Millisecond, "fourth attempt"},
		{10, 10 * time.Second, 10 * time.Second, "capped at max delay"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			delay := retry.calculateDelay(tt.attempt)
			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("Attempt %d: expected delay between %v and %v, got %v",
					tt.attempt, tt.expectedMin, tt.expectedMax, delay)
			}
		})
	}
}

func TestCalculateDelay_LinearBackoff(t *testing.T) {
	config := BackoffConfig{
		Strategy:     BackoffLinear,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.0,
		Jitter:       0, // No jitter for predictable testing
	}
	retry := NewBackoffRetry(config)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond}, // 100 * (1 + 1.0 * 0)
		{1, 200 * time.Millisecond}, // 100 * (1 + 1.0 * 1)
		{2, 300 * time.Millisecond}, // 100 * (1 + 1.0 * 2)
		{3, 400 * time.Millisecond}, // 100 * (1 + 1.0 * 3)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := retry.calculateDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("Attempt %d: expected delay %v, got %v", tt.attempt, tt.expected, delay)
			}
		})
	}
}

func TestCalculateDelay_FixedBackoff(t *testing.T) {
	config := BackoffConfig{
		Strategy:     BackoffFixed,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Jitter:       0, // No jitter for predictable testing
	}
	retry := NewBackoffRetry(config)

	for attempt := 0; attempt < 5; attempt++ {
		delay := retry.calculateDelay(attempt)
		if delay != config.InitialDelay {
			t.Errorf("Attempt %d: expected fixed delay %v, got %v", attempt, config.InitialDelay, delay)
		}
	}
}

func TestCalculateDelay_WithJitter(t *testing.T) {
	config := BackoffConfig{
		Strategy:     BackoffExponential,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.5, // 50% jitter
	}
	retry := NewBackoffRetry(config)

	// Test multiple times to ensure jitter varies
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = retry.calculateDelay(1) // Second attempt
	}

	// Check that not all delays are the same (jitter is working)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Expected delays to vary due to jitter, but all were the same")
	}

	// Check that delays are within reasonable bounds
	baseDelay := 200 * time.Millisecond // 100ms * 2^1
	jitterRange := float64(baseDelay) * config.Jitter
	minDelay := time.Duration(float64(baseDelay) - jitterRange/2)
	maxDelay := time.Duration(float64(baseDelay) + jitterRange/2)

	for i, delay := range delays {
		if delay < minDelay*8/10 || delay > maxDelay*12/10 { // Allow some variance
			t.Errorf("Delay %d (%v) outside expected range [%v, %v]", i, delay, minDelay, maxDelay)
		}
	}
}

func TestBackoffStats(t *testing.T) {
	config := DefaultBackoffConfig()
	retry := NewBackoffRetry(config)

	stats := retry.Stats()

	if stats.Strategy != config.Strategy {
		t.Errorf("Expected strategy %v, got %v", config.Strategy, stats.Strategy)
	}

	if stats.MaxRetries != config.MaxRetries {
		t.Errorf("Expected max retries %d, got %d", config.MaxRetries, stats.MaxRetries)
	}

	if stats.RetryableErrors != len(config.RetryableErrors) {
		t.Errorf("Expected %d retryable errors, got %d", len(config.RetryableErrors), stats.RetryableErrors)
	}

	// Test string representation
	statsStr := stats.String()
	if statsStr == "" {
		t.Error("Stats string should not be empty")
	}
}

func TestIsRetryableHTTPError(t *testing.T) {
	tests := []struct {
		statusCode int
		retryable  bool
	}{
		{200, false}, // OK
		{400, false}, // Bad Request
		{401, false}, // Unauthorized
		{403, false}, // Forbidden
		{404, false}, // Not Found
		{408, true},  // Request Timeout
		{429, true},  // Too Many Requests
		{500, true},  // Internal Server Error
		{502, true},  // Bad Gateway
		{503, true},  // Service Unavailable
		{504, true},  // Gateway Timeout
		{505, true},  // HTTP Version Not Supported
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := IsRetryableHTTPError(tt.statusCode)
			if result != tt.retryable {
				t.Errorf("IsRetryableHTTPError(%d) = %v, want %v", tt.statusCode, result, tt.retryable)
			}
		})
	}
}

func TestHTTPRetryConfig(t *testing.T) {
	config := HTTPRetryConfig()

	if config.Strategy != BackoffExponential {
		t.Errorf("Expected exponential strategy, got %v", config.Strategy)
	}

	if len(config.RetryableErrors) == 0 {
		t.Error("Expected HTTP retry config to have retryable errors")
	}

	// Check that it includes common retryable errors
	hasCircuitBreakerError := false
	hasHTTPError := false
	for _, err := range config.RetryableErrors {
		if errors.Is(err, ErrCircuitBreakerOpen) {
			hasCircuitBreakerError = true
		}
		if errors.Is(err, ErrRetryableHTTP) {
			hasHTTPError = true
		}
	}

	if !hasCircuitBreakerError {
		t.Error("Expected HTTP retry config to include circuit breaker errors")
	}

	if !hasHTTPError {
		t.Error("Expected HTTP retry config to include HTTP errors")
	}
}

func TestUniFiRetryConfig(t *testing.T) {
	config := UniFiRetryConfig()

	if config.Strategy != BackoffExponential {
		t.Errorf("Expected exponential strategy, got %v", config.Strategy)
	}

	if config.Multiplier >= 2.0 {
		t.Errorf("Expected conservative multiplier < 2.0, got %f", config.Multiplier)
	}

	if len(config.RetryableErrors) == 0 {
		t.Error("Expected UniFi retry config to have retryable errors")
	}
}

func TestIsRetryableNetworkError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil error", nil, false},
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"context canceled", context.Canceled, true},
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"connection timeout", errors.New("connection timeout"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"no route to host", errors.New("no route to host"), true},
		{"broken pipe", errors.New("broken pipe"), true},
		{"EOF error", errors.New("unexpected EOF"), true},
		{"timeout", errors.New("timeout occurred"), true},
		{"generic error", errors.New("some other error"), false},
		{"authentication error", errors.New("authentication failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableNetworkError(tt.err)
			if result != tt.retryable {
				t.Errorf("IsRetryableNetworkError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// Benchmark tests
func BenchmarkBackoffRetry_SuccessfulOperation(b *testing.B) {
	config := DefaultBackoffConfig()
	retry := NewBackoffRetry(config)

	operation := func() error {
		return nil // Always succeeds
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		retry.Retry(ctx, operation)
	}
}

func BenchmarkCalculateDelay_Exponential(b *testing.B) {
	config := DefaultBackoffConfig()
	retry := NewBackoffRetry(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retry.calculateDelay(i % 10) // Vary attempt number
	}
}

func BenchmarkCalculateDelay_WithJitter(b *testing.B) {
	config := DefaultBackoffConfig()
	config.Jitter = 0.5
	retry := NewBackoffRetry(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retry.calculateDelay(i % 10) // Vary attempt number
	}
}
