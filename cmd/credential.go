package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/spf13/cobra"
)

var (
	credOutputFile   string
	credEndpoint     string
	credUsername     string
	credPassword     string
	credKeychainSvc  string
	credKeychainAcct string
	credForceCreate  bool
)

// credentialCmd represents the credential command
var credentialCmd = &cobra.Command{
	Use:     "credential",
	Aliases: []string{"cred", "auth"},
	Short:   "Manage secure credential storage",
	Long: `Manage secure credential storage for UniFi controllers.

This command helps you create and manage credentials securely using various methods:
- JSON credential files with secure permissions
- System keychain/credential store integration
- Interactive credential creation`,
	Example: `  # Create a secure credential file
  unifi-scheduler credential create --file ~/.unifi-creds.json

  # Store credentials in system keychain (macOS/Linux)
  unifi-scheduler credential store-keychain --service unifi-prod --account admin

  # Create credentials interactively
  unifi-scheduler credential create --file ~/.unifi-creds.json --interactive`,
}

var credentialCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a secure credential file",
	Long: `Create a secure JSON credential file with restricted permissions (0600).

The credential file will contain username, password, and optionally the endpoint URL.
The file is created with read/write permissions only for the owner.`,
	Example: `  # Create credential file interactively
  unifi-scheduler credential create --file ~/.unifi-creds.json

  # Create with command line arguments
  unifi-scheduler credential create --file ~/.unifi-creds.json \
    --username admin --password secret --endpoint https://controller.local

  # Force overwrite existing file
  unifi-scheduler credential create --file ~/.unifi-creds.json --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if credOutputFile == "" {
			return fmt.Errorf("credential file path is required (use --file)")
		}

		// Expand home directory
		if credOutputFile[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("getting home directory: %w", err)
			}
			credOutputFile = filepath.Join(home, credOutputFile[1:])
		}

		// Check if file exists and handle overwrite
		if _, err := os.Stat(credOutputFile); err == nil && !credForceCreate {
			return fmt.Errorf("credential file already exists: %s (use --force to overwrite)", credOutputFile)
		}

		// Get credentials
		var username, password, endpoint string

		if credUsername != "" && credPassword != "" {
			// Use provided credentials
			username = credUsername
			password = credPassword
			endpoint = credEndpoint
		} else {
			// Get credentials interactively
			fmt.Fprintf(cmd.ErrOrStderr(), "Creating secure credential file: %s\n", credOutputFile)

			source := unifi.NewStdinCredentialSource(cmd.InOrStdin(), cmd.ErrOrStderr(), true)
			creds, err := source.GetCredentials()
			if err != nil {
				return fmt.Errorf("getting credentials: %w", err)
			}

			username = creds.Username
			password = creds.Password.String()

			if credEndpoint != "" {
				endpoint = credEndpoint
			} else {
				fmt.Fprint(cmd.ErrOrStderr(), "Endpoint (optional): ")
				if _, err := fmt.Fscanln(cmd.InOrStdin(), &endpoint); err != nil {
					endpoint = "" // Optional field
				}
			}
		}

		// Create the credential file
		if err := unifi.CreateSecureCredentialFile(credOutputFile, username, password, endpoint); err != nil {
			return fmt.Errorf("creating credential file: %w", err)
		}

		cmd.Printf("Credential file created successfully: %s\n", credOutputFile)
		cmd.Printf("File permissions: 0600 (owner read/write only)\n")
		return nil
	},
}

var credentialStoreKeychainCmd = &cobra.Command{
	Use:   "store-keychain",
	Short: "Store credentials in system keychain",
	Long: `Store credentials in the system keychain/credential store.

Supported platforms:
- macOS: Uses Keychain.app via 'security' command
- Linux: Uses Secret Service via 'secret-tool' command  
- Windows: Not yet implemented

The credentials are stored securely and can be retrieved later using the --keychain flag.`,
	Example: `  # Store credentials in keychain
  unifi-scheduler credential store-keychain --service unifi-prod --account admin

  # Store with custom service name
  unifi-scheduler credential store-keychain --service my-unifi --account myuser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if credKeychainSvc == "" {
			return fmt.Errorf("keychain service name is required (use --service)")
		}
		if credKeychainAcct == "" {
			return fmt.Errorf("keychain account is required (use --account)")
		}

		// Get password securely
		var password string
		if credPassword != "" {
			password = credPassword
		} else {
			fmt.Fprint(cmd.ErrOrStderr(), "Enter password: ")
			source := unifi.NewStdinCredentialSource(cmd.InOrStdin(), cmd.ErrOrStderr(), false)
			creds, err := source.GetCredentials()
			if err != nil {
				return fmt.Errorf("getting password: %w", err)
			}
			password = creds.Password.String()
		}

		// Store in keychain
		if err := unifi.StoreKeychainCredentials(credKeychainSvc, credKeychainAcct, password); err != nil {
			return fmt.Errorf("storing credentials in keychain: %w", err)
		}

		cmd.Printf("Credentials stored successfully in keychain\n")
		cmd.Printf("Service: %s\n", credKeychainSvc)
		cmd.Printf("Account: %s\n", credKeychainAcct)
		cmd.Printf("\nTo use these credentials, run commands with:\n")
		cmd.Printf("  --keychain --keychain-service %s --keychain-account %s\n", credKeychainSvc, credKeychainAcct)
		return nil
	},
}

var credentialTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test credential retrieval",
	Long: `Test credential retrieval from various sources.

This command attempts to retrieve credentials using the same logic as the main commands,
but only displays whether credentials were found (without revealing the actual values).`,
	Example: `  # Test credential file
  unifi-scheduler credential test --credential-file ~/.unifi-creds.json

  # Test keychain
  unifi-scheduler credential test --keychain --keychain-account admin

  # Test environment variables
  UNIFI_USERNAME=admin UNIFI_PASSWORD=secret unifi-scheduler credential test

  # Test stdin
  echo -e "admin\nsecret" | unifi-scheduler credential test --stdin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		foundSources := 0

		// Test command line flags
		if username != "" && password != "" {
			cmd.Printf("✓ Command line flags: credentials provided\n")
			foundSources++
		} else {
			cmd.Printf("✗ Command line flags: not provided\n")
		}

		// Test credential file
		if credentialFile != "" {
			source := unifi.NewFileCredentialSource(credentialFile)
			if _, err := source.GetCredentials(); err == nil {
				cmd.Printf("✓ Credential file (%s): valid\n", credentialFile)
				foundSources++
			} else {
				cmd.Printf("✗ Credential file (%s): %v\n", credentialFile, err)
			}
		} else {
			cmd.Printf("✗ Credential file: not specified\n")
		}

		// Test environment variables
		envSource := unifi.NewEnvironmentCredentialSource("UNIFI_USERNAME", "UNIFI_PASSWORD")
		if _, err := envSource.GetCredentials(); err == nil {
			cmd.Printf("✓ Environment variables: valid\n")
			foundSources++
		} else {
			cmd.Printf("✗ Environment variables: %v\n", err)
		}

		// Test keychain
		if useKeychain && keychainAccount != "" {
			source := unifi.NewKeychainCredentialSource(keychainService, keychainAccount)
			if _, err := source.GetCredentials(); err == nil {
				cmd.Printf("✓ Keychain (service: %s, account: %s): valid\n", keychainService, keychainAccount)
				foundSources++
			} else {
				cmd.Printf("✗ Keychain (service: %s, account: %s): %v\n", keychainService, keychainAccount, err)
			}
		} else {
			cmd.Printf("✗ Keychain: not configured\n")
		}

		// Test stdin availability
		if unifi.IsCredentialInputAvailable() {
			cmd.Printf("✓ Stdin: terminal available for secure input\n")
		} else {
			cmd.Printf("✗ Stdin: no terminal available\n")
		}

		cmd.Printf("\nSummary: %d credential source(s) available\n", foundSources)

		if foundSources == 0 {
			cmd.Printf("\nNo valid credential sources found. Available options:\n")
			cmd.Printf("  --username and --password flags\n")
			cmd.Printf("  --credential-file path/to/creds.json\n")
			cmd.Printf("  --keychain with --keychain-account\n")
			cmd.Printf("  UNIFI_USERNAME and UNIFI_PASSWORD environment variables\n")
			return fmt.Errorf("no credential sources available")
		}

		return nil
	},
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(credentialCmd)

	// Add subcommands
	credentialCmd.AddCommand(credentialCreateCmd)
	credentialCmd.AddCommand(credentialStoreKeychainCmd)
	credentialCmd.AddCommand(credentialTestCmd)

	// Create command flags
	credentialCreateCmd.Flags().StringVar(&credOutputFile, "file", "", "output credential file path")
	credentialCreateCmd.Flags().StringVar(&credUsername, "username", "", "username (if not provided, will prompt)")
	credentialCreateCmd.Flags().StringVar(&credPassword, "password", "", "password (if not provided, will prompt)")
	credentialCreateCmd.Flags().StringVar(&credEndpoint, "endpoint", "", "endpoint URL (optional)")
	credentialCreateCmd.Flags().BoolVar(&credForceCreate, "force", false, "overwrite existing file")
	_ = credentialCreateCmd.MarkFlagRequired("file")

	// Store keychain command flags
	credentialStoreKeychainCmd.Flags().StringVar(&credKeychainSvc, "service", "unifi-scheduler", "keychain service name")
	credentialStoreKeychainCmd.Flags().StringVar(&credKeychainAcct, "account", "", "keychain account/username")
	credentialStoreKeychainCmd.Flags().StringVar(&credPassword, "password", "", "password (if not provided, will prompt)")
	_ = credentialStoreKeychainCmd.MarkFlagRequired("account")

	// Test command inherits relevant flags from root
	credentialTestCmd.Flags().StringVar(&credentialFile, "credential-file", "", "path to JSON credential file")
	credentialTestCmd.Flags().BoolVar(&useKeychain, "keychain", false, "test keychain credentials")
	credentialTestCmd.Flags().StringVar(&keychainService, "keychain-service", "unifi-scheduler", "keychain service name")
	credentialTestCmd.Flags().StringVar(&keychainAccount, "keychain-account", "", "keychain account/username")
	credentialTestCmd.Flags().BoolVar(&useStdin, "stdin", false, "test stdin input")
}
