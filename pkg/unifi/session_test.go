package unifi

import (
	"os"
	"testing"
	"time"
)

func TestSession_Initialize_Idempotent(t *testing.T) {
	session := &Session{
		Endpoint: "https://example.com",
	}

	// Set up credentials
	creds, err := NewCredentials("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}

	// First initialization with custom TLS config
	customTLSConfig := NewTLSConfig(WithInsecureSkipVerify(true))

	err = session.Initialize(
		WithCredentials(creds),
		WithTLSConfig(customTLSConfig),
		WithHTTPTimeout(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}

	// Verify initial configuration
	if !session.tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true after first initialization")
	}
	if session.httpTimeout != 5*time.Minute {
		t.Errorf("Expected httpTimeout to be 5m, got %v", session.httpTimeout)
	}
	if !session.initialized {
		t.Error("Session should be marked as initialized")
	}

	// Second initialization without options (should not override)
	err = session.Initialize()
	if err != nil {
		t.Fatalf("Second initialization failed: %v", err)
	}

	// Verify configuration is preserved
	if !session.tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should still be true after second initialization")
	}
	if session.httpTimeout != 5*time.Minute {
		t.Errorf("httpTimeout should still be 5m, got %v", session.httpTimeout)
	}

	// Third initialization with new option (should apply new option)
	err = session.Initialize(WithHTTPTimeout(10 * time.Minute))
	if err != nil {
		t.Fatalf("Third initialization failed: %v", err)
	}

	// Verify new option was applied but other config preserved
	if !session.tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should still be true after third initialization")
	}
	if session.httpTimeout != 10*time.Minute {
		t.Errorf("httpTimeout should be updated to 10m, got %v", session.httpTimeout)
	}
}

func TestSession_Initialize_DefaultsOnlySetOnce(t *testing.T) {
	session := &Session{
		Endpoint: "https://example.com",
	}

	// Set up credentials
	creds, err := NewCredentials("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}

	// First initialization - should set defaults
	err = session.Initialize(WithCredentials(creds))
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}

	// Verify defaults were set
	if session.outWriter != os.Stdout {
		t.Error("Expected outWriter to be set to os.Stdout")
	}
	if session.errWriter != os.Stderr {
		t.Error("Expected errWriter to be set to os.Stderr")
	}
	if session.httpTimeout != 2*time.Minute {
		t.Errorf("Expected default httpTimeout to be 2m, got %v", session.httpTimeout)
	}
	if session.tlsConfig == nil {
		t.Error("Expected tlsConfig to be set")
	}
	if session.circuitBreaker == nil {
		t.Error("Expected circuitBreaker to be set")
	}

	// Save references to verify they don't change
	originalTLSConfig := session.tlsConfig
	originalCircuitBreaker := session.circuitBreaker

	// Second initialization - defaults should not be reset
	err = session.Initialize(WithCredentials(creds))
	if err != nil {
		t.Fatalf("Second initialization failed: %v", err)
	}

	// Verify objects are the same (not recreated)
	if session.tlsConfig != originalTLSConfig {
		t.Error("tlsConfig should not be recreated on second initialization")
	}
	if session.circuitBreaker != originalCircuitBreaker {
		t.Error("circuitBreaker should not be recreated on second initialization")
	}
}

