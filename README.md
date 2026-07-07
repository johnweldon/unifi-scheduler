# UniFi Scheduler

A CLI tool for managing UniFi network controllers: client management, device monitoring, event tracking, and distributed operations via NATS messaging.

Works with current UniFi OS consoles (UniFi Network 9/10+). Events are sourced from the v2 system-log API; the legacy `/stat/event` endpoint removed in recent UniFi Network releases is no longer used.

## Features

- **Client Management**: Block, unblock, kick, and monitor network clients
- **Device Operations**: List devices, detect IPv6 delegated prefixes
- **Event Monitoring**: Track network events and connection changes
- **NATS Caching**: Long-running agent caches UniFi data in NATS for fast distributed access
- **Raw API Access**: Call arbitrary UniFi controller endpoints
- **Secure Credentials**: Multiple credential sources with secret scrubbing in debug output

## Installation

### Docker (recommended for the NATS agent)

Multi-arch images (linux/amd64, linux/arm64) are published to Docker Hub:

```bash
docker run --rm \
  -e UNIFI_ENDPOINT=https://your-controller \
  -e UNIFI_USERNAME=viewer \
  -e UNIFI_PASSWORD=secret \
  johnweldon/unifi-scheduler:latest client list
```

### Go Install

```bash
go install github.com/johnweldon/unifi-scheduler@latest
```

### From Source

```bash
git clone https://github.com/johnweldon/unifi-scheduler.git
cd unifi-scheduler
go build -o unifi-scheduler .
```

## Quick Start

```bash
# Create a secure credential file (interactive, one time)
unifi-scheduler credential create --file ~/.unifi-creds.json

# List connected clients
unifi-scheduler --endpoint https://your-controller --credential-file ~/.unifi-creds.json client list

# Block a client by name or MAC
unifi-scheduler --endpoint https://your-controller --credential-file ~/.unifi-creds.json client block "Problem Device"

# Monitor recent events
unifi-scheduler --endpoint https://your-controller --credential-file ~/.unifi-creds.json event list
```

A read-only UniFi account (Viewer role) is sufficient for all list/monitor commands and the NATS agent. Mutating commands (`client block|unblock|kick|forget`, `user set`) require an account with admin rights.

## Configuration

Settings are resolved in this order: command-line flags, environment variables, config file.

### Config File

Create `~/.unifi-scheduler.yaml` (or pass `--config`); keys match flag names:

```yaml
endpoint: "https://unifi.example.com"
credential-file: "~/.unifi-creds.json"
http-timeout: "2m"

# NATS settings (only needed for nats subcommands)
nats_url: "nats://localhost:4222"
stream-replicas: 3
kv-replicas: 3
```

### Environment Variables

Any flag can be set via environment variable: prefix with `UNIFI_`, uppercase, and replace hyphens with underscores.

```bash
export UNIFI_ENDPOINT="https://unifi.example.com"
export UNIFI_USERNAME="admin"
export UNIFI_PASSWORD="secret"
export UNIFI_TLS_INSECURE="true"   # --tls-insecure

unifi-scheduler client list
```

### Credential Sources

Credentials are loaded in this order (first available wins):

1. Command-line flags (`--username` and `--password`)
2. Credential file (`--credential-file`)
3. Environment variables (`UNIFI_USERNAME`, `UNIFI_PASSWORD`)
4. System keychain (`--keychain` with `--keychain-account`)
5. Standard input (`--stdin` or interactive prompt)

Prefer the credential file or keychain; flags are visible in the process list. Debug output (`--debug`) automatically scrubs passwords and tokens.

## Command Reference

### Credentials

```bash
unifi-scheduler credential create --file ~/.unifi-creds.json
unifi-scheduler credential store-keychain --service unifi-prod --account admin
unifi-scheduler credential test --credential-file ~/.unifi-creds.json
```

### Clients

```bash
unifi-scheduler client list             # connected clients
unifi-scheduler client list --all       # including offline/historical
unifi-scheduler client block "iPhone"   # by name, hostname, or MAC
unifi-scheduler client unblock "iPhone"
unifi-scheduler client kick "iPhone"
unifi-scheduler client forget "aa:bb:cc:dd:ee:ff"
unifi-scheduler client lookup "iPhone"
```

### Devices

```bash
unifi-scheduler device list
unifi-scheduler device ipv6-prefix      # IPv6 delegated prefix from the gateway
```

### Events

```bash
unifi-scheduler event list              # recent events
unifi-scheduler event list --all        # all available events
unifi-scheduler event connections       # connection/disconnection events
```

### Users

```bash
# Set a friendly name and static IP for a client (positional args)
unifi-scheduler user set "aa:bb:cc:dd:ee:ff" "Security Camera" "192.168.1.50"
```

### Raw API

```bash
unifi-scheduler raw GET /stat/sta
unifi-scheduler raw POST /cmd/stamgr '{"cmd":"kick-sta","mac":"aa:bb:cc:dd:ee:ff"}'

# Combine with jq for custom queries
unifi-scheduler --output json raw GET /stat/device | jq '.data[] | select(.type=="udm")'
```

### NATS

```bash
# Run the caching agent (long-running; polls the controller and caches to NATS)
unifi-scheduler --nats_url nats://server:4222 nats agent

# Query cached data (requires a running agent)
unifi-scheduler --nats_url nats://server:4222 nats clients
unifi-scheduler --nats_url nats://server:4222 nats connections

# Check delegated IPv6 prefix and publish changes to NATS
unifi-scheduler --nats_url nats://server:4222 nats prefix-check
```

NATS credentials are read from `--nats_creds`, `UNIFI_NATS_CREDS`, or `NATS_CREDS`.

### TLS

```bash
unifi-scheduler tls test --endpoint https://controller.local
unifi-scheduler tls config --endpoint https://controller.local
```

For self-signed controller certificates, provide the CA with `--tls-root-ca /path/to/ca.pem`, or use `--tls-insecure` to skip verification (not recommended for production). Mutual TLS is supported via `--tls-client-cert`/`--tls-client-key`, and versions are pinned with `--tls-min-version`/`--tls-max-version`.

### Multi-Site and Output

```bash
unifi-scheduler --site branch-office client list   # default site is "default"
unifi-scheduler --output json client list          # table (default), json, yaml
```

## Troubleshooting

**Connection refused**: check the endpoint URL and network reachability, then run with `--debug` to see the full request/response flow (credentials are scrubbed).

**Authentication failed**: verify credentials by logging into the controller web UI. List/monitor commands work with a read-only Viewer account; mutating commands return 403 without admin rights.

**Certificate errors**: the controller's certificate is verified by default. Use `--tls-root-ca` with your CA bundle, or `--tls-insecure` for testing.

## API Notes

The `raw` command can call UniFi controller endpoints directly. Commonly useful:

- `/stat/sta` — connected clients
- `/stat/device` — devices
- `/stat/user/{mac}` — single client by MAC
- `/rest/user` — all known clients
- `/cmd/stamgr` — client management commands (block/unblock/kick/forget)

Events use `POST /proxy/network/v2/api/site/{site}/system-log/all` (handled internally by `event` commands); the legacy `/stat/event` and `/rest/event` endpoints return 404/400 on current UniFi Network releases. See the [community API documentation](https://ubntwiki.com/products/software/unifi-controller/api) for more endpoints.

## Development

```bash
go test ./...        # run tests
make build           # snapshot build for all platforms (goreleaser, no publish)
make publish         # publish a release (normally done by CI)
```

Releases are published automatically by GitHub Actions when a `v*` tag is pushed: the workflow runs the test suite, then builds and pushes the multi-arch Docker image.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `go test ./...`
5. Submit a pull request

## License

MIT License; see the LICENSE file for details.
