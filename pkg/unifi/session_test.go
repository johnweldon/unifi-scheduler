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

func TestSession_WithSite(t *testing.T) {
	session := &Session{
		Endpoint: "https://example.com",
	}

	// Set up credentials
	creds, err := NewCredentials("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}

	// Test default site behavior
	err = session.Initialize(WithCredentials(creds))
	if err != nil {
		t.Fatalf("Initialization failed: %v", err)
	}

	// Site should be empty (defaults to "default" in buildURL)
	if session.site != "" {
		t.Errorf("Expected empty site initially, got %q", session.site)
	}

	// Test custom site
	customSite := "branch-office"
	session2 := &Session{
		Endpoint: "https://example.com",
	}

	err = session2.Initialize(
		WithCredentials(creds),
		WithSite(customSite),
	)
	if err != nil {
		t.Fatalf("Initialization with custom site failed: %v", err)
	}

	if session2.site != customSite {
		t.Errorf("Expected site to be %q, got %q", customSite, session2.site)
	}

	// Test that site can be changed
	newSite := "main-office"
	session2.Initialize(WithSite(newSite))

	if session2.site != newSite {
		t.Errorf("Expected site to be updated to %q, got %q", newSite, session2.site)
	}
}

func TestSession_BuildURL_WithSite(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		site         string
		path         string
		nonUDMPro    bool
		expectedPath string
	}{
		{
			name:         "default site with UDM Pro",
			endpoint:     "https://controller.example.com",
			site:         "",
			path:         "/stat/user",
			nonUDMPro:    false,
			expectedPath: "/proxy/network/api/s/default/stat/user",
		},
		{
			name:         "custom site with UDM Pro",
			endpoint:     "https://controller.example.com",
			site:         "branch-office",
			path:         "/stat/user",
			nonUDMPro:    false,
			expectedPath: "/proxy/network/api/s/branch-office/stat/user",
		},
		{
			name:         "default site without UDM Pro",
			endpoint:     "https://controller.example.com",
			site:         "",
			path:         "/stat/user",
			nonUDMPro:    true,
			expectedPath: "/api/s/default/stat/user",
		},
		{
			name:         "custom site without UDM Pro",
			endpoint:     "https://controller.example.com",
			site:         "branch-office",
			path:         "/stat/user",
			nonUDMPro:    true,
			expectedPath: "/api/s/branch-office/stat/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				Endpoint:  tt.endpoint,
				site:      tt.site,
				nonUDMPro: tt.nonUDMPro,
			}

			url, err := session.buildURL(tt.path)
			if err != nil {
				t.Fatalf("buildURL failed: %v", err)
			}

			if url.Path != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, url.Path)
			}

			if url.Host != "controller.example.com" {
				t.Errorf("Expected host controller.example.com, got %q", url.Host)
			}
		})
	}
}
