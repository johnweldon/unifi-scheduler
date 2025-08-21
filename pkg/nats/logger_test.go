package nats

import (
	"io"
	"testing"
)

func TestLoggerImplementsWriter(t *testing.T) {
	// Test that Logger implements io.Writer interface
	var _ io.Writer = (*Logger)(nil)

	// Test that Logger struct can be created with expected fields
	logger := &Logger{
		Connection:     nil, // We can't test actual connection without setup
		LogFlags:       42,
		LogPrefix:      "TEST: ",
		PublishSubject: "test.subject",
	}

	if logger.LogFlags != 42 {
		t.Errorf("Logger.LogFlags = %d, want 42", logger.LogFlags)
	}

	if logger.LogPrefix != "TEST: " {
		t.Errorf("Logger.LogPrefix = %q, want %q", logger.LogPrefix, "TEST: ")
	}

	if logger.PublishSubject != "test.subject" {
		t.Errorf("Logger.PublishSubject = %q, want %q", logger.PublishSubject, "test.subject")
	}
}

func TestNewStdLoggerReturnsLogger(t *testing.T) {
	logger := &Logger{
		Connection:     nil,
		LogFlags:       0,
		LogPrefix:      "",
		PublishSubject: "test",
	}

	stdLogger := NewStdLogger(logger)
	if stdLogger == nil {
		t.Fatal("NewStdLogger returned nil")
	}

	// We can't test the actual logging without a real NATS connection,
	// but we can verify the logger was created successfully
}

