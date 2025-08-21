package unifi

import (
	"io"
	"regexp"
	"strings"
)

// SecureWriter wraps an io.Writer to scrub sensitive information from output
type SecureWriter struct {
	underlying io.Writer
	scrubbers  []func(string) string
}

// NewSecureWriter creates a new SecureWriter that scrubs sensitive data
func NewSecureWriter(w io.Writer) *SecureWriter {
	sw := &SecureWriter{
		underlying: w,
	}

	// Add default scrubbers
	sw.AddPasswordScrubber()
	sw.AddAuthTokenScrubber()
	sw.AddURLCredentialScrubber()

	return sw
}

// Write implements io.Writer interface with secret scrubbing
func (sw *SecureWriter) Write(p []byte) (n int, err error) {
	data := string(p)

	// Apply all scrubbers
	for _, scrubber := range sw.scrubbers {
		data = scrubber(data)
	}

	scrubbed := []byte(data)
	written, err := sw.underlying.Write(scrubbed)

	// Return the original length to maintain contract
	if err == nil {
		return len(p), nil
	}
	return written, err
}

// AddPasswordScrubber adds password scrubbing patterns
func (sw *SecureWriter) AddPasswordScrubber() {
	patterns := []*regexp.Regexp{
		// JSON password field
		regexp.MustCompile(`("password"\s*:\s*)"[^"]*"`),
		// URL encoded password
		regexp.MustCompile(`(password=)[^&\s]*`),
		// Basic auth in URLs
		regexp.MustCompile(`(://[^:]*:)[^@]*(@)`),
		// Password flags in command lines
		regexp.MustCompile(`(--password\s+|--password=)[^\s]*`),
		regexp.MustCompile(`(-p\s+)[^\s]*`),
	}

	sw.scrubbers = append(sw.scrubbers, func(data string) string {
		result := data
		for _, pattern := range patterns {
			if pattern.NumSubexp() == 2 {
				result = pattern.ReplaceAllString(result, "${1}[REDACTED]${2}")
			} else {
				result = pattern.ReplaceAllString(result, "${1}[REDACTED]")
			}
		}
		return result
	})
}

// AddAuthTokenScrubber adds authentication token scrubbing
func (sw *SecureWriter) AddAuthTokenScrubber() {
	patterns := []*regexp.Regexp{
		// Bearer tokens
		regexp.MustCompile(`(Bearer\s+)[A-Za-z0-9\-_\.]+`),
		// API keys
		regexp.MustCompile(`(api[_-]?key["\s:=]+)[A-Za-z0-9\-_]+`),
		// Session tokens
		regexp.MustCompile(`(session["\s:=]+)[A-Za-z0-9\-_]+`),
		// CSRF tokens
		regexp.MustCompile(`(csrf[_-]?token["\s:=]+)[A-Za-z0-9\-_]+`),
	}

	sw.scrubbers = append(sw.scrubbers, func(data string) string {
		result := data
		for _, pattern := range patterns {
			result = pattern.ReplaceAllStringFunc(result, func(match string) string {
				parts := pattern.FindStringSubmatch(match)
				if len(parts) > 1 {
					return parts[1] + "[REDACTED]"
				}
				return "[REDACTED]"
			})
		}
		return result
	})
}

// AddURLCredentialScrubber scrubs credentials from URLs
func (sw *SecureWriter) AddURLCredentialScrubber() {
	// Pattern for URLs with credentials: protocol://user:pass@host
	pattern := regexp.MustCompile(`(https?://)[^:]*:[^@]*(@[^/\s]*)`)

	sw.scrubbers = append(sw.scrubbers, func(data string) string {
		return pattern.ReplaceAllString(data, "${1}[REDACTED]${2}")
	})
}

// AddCustomScrubber allows adding custom scrubbing functions
func (sw *SecureWriter) AddCustomScrubber(scrubber func(string) string) {
	sw.scrubbers = append(sw.scrubbers, scrubber)
}

// ScrubString directly scrubs a string using all configured scrubbers
func (sw *SecureWriter) ScrubString(input string) string {
	result := input
	for _, scrubber := range sw.scrubbers {
		result = scrubber(result)
	}
	return result
}

// SecureLogOption creates an option to use secure logging
func SecureLogOption(w io.Writer) Option {
	return WithDbg(NewSecureWriter(w))
}

// RedactSensitiveFields scrubs common sensitive field patterns
func RedactSensitiveFields(data string) string {
	// Common sensitive field names
	sensitiveFields := []string{
		"password", "passwd", "pwd",
		"secret", "key", "token",
		"auth", "credential", "cred",
		"private", "confidential",
	}

	result := data
	for _, field := range sensitiveFields {
		// JSON field patterns
		patterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)("` + field + `"\s*:\s*)"[^"]*"`),
			regexp.MustCompile(`(?i)(` + field + `\s*[:=]\s*)[^\s,}\]]*`),
		}

		for _, pattern := range patterns {
			if pattern.NumSubexp() > 0 {
				result = pattern.ReplaceAllString(result, "${1}[REDACTED]")
			} else {
				result = pattern.ReplaceAllLiteralString(result, "[REDACTED]")
			}
		}
	}

	return result
}

// IsLikelySensitive checks if a string contains patterns that look sensitive
func IsLikelySensitive(data string) bool {
	lower := strings.ToLower(data)

	sensitivePatterns := []string{
		"password", "passwd", "secret", "key", "token",
		"auth", "credential", "private", "confidential",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

