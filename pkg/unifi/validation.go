package unifi

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	// ErrInvalidPath is returned when a path fails validation
	ErrInvalidPath = errors.New("invalid path")
	// ErrInvalidMethod is returned when an HTTP method is invalid
	ErrInvalidMethod = errors.New("invalid HTTP method")
	// ErrInvalidPayload is returned when a payload fails validation
	ErrInvalidPayload = errors.New("invalid payload")
	// ErrUnsafePath is returned when a path contains potentially unsafe elements
	ErrUnsafePath = errors.New("unsafe path detected")
)

// PathValidator defines interface for path validation
type PathValidator interface {
	ValidatePath(path string) error
}

// DefaultPathValidator implements basic path validation for UniFi API
type DefaultPathValidator struct {
	// AllowedPaths contains regex patterns for allowed API paths
	AllowedPaths []*regexp.Regexp
	// DeniedPaths contains regex patterns for explicitly denied paths
	DeniedPaths []*regexp.Regexp
}

// NewDefaultPathValidator creates a new path validator with UniFi API patterns
func NewDefaultPathValidator() *DefaultPathValidator {
	// Define safe UniFi API path patterns
	allowedPatterns := []string{
		`^/stat/[a-zA-Z][a-zA-Z0-9_-]*$`,                // stat endpoints like /stat/device, /stat/sta
		`^/rest/[a-zA-Z][a-zA-Z0-9_-]*$`,                // rest endpoints like /rest/user, /rest/event
		`^/rest/[a-zA-Z][a-zA-Z0-9_-]*/[a-zA-Z0-9_-]+$`, // rest with ID like /rest/user/12345
		`^/cmd/[a-zA-Z][a-zA-Z0-9_-]*$`,                 // command endpoints like /cmd/stamgr
		`^/list/[a-zA-Z][a-zA-Z0-9_-]*$`,                // list endpoints
		`^/get/[a-zA-Z][a-zA-Z0-9_-]*$`,                 // get endpoints
		`^/set/[a-zA-Z][a-zA-Z0-9_-]*$`,                 // set endpoints
	}

	// Define dangerous path patterns to explicitly deny
	deniedPatterns := []string{
		`\.\./`,                      // Path traversal attempts
		`/\.\./`,                     // Path traversal
		`//`,                         // Double slashes
		`/etc/`,                      // System directories
		`/proc/`,                     // Process directories
		`/sys/`,                      // System directories
		`/var/`,                      // Variable directories
		`/tmp/`,                      // Temporary directories
		`/home/`,                     // Home directories
		`/root/`,                     // Root directory
		`[<>"|{}\\^` + "`" + `\[\]]`, // Dangerous characters
		`%2e%2e`,                     // URL-encoded path traversal
		`%2f`,                        // URL-encoded slash
		`%5c`,                        // URL-encoded backslash
	}

	validator := &DefaultPathValidator{
		AllowedPaths: make([]*regexp.Regexp, 0, len(allowedPatterns)),
		DeniedPaths:  make([]*regexp.Regexp, 0, len(deniedPatterns)),
	}

	// Compile allowed patterns
	for _, pattern := range allowedPatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			validator.AllowedPaths = append(validator.AllowedPaths, re)
		}
	}

	// Compile denied patterns
	for _, pattern := range deniedPatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			validator.DeniedPaths = append(validator.DeniedPaths, re)
		}
	}

	return validator
}

// ValidatePath validates an API path against security rules
func (v *DefaultPathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	// Basic length check
	if len(path) > 1000 {
		return fmt.Errorf("%w: path too long", ErrInvalidPath)
	}

	// Must start with /
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: path must start with /", ErrInvalidPath)
	}

	// Check for denied patterns first
	for _, deniedRe := range v.DeniedPaths {
		if deniedRe.MatchString(path) {
			return fmt.Errorf("%w: path matches denied pattern", ErrUnsafePath)
		}
	}

	// URL decode the path and check for path traversal
	decoded, err := url.QueryUnescape(path)
	if err != nil {
		return fmt.Errorf("%w: invalid URL encoding", ErrInvalidPath)
	}

	// Check decoded path for traversal attempts
	if strings.Contains(decoded, "../") || strings.Contains(decoded, "..\\") {
		return fmt.Errorf("%w: path traversal attempt detected", ErrUnsafePath)
	}

	// Check for null bytes
	if strings.Contains(decoded, "\x00") {
		return fmt.Errorf("%w: null byte detected", ErrUnsafePath)
	}

	// Check against allowed patterns
	pathMatched := false
	for _, allowedRe := range v.AllowedPaths {
		if allowedRe.MatchString(path) {
			pathMatched = true
			break
		}
	}

	if !pathMatched {
		return fmt.Errorf("%w: path does not match any allowed pattern", ErrInvalidPath)
	}

	return nil
}

// ValidateHTTPMethod validates an HTTP method
func ValidateHTTPMethod(method string) error {
	if method == "" {
		return fmt.Errorf("%w: empty method", ErrInvalidMethod)
	}

	// Only allow specific HTTP methods that are safe for UniFi API
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut:
		return nil
	case http.MethodDelete:
		// DELETE is potentially destructive, require explicit allowance
		return fmt.Errorf("%w: DELETE method not allowed for security", ErrInvalidMethod)
	default:
		return fmt.Errorf("%w: unsupported method %q", ErrInvalidMethod, method)
	}
}

// ValidatePayload performs basic validation on request payloads
func ValidatePayload(payload []byte) error {
	if len(payload) == 0 {
		return nil // Empty payload is allowed
	}

	// Reasonable size limit (1MB)
	if len(payload) > 1024*1024 {
		return fmt.Errorf("%w: payload too large", ErrInvalidPayload)
	}

	// Check for null bytes
	if strings.Contains(string(payload), "\x00") {
		return fmt.Errorf("%w: null byte detected in payload", ErrInvalidPayload)
	}

	// Very basic JSON structure validation - should start with { or [
	payloadStr := strings.TrimSpace(string(payload))
	if len(payloadStr) > 0 && !(strings.HasPrefix(payloadStr, "{") || strings.HasPrefix(payloadStr, "[")) {
		return fmt.Errorf("%w: payload should be JSON format", ErrInvalidPayload)
	}

	// Check for potential script injection patterns
	dangerous := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"eval(",
		"setTimeout(",
		"setInterval(",
	}

	lowerPayload := strings.ToLower(payloadStr)
	for _, danger := range dangerous {
		if strings.Contains(lowerPayload, danger) {
			return fmt.Errorf("%w: potentially dangerous content detected", ErrInvalidPayload)
		}
	}

	return nil
}

// SanitizeInput performs basic input sanitization
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove carriage returns and line feeds that could cause header injection
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\n", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}
