package nats

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	tests := []struct {
		name     string
		live     func() bool
		expected int
	}{
		{
			name:     "connected returns 200",
			live:     func() bool { return true },
			expected: http.StatusOK,
		},
		{
			name:     "disconnected returns 503",
			live:     func() bool { return false },
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "nil check returns 503",
			live:     nil,
			expected: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()

			HealthzHandler(tt.live)(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("HealthzHandler status = %d, want %d", rec.Code, tt.expected)
			}
		})
	}
}
