package unifi

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialManager_GetCredentials(t *testing.T) {
	tests := []struct {
		name     string
		sources  []CredentialSource
		wantErr  bool
		wantUser string
		errType  error
	}{
		{
			name:    "no sources",
			sources: []CredentialSource{},
			wantErr: true,
			errType: ErrNoCredentialsFound,
		},
		{
			name: "successful first source",
			sources: []CredentialSource{
				&mockCredentialSource{username: "test", password: "pass", err: nil},
			},
			wantErr:  false,
			wantUser: "test",
		},
		{
			name: "fallback to second source",
			sources: []CredentialSource{
				&mockCredentialSource{err: errors.New("first failed")},
				&mockCredentialSource{username: "test2", password: "pass2", err: nil},
			},
			wantErr:  false,
			wantUser: "test2",
		},
		{
			name: "all sources fail",
			sources: []CredentialSource{
				&mockCredentialSource{err: errors.New("first failed")},
				&mockCredentialSource{err: errors.New("second failed")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewCredentialManager(false)
			for _, source := range tt.sources {
				cm.AddSource(source)
			}

			creds, err := cm.GetCredentials()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("GetCredentials() error = %v, wantErrType %v", err, tt.errType)
			}

			if !tt.wantErr && creds != nil {
				username := creds.Username
				if username != tt.wantUser {
					t.Errorf("GetCredentials() username = %v, want %v", username, tt.wantUser)
				}
			}
		})
	}
}

func TestStdinCredentialSource_GetCredentials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantUser string
	}{
		{
			name:     "valid credentials",
			input:    "testuser\ntestpass\n",
			wantErr:  false,
			wantUser: "testuser",
		},
		{
			name:    "empty username",
			input:   "\ntestpass\n",
			wantErr: true,
		},
		{
			name:    "empty password",
			input:   "testuser\n\n",
			wantErr: true,
		},
		{
			name:     "credentials with whitespace",
			input:    "  testuser  \n  testpass  \n",
			wantErr:  false,
			wantUser: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}
			source := NewStdinCredentialSource(input, output, false)

			creds, err := source.GetCredentials()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && creds != nil {
				username := creds.Username
				if username != tt.wantUser {
					t.Errorf("GetCredentials() username = %v, want %v", username, tt.wantUser)
				}
			}
		})
	}
}

func TestFileCredentialSource_GetCredentials(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "credential_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create valid credential file
	validFile := filepath.Join(tmpDir, "valid.json")
	validCreds := CredentialFile{
		Username: "testuser",
		Password: "testpass",
		Endpoint: "https://controller",
	}
	validData, _ := json.Marshal(validCreds)
	if err := os.WriteFile(validFile, validData, 0o600); err != nil {
		t.Fatalf("Failed to create valid credential file: %v", err)
	}

	// Create invalid credential file
	invalidFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFile, []byte(`{"username": "test"}`), 0o600); err != nil {
		t.Fatalf("Failed to create invalid credential file: %v", err)
	}

	// Create malformed JSON file
	malformedFile := filepath.Join(tmpDir, "malformed.json")
	if err := os.WriteFile(malformedFile, []byte(`{username: test`), 0o600); err != nil {
		t.Fatalf("Failed to create malformed credential file: %v", err)
	}

	tests := []struct {
		name     string
		filepath string
		wantErr  bool
		wantUser string
		errType  error
	}{
		{
			name:     "valid credential file",
			filepath: validFile,
			wantErr:  false,
			wantUser: "testuser",
		},
		{
			name:     "missing password in file",
			filepath: invalidFile,
			wantErr:  true,
			errType:  ErrInvalidCredentialFile,
		},
		{
			name:     "malformed JSON",
			filepath: malformedFile,
			wantErr:  true,
			errType:  ErrInvalidCredentialFile,
		},
		{
			name:     "non-existent file",
			filepath: "/nonexistent/file.json",
			wantErr:  true,
		},
		{
			name:     "empty file path",
			filepath: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewFileCredentialSource(tt.filepath)
			creds, err := source.GetCredentials()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("GetCredentials() error = %v, wantErrType %v", err, tt.errType)
			}

			if !tt.wantErr && creds != nil {
				username := creds.Username
				if username != tt.wantUser {
					t.Errorf("GetCredentials() username = %v, want %v", username, tt.wantUser)
				}
			}
		})
	}
}

func TestEnvironmentCredentialSource_GetCredentials(t *testing.T) {
	tests := []struct {
		name        string
		usernameVar string
		passwordVar string
		username    string
		password    string
		wantErr     bool
		wantUser    string
	}{
		{
			name:        "valid environment variables",
			usernameVar: "TEST_USERNAME",
			passwordVar: "TEST_PASSWORD",
			username:    "testuser",
			password:    "testpass",
			wantErr:     false,
			wantUser:    "testuser",
		},
		{
			name:        "empty username",
			usernameVar: "TEST_USERNAME",
			passwordVar: "TEST_PASSWORD",
			username:    "",
			password:    "testpass",
			wantErr:     true,
		},
		{
			name:        "empty password",
			usernameVar: "TEST_USERNAME",
			passwordVar: "TEST_PASSWORD",
			username:    "testuser",
			password:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv(tt.usernameVar, tt.username)
			os.Setenv(tt.passwordVar, tt.password)
			defer func() {
				os.Unsetenv(tt.usernameVar)
				os.Unsetenv(tt.passwordVar)
			}()

			source := NewEnvironmentCredentialSource(tt.usernameVar, tt.passwordVar)
			creds, err := source.GetCredentials()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && creds != nil {
				username := creds.Username
				if username != tt.wantUser {
					t.Errorf("GetCredentials() username = %v, want %v", username, tt.wantUser)
				}
			}
		})
	}
}

func TestCreateSecureCredentialFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "credential_create_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_creds.json")

	err = CreateSecureCredentialFile(testFile, "testuser", "testpass", "https://controller")
	if err != nil {
		t.Errorf("CreateSecureCredentialFile() error = %v", err)
		return
	}

	// Verify file exists and has correct permissions
	stat, err := os.Stat(testFile)
	if err != nil {
		t.Errorf("Failed to stat created file: %v", err)
		return
	}

	// Check permissions (should be 0600)
	if stat.Mode().Perm() != 0o600 {
		t.Errorf("File permissions = %o, want 0600", stat.Mode().Perm())
	}

	// Verify file content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
		return
	}

	var credFile CredentialFile
	if err := json.Unmarshal(data, &credFile); err != nil {
		t.Errorf("Failed to unmarshal created file: %v", err)
		return
	}

	if credFile.Username != "testuser" {
		t.Errorf("Username in file = %v, want %v", credFile.Username, "testuser")
	}
	if credFile.Password != "testpass" {
		t.Errorf("Password in file = %v, want %v", credFile.Password, "testpass")
	}
	if credFile.Endpoint != "https://controller" {
		t.Errorf("Endpoint in file = %v, want %v", credFile.Endpoint, "https://controller")
	}
}

func TestKeychainCredentialSource_String(t *testing.T) {
	source := NewKeychainCredentialSource("test-service", "test-account")
	expected := "keychain (service: test-service, account: test-account)"
	if source.String() != expected {
		t.Errorf("String() = %v, want %v", source.String(), expected)
	}
}

func TestStdinCredentialSource_String(t *testing.T) {
	source := NewStdinCredentialSource(nil, nil, false)
	expected := "stdin"
	if source.String() != expected {
		t.Errorf("String() = %v, want %v", source.String(), expected)
	}
}

func TestFileCredentialSource_String(t *testing.T) {
	source := NewFileCredentialSource("/path/to/file.json")
	expected := "file: /path/to/file.json"
	if source.String() != expected {
		t.Errorf("String() = %v, want %v", source.String(), expected)
	}
}

func TestEnvironmentCredentialSource_String(t *testing.T) {
	source := NewEnvironmentCredentialSource("USER_VAR", "PASS_VAR")
	expected := "environment (USER_VAR, PASS_VAR)"
	if source.String() != expected {
		t.Errorf("String() = %v, want %v", source.String(), expected)
	}
}

// Mock credential source for testing
type mockCredentialSource struct {
	username string
	password string
	err      error
}

func (m *mockCredentialSource) GetCredentials() (*Credentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	return NewCredentials(m.username, m.password)
}

func (m *mockCredentialSource) String() string {
	return "mock"
}

// Benchmark tests
func BenchmarkStdinCredentialSource_GetCredentials(b *testing.B) {
	input := strings.NewReader("testuser\ntestpass\n")
	source := NewStdinCredentialSource(input, &bytes.Buffer{}, false)

	for i := 0; i < b.N; i++ {
		input.Seek(0, 0) // Reset reader
		source.GetCredentials()
	}
}

func BenchmarkFileCredentialSource_GetCredentials(b *testing.B) {
	// Create temporary credential file
	tmpFile, err := os.CreateTemp("", "bench_creds.json")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	credFile := CredentialFile{Username: "testuser", Password: "testpass"}
	data, _ := json.Marshal(credFile)
	tmpFile.Write(data)
	tmpFile.Close()

	source := NewFileCredentialSource(tmpFile.Name())

	for i := 0; i < b.N; i++ {
		source.GetCredentials()
	}
}
