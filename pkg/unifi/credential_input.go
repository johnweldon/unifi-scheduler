package unifi

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var (
	// ErrNoCredentialsFound is returned when no credentials can be found
	ErrNoCredentialsFound = errors.New("no credentials found")
	// ErrInvalidCredentialFile is returned when a credential file is malformed
	ErrInvalidCredentialFile = errors.New("invalid credential file")
	// ErrKeychainNotSupported is returned on unsupported platforms
	ErrKeychainNotSupported = errors.New("keychain not supported on this platform")
)

// CredentialSource defines the interface for credential sources
type CredentialSource interface {
	GetCredentials() (*Credentials, error)
	String() string
}

// CredentialManager handles multiple credential sources with fallback
type CredentialManager struct {
	sources []CredentialSource
	debug   bool
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(debug bool) *CredentialManager {
	return &CredentialManager{
		sources: make([]CredentialSource, 0),
		debug:   debug,
	}
}

// AddSource adds a credential source to the manager
func (cm *CredentialManager) AddSource(source CredentialSource) {
	cm.sources = append(cm.sources, source)
}

// GetCredentials attempts to get credentials from sources in order
func (cm *CredentialManager) GetCredentials() (*Credentials, error) {
	if len(cm.sources) == 0 {
		return nil, ErrNoCredentialsFound
	}

	var lastErr error
	for _, source := range cm.sources {
		if cm.debug {
			fmt.Fprintf(os.Stderr, "Trying credential source: %s\n", source.String())
		}

		creds, err := source.GetCredentials()
		if err != nil {
			lastErr = err
			if cm.debug {
				fmt.Fprintf(os.Stderr, "Failed to get credentials from %s: %v\n", source.String(), err)
			}
			continue
		}

		if cm.debug {
			fmt.Fprintf(os.Stderr, "Successfully obtained credentials from: %s\n", source.String())
		}
		return creds, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", lastErr)
	}
	return nil, ErrNoCredentialsFound
}

// StdinCredentialSource reads credentials from stdin
type StdinCredentialSource struct {
	input  io.Reader
	output io.Writer
	prompt bool
}

// NewStdinCredentialSource creates a new stdin credential source
func NewStdinCredentialSource(input io.Reader, output io.Writer, prompt bool) *StdinCredentialSource {
	return &StdinCredentialSource{
		input:  input,
		output: output,
		prompt: prompt,
	}
}

// GetCredentials reads username and password from stdin
func (s *StdinCredentialSource) GetCredentials() (*Credentials, error) {
	if s.input == nil {
		s.input = os.Stdin
	}
	if s.output == nil {
		s.output = os.Stderr
	}

	var username, password string
	var err error

	// Read username
	if s.prompt {
		fmt.Fprint(s.output, "Username: ")
	}
	scanner := bufio.NewScanner(s.input)
	if scanner.Scan() {
		username = strings.TrimSpace(scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading username: %w", err)
	}

	// Read password securely
	if s.prompt {
		fmt.Fprint(s.output, "Password: ")
	}

	// Try to read password securely if we're connected to a terminal
	if file, ok := s.input.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		passwordBytes, err := term.ReadPassword(int(file.Fd()))
		if err != nil {
			return nil, fmt.Errorf("reading password: %w", err)
		}
		password = string(passwordBytes)
		if s.prompt {
			fmt.Fprintln(s.output) // Add newline after password input
		}
	} else {
		// Fallback to regular scanning for non-terminal input
		if scanner.Scan() {
			password = strings.TrimSpace(scanner.Text())
		}
		if err = scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading password: %w", err)
		}
	}

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	return NewCredentials(username, password)
}

// String returns a description of this credential source
func (s *StdinCredentialSource) String() string {
	return "stdin"
}

// FileCredentialSource reads credentials from a JSON file
type FileCredentialSource struct {
	filepath string
}

// NewFileCredentialSource creates a new file credential source
func NewFileCredentialSource(filepath string) *FileCredentialSource {
	return &FileCredentialSource{filepath: filepath}
}

// CredentialFile represents the structure of a credential file
type CredentialFile struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Endpoint string `json:"endpoint,omitempty"`
}

// GetCredentials reads credentials from a JSON file
func (f *FileCredentialSource) GetCredentials() (*Credentials, error) {
	if f.filepath == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	// Expand user home directory
	if strings.HasPrefix(f.filepath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		f.filepath = filepath.Join(home, f.filepath[2:])
	}

	// Check file permissions for security
	stat, err := os.Stat(f.filepath)
	if err != nil {
		return nil, fmt.Errorf("accessing credential file: %w", err)
	}

	// On Unix-like systems, warn if file is readable by others
	if runtime.GOOS != "windows" {
		if stat.Mode().Perm()&0o077 != 0 {
			fmt.Fprintf(os.Stderr, "Warning: credential file %s has broad permissions (recommended: 0600)\n", f.filepath)
		}
	}

	// Read and parse file
	data, err := os.ReadFile(f.filepath)
	if err != nil {
		return nil, fmt.Errorf("reading credential file: %w", err)
	}

	var credFile CredentialFile
	if err := json.Unmarshal(data, &credFile); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCredentialFile, err)
	}

	if credFile.Username == "" || credFile.Password == "" {
		return nil, fmt.Errorf("%w: username and password required", ErrInvalidCredentialFile)
	}

	return NewCredentials(credFile.Username, credFile.Password)
}

// String returns a description of this credential source
func (f *FileCredentialSource) String() string {
	return fmt.Sprintf("file: %s", f.filepath)
}

// KeychainCredentialSource reads credentials from system keychain/credential store
type KeychainCredentialSource struct {
	service string
	account string
}

// NewKeychainCredentialSource creates a new keychain credential source
func NewKeychainCredentialSource(service, account string) *KeychainCredentialSource {
	return &KeychainCredentialSource{
		service: service,
		account: account,
	}
}

// GetCredentials reads credentials from system keychain
func (k *KeychainCredentialSource) GetCredentials() (*Credentials, error) {
	switch runtime.GOOS {
	case "darwin":
		return k.getMacOSCredentials()
	case "linux":
		return k.getLinuxCredentials()
	case "windows":
		return k.getWindowsCredentials()
	default:
		return nil, fmt.Errorf("%w: %s", ErrKeychainNotSupported, runtime.GOOS)
	}
}

// getMacOSCredentials reads from macOS Keychain using security command
func (k *KeychainCredentialSource) getMacOSCredentials() (*Credentials, error) {
	// Try to get password from keychain
	cmd := exec.Command("security", "find-generic-password",
		"-s", k.service,
		"-a", k.account,
		"-w") // -w outputs only the password

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("reading from macOS keychain: %w", err)
	}

	password := strings.TrimSpace(string(output))
	if password == "" {
		return nil, fmt.Errorf("empty password from keychain")
	}

	return NewCredentials(k.account, password)
}

// getLinuxCredentials reads from Linux Secret Service (libsecret/gnome-keyring)
func (k *KeychainCredentialSource) getLinuxCredentials() (*Credentials, error) {
	// Try secret-tool first (part of libsecret)
	cmd := exec.Command("secret-tool", "lookup", "service", k.service, "username", k.account)
	output, err := cmd.Output()
	if err == nil {
		password := strings.TrimSpace(string(output))
		if password != "" {
			return NewCredentials(k.account, password)
		}
	}

	return nil, fmt.Errorf("failed to read from Linux secret service")
}

// getWindowsCredentials reads from Windows Credential Manager
func (k *KeychainCredentialSource) getWindowsCredentials() (*Credentials, error) {
	// Use cmdkey command to read from Windows Credential Manager
	// Note: This is a simplified implementation
	// A production implementation might use the Windows API directly
	return nil, fmt.Errorf("Windows credential manager support not implemented")
}

// String returns a description of this credential source
func (k *KeychainCredentialSource) String() string {
	return fmt.Sprintf("keychain (service: %s, account: %s)", k.service, k.account)
}

// EnvironmentCredentialSource reads credentials from environment variables
type EnvironmentCredentialSource struct {
	usernameVar string
	passwordVar string
}

// NewEnvironmentCredentialSource creates a new environment credential source
func NewEnvironmentCredentialSource(usernameVar, passwordVar string) *EnvironmentCredentialSource {
	return &EnvironmentCredentialSource{
		usernameVar: usernameVar,
		passwordVar: passwordVar,
	}
}

// GetCredentials reads credentials from environment variables
func (e *EnvironmentCredentialSource) GetCredentials() (*Credentials, error) {
	username := os.Getenv(e.usernameVar)
	password := os.Getenv(e.passwordVar)

	if username == "" {
		return nil, fmt.Errorf("environment variable %s is empty", e.usernameVar)
	}
	if password == "" {
		return nil, fmt.Errorf("environment variable %s is empty", e.passwordVar)
	}

	return NewCredentials(username, password)
}

// String returns a description of this credential source
func (e *EnvironmentCredentialSource) String() string {
	return fmt.Sprintf("environment (%s, %s)", e.usernameVar, e.passwordVar)
}

// CreateSecureCredentialFile creates a credential file with secure permissions
func CreateSecureCredentialFile(filepath, username, password, endpoint string) error {
	credFile := CredentialFile{
		Username: username,
		Password: password,
		Endpoint: endpoint,
	}

	data, err := json.MarshalIndent(credFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	// Expand user home directory
	if strings.HasPrefix(filepath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		filepath = filepath[2:]
		filepath = home + "/" + filepath
	}

	// Create file with restrictive permissions
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating credential file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("writing credential file: %w", err)
	}

	return nil
}

// StoreKeychainCredentials stores credentials in system keychain
func StoreKeychainCredentials(service, account, password string) error {
	switch runtime.GOOS {
	case "darwin":
		return storeMacOSCredentials(service, account, password)
	case "linux":
		return storeLinuxCredentials(service, account, password)
	case "windows":
		return storeWindowsCredentials(service, account, password)
	default:
		return fmt.Errorf("%w: %s", ErrKeychainNotSupported, runtime.GOOS)
	}
}

// storeMacOSCredentials stores credentials in macOS Keychain
func storeMacOSCredentials(service, account, password string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", account,
		"-w", password,
		"-U") // -U updates if exists

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("storing to macOS keychain: %w", err)
	}
	return nil
}

// storeLinuxCredentials stores credentials in Linux Secret Service
func storeLinuxCredentials(service, account, password string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("%s (%s)", service, account),
		"service", service,
		"username", account)

	cmd.Stdin = strings.NewReader(password)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("storing to Linux secret service: %w", err)
	}
	return nil
}

// storeWindowsCredentials stores credentials in Windows Credential Manager
func storeWindowsCredentials(service, account, password string) error {
	return fmt.Errorf("Windows credential manager support not implemented")
}

// IsCredentialInputAvailable checks if secure password input is available
func IsCredentialInputAvailable() bool {
	// Check if we're connected to a terminal
	return term.IsTerminal(int(syscall.Stdin))
}
