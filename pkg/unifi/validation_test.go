package unifi

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestValidateHTTPMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
		errType error
	}{
		{
			name:    "valid GET method",
			method:  http.MethodGet,
			wantErr: false,
		},
		{
			name:    "valid POST method",
			method:  http.MethodPost,
			wantErr: false,
		},
		{
			name:    "valid PUT method",
			method:  http.MethodPut,
			wantErr: false,
		},
		{
			name:    "invalid DELETE method",
			method:  http.MethodDelete,
			wantErr: true,
			errType: ErrInvalidMethod,
		},
		{
			name:    "invalid PATCH method",
			method:  http.MethodPatch,
			wantErr: true,
			errType: ErrInvalidMethod,
		},
		{
			name:    "empty method",
			method:  "",
			wantErr: true,
			errType: ErrInvalidMethod,
		},
		{
			name:    "invalid custom method",
			method:  "CUSTOM",
			wantErr: true,
			errType: ErrInvalidMethod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPMethod(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHTTPMethod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("ValidateHTTPMethod() error = %v, wantErrType %v", err, tt.errType)
			}
		})
	}
}

func TestValidatePayload(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
		errType error
	}{
		{
			name:    "empty payload",
			payload: []byte{},
			wantErr: false,
		},
		{
			name:    "valid JSON object",
			payload: []byte(`{"cmd": "block", "mac": "aa:bb:cc:dd:ee:ff"}`),
			wantErr: false,
		},
		{
			name:    "valid JSON array",
			payload: []byte(`[{"id": 1}, {"id": 2}]`),
			wantErr: false,
		},
		{
			name:    "payload with null byte",
			payload: []byte("test\x00data"),
			wantErr: true,
			errType: ErrInvalidPayload,
		},
		{
			name:    "payload too large",
			payload: make([]byte, 1024*1024+1), // 1MB + 1 byte
			wantErr: true,
			errType: ErrInvalidPayload,
		},
		{
			name:    "non-JSON payload",
			payload: []byte("not json at all"),
			wantErr: true,
			errType: ErrInvalidPayload,
		},
		{
			name:    "script injection attempt",
			payload: []byte(`{"data": "<script>alert('xss')</script>"}`),
			wantErr: true,
			errType: ErrInvalidPayload,
		},
		{
			name:    "javascript injection",
			payload: []byte(`{"url": "javascript:alert('xss')"}`),
			wantErr: true,
			errType: ErrInvalidPayload,
		},
		{
			name:    "eval injection",
			payload: []byte(`{"code": "eval('malicious code')"}`),
			wantErr: true,
			errType: ErrInvalidPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("ValidatePayload() error = %v, wantErrType %v", err, tt.errType)
			}
		})
	}
}

func TestDefaultPathValidator_ValidatePath(t *testing.T) {
	validator := NewDefaultPathValidator()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{
			name:    "valid stat path",
			path:    "/stat/device",
			wantErr: false,
		},
		{
			name:    "valid rest path",
			path:    "/rest/user",
			wantErr: false,
		},
		{
			name:    "valid rest path with ID",
			path:    "/rest/user/123456",
			wantErr: false,
		},
		{
			name:    "valid cmd path",
			path:    "/cmd/stamgr",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "path without leading slash",
			path:    "stat/device",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "path traversal attempt",
			path:    "/stat/../../../etc/passwd",
			wantErr: true,
			errType: ErrUnsafePath,
		},
		{
			name:    "URL encoded path traversal",
			path:    "/stat/%2e%2e/sensitive",
			wantErr: true,
			errType: ErrUnsafePath,
		},
		{
			name:    "double slash attack",
			path:    "/stat//device",
			wantErr: true,
			errType: ErrUnsafePath,
		},
		{
			name:    "system directory access",
			path:    "/etc/passwd",
			wantErr: true,
			errType: ErrUnsafePath,
		},
		{
			name:    "dangerous characters",
			path:    "/stat/device<script>",
			wantErr: true,
			errType: ErrUnsafePath,
		},
		{
			name:    "path too long",
			path:    "/" + strings.Repeat("a", 1000),
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "invalid API pattern",
			path:    "/invalid/api/call",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "null byte in path",
			path:    "/stat/device\x00",
			wantErr: true,
			errType: ErrUnsafePath,
		},
		{
			name:    "path with query parameters (should fail)",
			path:    "/rest/user?mac=aa:bb:cc:dd:ee:ff",
			wantErr: true,
			errType: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("ValidatePath() error = %v, wantErrType %v", err, tt.errType)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean input",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "input with null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "input with CRLF",
			input:    "hello\r\nworld",
			expected: "helloworld",
		},
		{
			name:     "input with whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "input with mixed bad characters",
			input:    " \x00hello\r\nworld\x00 ",
			expected: "helloworld",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \r\n\t  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewDefaultPathValidator(t *testing.T) {
	validator := NewDefaultPathValidator()

	if validator == nil {
		t.Fatal("NewDefaultPathValidator() returned nil")
	}

	if len(validator.AllowedPaths) == 0 {
		t.Error("NewDefaultPathValidator() created validator with no allowed paths")
	}

	if len(validator.DeniedPaths) == 0 {
		t.Error("NewDefaultPathValidator() created validator with no denied paths")
	}
}

func TestPathValidatorInterface(t *testing.T) {
	// Ensure DefaultPathValidator implements PathValidator interface
	var _ PathValidator = (*DefaultPathValidator)(nil)
}

// Benchmark tests for performance validation
func BenchmarkValidateHTTPMethod(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateHTTPMethod(http.MethodGet)
	}
}

func BenchmarkValidatePayload(b *testing.B) {
	payload := []byte(`{"cmd": "block", "mac": "aa:bb:cc:dd:ee:ff"}`)
	for i := 0; i < b.N; i++ {
		ValidatePayload(payload)
	}
}

func BenchmarkValidatePath(b *testing.B) {
	validator := NewDefaultPathValidator()
	path := "/stat/device"
	for i := 0; i < b.N; i++ {
		validator.ValidatePath(path)
	}
}

func BenchmarkSanitizeInput(b *testing.B) {
	input := "  hello world\r\n  "
	for i := 0; i < b.N; i++ {
		SanitizeInput(input)
	}
}
