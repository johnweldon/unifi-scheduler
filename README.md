# UniFi Scheduler

A powerful CLI tool for managing UniFi network controllers, providing comprehensive client management, device monitoring, event tracking, and distributed operations via NATS messaging.

## Features

- **Client Management**: Block, unblock, kick, and monitor network clients
- **Device Operations**: List and monitor UniFi network devices
- **Event Monitoring**: Track network events and connection changes
- **NATS Caching**: Cache UniFi data in NATS for fast distributed access
- **Raw API Access**: Call arbitrary UniFi controller endpoints
- **Secure Credentials**: Built-in credential protection and secret scrubbing
- **Flexible Configuration**: File, environment, and flag-based configuration

## Quick Start

### Installation

#### From Release (Recommended)

```bash
# Download the latest release for your platform
curl -L https://github.com/johnweldon/unifi-scheduler/releases/latest/download/unifi-scheduler-linux-amd64 -o unifi-scheduler
chmod +x unifi-scheduler
sudo mv unifi-scheduler /usr/local/bin/
```

#### From Source

```bash
git clone https://github.com/johnweldon/unifi-scheduler.git
cd unifi-scheduler
go build -o unifi-scheduler .
```

#### Using Go Install

```bash
go install github.com/johnweldon/unifi-scheduler@latest
```

### Basic Usage

```bash
# First, create secure credentials (recommended)
unifi-scheduler credential create --file ~/.unifi-creds.json

# List all connected clients (using credential file)
unifi-scheduler --endpoint https://your-controller --credential-file ~/.unifi-creds.json client list

# Block a client by name or MAC
unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json client block "Problem Device"

# Monitor network events
unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json event list

# View all available commands
unifi-scheduler --help
```

## Configuration

### Command Line Flags

All configuration can be provided via flags:

```bash
unifi-scheduler \
  --endpoint https://unifi.example.com \
  --username admin \
  --password yourpassword \
  --config ~/.unifi-scheduler.yaml \
  --debug \
  client list
```

### Configuration File

Create `~/.unifi-scheduler.yaml`:

```yaml
# UniFi Controller Settings
endpoint: "https://unifi.example.com"
username: "admin"
password: "yourpassword"

# Network Timeouts
http-timeout: "2m"

# NATS Settings (optional)
nats_url: "nats://localhost:4222"
nats-conn-timeout: "15s"
nats-op-timeout: "30s"
stream-replicas: 3
kv-replicas: 3

# Debug Settings
debug: false
```

### Environment Variables

All settings can be provided via environment variables with `UNIFI_` prefix:

```bash
export UNIFI_ENDPOINT="https://unifi.example.com"
export UNIFI_USERNAME="admin"
export UNIFI_PASSWORD="yourpassword"
export UNIFI_DEBUG="true"

unifi-scheduler client list
```

## Command Reference

### Credential Management

```bash
# Create a secure credential file (interactive)
unifi-scheduler credential create --file ~/.unifi-creds.json

# Store credentials in system keychain (macOS/Linux)
unifi-scheduler credential store-keychain --service unifi-prod --account admin

# Test credential retrieval
unifi-scheduler credential test --credential-file ~/.unifi-creds.json
```

### TLS Configuration

```bash
# Test TLS connection to controller
unifi-scheduler tls test --endpoint https://controller.local

# Show current TLS configuration
unifi-scheduler tls config --endpoint https://controller.local

# Test with custom TLS settings
unifi-scheduler tls test --endpoint https://controller.local --tls-min-version 1.3
```

### Client Management

```bash
# List all connected clients
unifi-scheduler client list

# List all clients (including historical)
unifi-scheduler client list --all

# Block a client (by name, hostname, or MAC)
unifi-scheduler client block "iPhone"
unifi-scheduler client block "aa:bb:cc:dd:ee:ff"

# Unblock a client
unifi-scheduler client unblock "iPhone"

# Kick a client (disconnect)
unifi-scheduler client kick "iPhone"

# Forget a client (remove from controller)
unifi-scheduler client forget "aa:bb:cc:dd:ee:ff"

# Look up client information
unifi-scheduler client lookup "iPhone"
```

### Device Operations

```bash
# List all devices
unifi-scheduler device list

# Display IPv6 delegated prefix from gateway
unifi-scheduler device ipv6-prefix
```

### Event Monitoring

```bash
# List recent events
unifi-scheduler event list

# Monitor connection events
unifi-scheduler event connections
```

### User Management

```bash
# Set user details (name and static IP)
unifi-scheduler user set --mac "aa:bb:cc:dd:ee:ff" --name "My Device" --ip "192.168.1.100"
```

### Raw API Access

```bash
# Call arbitrary UniFi API endpoints
unifi-scheduler raw GET /stat/sta
unifi-scheduler raw POST /cmd/stamgr '{"cmd":"kick-sta","mac":"aa:bb:cc:dd:ee:ff"}'
```

### NATS Operations

```bash
# View cached active clients (requires running agent)
unifi-scheduler --nats_url nats://server:4222 nats clients

# View cached connection events (requires running agent)
unifi-scheduler --nats_url nats://server:4222 nats connections

# Run NATS caching agent (long-running service)
unifi-scheduler --nats_url nats://server:4222 nats agent

# Check IPv6 prefix and publish changes to NATS
unifi-scheduler --nats_url nats://server:4222 nats prefix-check
```

### Multi-Site Controllers

```bash
# Connect to specific site in multi-site controller
unifi-scheduler --endpoint https://controller --site branch-office client list

# Default site is "default"
unifi-scheduler --endpoint https://controller --site default client list
```

### Output Formats

```bash
# Table format (default, human-readable)
unifi-scheduler --output table client list

# JSON format (for automation and parsing)
unifi-scheduler --output json client list

# YAML format (for configuration management)
unifi-scheduler --output yaml device list
```

### Version Information

```bash
# Display version information
unifi-scheduler version
```

## Security Features

### Credential Protection

UniFi Scheduler includes multiple secure credential methods:

- **JSON credential files** with secure file permissions
- **System keychain integration** (macOS/Linux)
- **Stdin credential input** for interactive use
- **Environment variables** for containerized deployments
- **Automatic secret scrubbing** in logs and debug output
- **Memory obfuscation** of sensitive data
- **Credential validation** with size limits

### Credential Method Priority

Credentials are loaded in this order (first available wins):

1. Command-line flags (`--username` and `--password`)
2. Credential file (`--credential-file`)
3. Environment variables (`UNIFI_USERNAME`, `UNIFI_PASSWORD`)
4. System keychain (`--keychain` with `--keychain-account`)
5. Standard input (`--stdin` or interactive prompt)

### Secure Credential Examples

```bash
# Most secure: Use credential file (recommended)
unifi-scheduler credential create --file ~/.unifi-creds.json
unifi-scheduler --credential-file ~/.unifi-creds.json client list

# Secure: Use system keychain
unifi-scheduler credential store-keychain --service unifi --account admin
unifi-scheduler --keychain --keychain-account admin client list

# Secure: Use environment variables
export UNIFI_USERNAME="admin"
export UNIFI_PASSWORD="secret"
unifi-scheduler client list

# Interactive: Read from stdin
unifi-scheduler --stdin client list

# Less secure: Command-line flags (visible in process list)
unifi-scheduler --username admin --password secret client list
```

### Debug Output Safety

Debug output automatically scrubs sensitive information:

```bash
# Passwords and tokens are automatically redacted in debug logs
unifi-scheduler --debug client list
```

## Examples

### Basic Network Management

```bash
# Setup credentials once
unifi-scheduler credential create --file ~/.unifi-creds.json

# Check who's connected
unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json client list

# Block a problematic device
unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json client block "Suspicious-Device"

# Monitor recent network activity
unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json event list
```

### Advanced Operations

```bash
# Create a config file for convenience
cat > ~/.unifi-scheduler.yaml <<EOF
endpoint: "https://controller.local"
credential-file: "~/.unifi-creds.json"
EOF

# Now commands are much shorter
unifi-scheduler client list

# Set up a device with static IP
unifi-scheduler user set \
  --mac "aa:bb:cc:dd:ee:ff" \
  --name "Security Camera" \
  --ip "192.168.1.50"

# Use raw API for custom operations with jq
unifi-scheduler --output json raw GET "/stat/device" | jq '.data[] | select(.type=="udm")'

# Run NATS caching agent for fast data access
unifi-scheduler --nats_url nats://server:4222 nats agent
```

### Batch Operations

```bash
#!/bin/bash
# Block multiple devices (assumes config file is set up)
DEVICES=("Bad-Device-1" "Bad-Device-2" "Suspicious-Phone")

for device in "${DEVICES[@]}"; do
    echo "Blocking $device"
    unifi-scheduler client block "$device"
done

# Or with explicit credentials
ENDPOINT="https://controller"
CREDS="~/.unifi-creds.json"

for device in "${DEVICES[@]}"; do
    echo "Blocking $device"
    unifi-scheduler --endpoint "$ENDPOINT" --credential-file "$CREDS" client block "$device"
done
```

## Troubleshooting

### Common Issues

**Connection Refused**

```bash
# Check endpoint URL and network connectivity
curl -k https://your-unifi-controller/status

# Verify credentials
unifi-scheduler --debug client list
```

**Authentication Failed**

- Verify username and password
- Check if user has admin privileges in UniFi controller
- Try logging into the web interface first

**Certificate Errors**

```bash
# For self-signed certificates, the tool handles them automatically
# If you get certificate errors, verify the endpoint URL
```

**Permission Denied**

- Ensure the UniFi user account has admin privileges
- Check that the account is not restricted

### Debug Mode

Enable debug output for troubleshooting:

```bash
unifi-scheduler --debug client list
```

Debug output includes:

- HTTP request/response details (with credentials scrubbed)
- API endpoint calls
- Authentication flow
- Error details

### NATS Caching and Monitoring

Set up NATS caching for fast distributed access to UniFi data:

```bash
# Start NATS agent to cache UniFi data
unifi-scheduler --nats_url nats://monitoring-server:4222 nats agent

# Query cached data from other instances
unifi-scheduler --nats_url nats://monitoring-server:4222 nats clients
unifi-scheduler --nats_url nats://monitoring-server:4222 nats connections
```

## API Reference

This tool supports all UniFi controller API endpoints. Common endpoints:

- `/stat/sta` - List connected clients
- `/stat/device` - List devices
- `/stat/event` - List events
- `/rest/user` - Manage users
- `/cmd/stamgr` - Client management commands

See [UniFi API Documentation](https://ubntwiki.com/products/software/UniFi-controller/api) for complete endpoint reference.

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Submit a pull request

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make publish

# Run tests
go test ./...

# Clean build artifacts
make clean
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Changelog

### Recent Improvements

- **Security**: Implemented secure credential management with multiple methods (file, keychain, stdin)
- **TLS**: Added comprehensive TLS configuration with custom CA, mutual TLS, and version control
- **Testing**: Added comprehensive test suite with 54.5% coverage in core pkg/unifi
- **Reliability**: Proper error handling throughout with circuit breakers and backoff strategies
- **Performance**: Configurable timeouts and retry logic
- **NATS Integration**: Enhanced distributed operations with caching and event publishing
- **Multi-Site**: Support for multi-site UniFi controllers
- **IPv6**: IPv6 delegated prefix detection and monitoring
- **Output Formats**: Support for table, JSON, and YAML output formats
