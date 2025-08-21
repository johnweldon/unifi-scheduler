package unifi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// BackoffStrategy defines different backoff strategies
type BackoffStrategy int

const (
	// BackoffExponential uses exponential backoff with jitter
	BackoffExponential BackoffStrategy = iota
	// BackoffLinear uses linear backoff
	BackoffLinear
	// BackoffFixed uses fixed delay
	BackoffFixed
)

// String returns the string representation of the backoff strategy
func (s BackoffStrategy) String() string {
	switch s {
	case BackoffExponential:
		return "EXPONENTIAL"
	case BackoffLinear:
		return "LINEAR"
	case BackoffFixed:
		return "FIXED"
	default:
		return "UNKNOWN"
	}
}

// BackoffConfig holds the configuration for retry backoff logic
type BackoffConfig struct {
	// Strategy defines the backoff strategy to use
	Strategy BackoffStrategy
	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// Multiplier is used for exponential and linear backoff (default: 2.0 for exponential)
	Multiplier float64
	// Jitter adds randomness to prevent thundering herd (0.0 = no jitter, 1.0 = full jitter)
	Jitter float64
	// RetryableErrors is a list of errors that should trigger retries
	RetryableErrors []error
}

// DefaultBackoffConfig returns a sensible default configuration for exponential backoff
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		Strategy:     BackoffExponential,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		MaxRetries:   5,
		Multiplier:   2.0,
		Jitter:       0.1, // 10% jitter
		RetryableErrors: []error{
			ErrCircuitBreakerOpen,
		},
	}
}

// BackoffRetry implements retry logic with configurable backoff strategies
type BackoffRetry struct {
	config BackoffConfig
	rand   *rand.Rand
}

// NewBackoffRetry creates a new backoff retry instance
func NewBackoffRetry(config BackoffConfig) *BackoffRetry {
	// Validate and set defaults
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 5
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.Jitter < 0 || config.Jitter > 1 {
		config.Jitter = 0.1
	}

	return &BackoffRetry{
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RetryOperation represents an operation that can be retried
type RetryOperation func() error

// Retry executes the given operation with backoff retry logic
func (br *BackoffRetry) Retry(ctx context.Context, operation RetryOperation) error {
	var lastErr error

	for attempt := 0; attempt <= br.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled by context: %w", ctx.Err())
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry this error
		if !br.shouldRetry(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// If this is the last attempt, don't wait
		if attempt == br.config.MaxRetries {
			break
		}

		// Calculate backoff delay
		delay := br.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded, last error: %w", br.config.MaxRetries, lastErr)
}

// RetryWithCallback executes the operation with retry logic and calls the callback on each attempt
func (br *BackoffRetry) RetryWithCallback(ctx context.Context, operation RetryOperation, callback func(attempt int, err error, delay time.Duration)) error {
	var lastErr error

	for attempt := 0; attempt <= br.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled by context: %w", ctx.Err())
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			if callback != nil {
				callback(attempt, nil, 0)
			}
			return nil // Success
		}

		lastErr = err

		// Check if we should retry this error
		if !br.shouldRetry(err) {
			if callback != nil {
				callback(attempt, err, 0)
			}
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// If this is the last attempt, don't wait
		if attempt == br.config.MaxRetries {
			if callback != nil {
				callback(attempt, err, 0)
			}
			break
		}

		// Calculate backoff delay
		delay := br.calculateDelay(attempt)

		// Call callback with attempt info
		if callback != nil {
			callback(attempt, err, delay)
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded, last error: %w", br.config.MaxRetries, lastErr)
}

// shouldRetry determines if an error should trigger a retry
func (br *BackoffRetry) shouldRetry(err error) bool {
	// If no specific retryable errors are configured, retry all errors
	if len(br.config.RetryableErrors) == 0 {
		return true
	}

	// Check if the error matches any of the configured retryable errors
	for _, retryableErr := range br.config.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay for the given attempt number
func (br *BackoffRetry) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch br.config.Strategy {
	case BackoffExponential:
		// Exponential backoff: initialDelay * (multiplier ^ attempt)
		delay = time.Duration(float64(br.config.InitialDelay) * math.Pow(br.config.Multiplier, float64(attempt)))

	case BackoffLinear:
		// Linear backoff: initialDelay * (1 + multiplier * attempt)
		delay = time.Duration(float64(br.config.InitialDelay) * (1 + br.config.Multiplier*float64(attempt)))

	case BackoffFixed:
		// Fixed delay
		delay = br.config.InitialDelay

	default:
		delay = br.config.InitialDelay
	}

	// Apply maximum delay limit
	if delay > br.config.MaxDelay {
		delay = br.config.MaxDelay
	}

	// Apply jitter if configured
	if br.config.Jitter > 0 {
		jitterRange := float64(delay) * br.config.Jitter
		jitter := br.rand.Float64() * jitterRange
		delay = time.Duration(float64(delay) + jitter - jitterRange/2)
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = br.config.InitialDelay
	}

	return delay
}

// Stats returns statistics about the backoff configuration
func (br *BackoffRetry) Stats() BackoffStats {
	return BackoffStats{
		Strategy:        br.config.Strategy,
		InitialDelay:    br.config.InitialDelay,
		MaxDelay:        br.config.MaxDelay,
		MaxRetries:      br.config.MaxRetries,
		Multiplier:      br.config.Multiplier,
		Jitter:          br.config.Jitter,
		RetryableErrors: len(br.config.RetryableErrors),
	}
}

// BackoffStats holds statistics about the backoff configuration
type BackoffStats struct {
	Strategy        BackoffStrategy
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	MaxRetries      int
	Multiplier      float64
	Jitter          float64
	RetryableErrors int
}

// String returns a human-readable representation of the backoff stats
func (bs BackoffStats) String() string {
	return fmt.Sprintf("Strategy: %s, Initial: %v, Max: %v, Retries: %d, Multiplier: %.1f, Jitter: %.1f%%, Retryable Errors: %d",
		bs.Strategy, bs.InitialDelay, bs.MaxDelay, bs.MaxRetries, bs.Multiplier, bs.Jitter*100, bs.RetryableErrors)
}

// Common retryable error types for HTTP operations
var (
	// ErrRetryableHTTP represents HTTP errors that should be retried
	ErrRetryableHTTP = errors.New("retryable HTTP error")
	// ErrTimeout represents timeout errors that should be retried
	ErrTimeout = errors.New("timeout error")
	// ErrNetworkUnavailable represents network unavailability errors
	ErrNetworkUnavailable = errors.New("network unavailable")
)

// IsRetryableHTTPError checks if an HTTP status code indicates a retryable error
func IsRetryableHTTPError(statusCode int) bool {
	// Retry on server errors (5xx) and specific client errors
	switch statusCode {
	case 408, // Request Timeout
		429, // Too Many Requests
		500, // Internal Server Error
		502, // Bad Gateway
		503, // Service Unavailable
		504: // Gateway Timeout
		return true
	default:
		return statusCode >= 500 // All 5xx errors
	}
}

// IsRetryableNetworkError checks if a network error indicates a retryable condition
func IsRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context timeout or cancellation
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// Check for network-related errors by examining the error string
	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no route to host",
		"broken pipe",
		"eof",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryable) {
			return true
		}
	}

	return false
}

// HTTPRetryConfig creates a backoff configuration optimized for HTTP operations
func HTTPRetryConfig() BackoffConfig {
	config := DefaultBackoffConfig()
	config.InitialDelay = 200 * time.Millisecond
	config.MaxDelay = 60 * time.Second
	config.MaxRetries = 3
	config.Multiplier = 2.0
	config.Jitter = 0.2 // 20% jitter for HTTP operations
	config.RetryableErrors = []error{
		ErrCircuitBreakerOpen,
		ErrRetryableHTTP,
		ErrTimeout,
		ErrNetworkUnavailable,
	}
	return config
}

// UniFiRetryConfig creates a backoff configuration optimized for UniFi API operations
func UniFiRetryConfig() BackoffConfig {
	config := HTTPRetryConfig()
	config.InitialDelay = 500 * time.Millisecond
	config.MaxDelay = 30 * time.Second
	config.MaxRetries = 4
	config.Multiplier = 1.5 // More conservative for UniFi API
	config.Jitter = 0.15    // 15% jitter
	return config
}
