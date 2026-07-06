package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestVersionCommand(t *testing.T) {
	// The root command marks --endpoint required; satisfy it hermetically so
	// the test does not depend on the local environment.
	t.Setenv("UNIFI_ENDPOINT", "https://controller.example.com")

	// Set a test version
	Version = "test-version"

	// Create a buffer to capture output
	var buf bytes.Buffer

	// Execute the version command with args
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("version command execution returned error: %v", err)
	}

	// Check the output
	output := buf.String()
	expected := "Version: test-version\n"
	if output != expected {
		t.Errorf("version command output = %q, want %q", output, expected)
	}
}

func TestRootCommandStructure(t *testing.T) {
	// Test that root command is properly configured
	if rootCmd.Use != "unifi-scheduler" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "unifi-scheduler")
	}

	// Test aliases
	expectedAliases := []string{"ucli"}
	if len(rootCmd.Aliases) != len(expectedAliases) {
		t.Errorf("len(rootCmd.Aliases) = %d, want %d", len(rootCmd.Aliases), len(expectedAliases))
	} else {
		for i, alias := range expectedAliases {
			if rootCmd.Aliases[i] != alias {
				t.Errorf("rootCmd.Aliases[%d] = %q, want %q", i, rootCmd.Aliases[i], alias)
			}
		}
	}

	// Test that version command is added
	versionSubCmd := rootCmd.Commands()
	found := false
	for _, cmd := range versionSubCmd {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("version command not found in root command")
	}
}

func TestExecuteFunction(t *testing.T) {
	// Test that Execute function sets version correctly
	testVersion := "test-execute-version"
	Version = "" // Reset version

	// This would normally call rootCmd.Execute(), but we can't test that easily
	// without mocking. Instead, we test that the version gets set.
	Execute(testVersion)

	if Version != testVersion {
		t.Errorf("Execute() set Version = %q, want %q", Version, testVersion)
	}
}

func TestPresetRequiredFlagsFunction(t *testing.T) {
	// Test the presetRequiredFlags function with a simple command
	testCmd := &cobra.Command{
		Use: "test",
	}

	// Add a simple flag
	testCmd.Flags().String("test-flag", "", "test flag")

	// Test that presetRequiredFlags doesn't return error for simple case
	err := presetRequiredFlags(testCmd)
	if err != nil {
		t.Errorf("presetRequiredFlags() returned error: %v", err)
	}
}

func TestPostInitConfigFunction(t *testing.T) {
	// Test the postInitConfig function with simple commands
	testCmd1 := &cobra.Command{Use: "test1"}
	testCmd2 := &cobra.Command{Use: "test2"}
	testCmd1.AddCommand(testCmd2)

	commands := []*cobra.Command{testCmd1}

	// Test that postInitConfig doesn't return error for simple case
	err := postInitConfig(commands)
	if err != nil {
		t.Errorf("postInitConfig() returned error: %v", err)
	}
}

// TestConfigureEnv_HyphenatedFlags verifies hyphenated flag names resolve
// from underscore-form environment variables (e.g. --tls-insecure from
// UNIFI_TLS_INSECURE).
func TestConfigureEnv_HyphenatedFlags(t *testing.T) {
	t.Setenv("UNIFI_TLS_INSECURE", "true")
	t.Setenv("UNIFI_ENDPOINT", "https://controller.example.com")

	v := viper.New()
	configureEnv(v)

	if !v.GetBool("tls-insecure") {
		t.Error("expected tls-insecure to be settable via UNIFI_TLS_INSECURE")
	}

	if got := v.GetString("endpoint"); got != "https://controller.example.com" {
		t.Errorf("expected endpoint from UNIFI_ENDPOINT, got %q", got)
	}
}

func TestFlagConstants(t *testing.T) {
	// Test that flag constants are defined correctly
	expectedFlags := map[string]string{
		usernameFlag: "username",
		passwordFlag: "password",
		endpointFlag: "endpoint",
	}

	for constant, expected := range expectedFlags {
		if constant != expected {
			t.Errorf("flag constant = %q, want %q", constant, expected)
		}
	}
}
