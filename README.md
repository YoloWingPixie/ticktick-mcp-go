# ticktick-mcp-go

A Go MCP server for TickTick task management.

## Features

- **Single static binary** -- zero runtime dependencies, `CGO_ENABLED=0`
- **OS keyring token storage** -- macOS Keychain, Windows Credential Manager, Secret Service, or encrypted file fallback
- **Capability modes** -- read-only, read-write, or read-write-destructive
- **22 tools** -- projects, tasks, filtering, batch operations, search
- **Local filtering** -- date/priority/search queries resolved in-process, no extra API calls
- **Caching** -- LRU + singleflight deduplication with configurable TTL
- **Rate limiting** -- 10 req/s with burst of 20, applied before all outbound calls
- **Multi-profile** -- separate credentials per TickTick account

## Install

```bash
go install github.com/YoloWingPixie/ticktick-mcp-go/cmd/ticktick-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/YoloWingPixie/ticktick-mcp-go.git
cd ticktick-mcp-go
task build
```

## Quick Start

```bash
# 1. Register an app at the TickTick developer portal
#    https://developer.ticktick.com/manage
#    Set the redirect URL to http://127.0.0.1:8000/callback

# 2. Export your credentials
export TICKTICK_CLIENT_ID="your-client-id"
export TICKTICK_CLIENT_SECRET="your-client-secret"

# 3. Authenticate (opens browser)
ticktick-mcp auth

# 4. Verify
ticktick-mcp --healthcheck
```

## Claude Desktop Configuration

### Single account

```json
{
  "mcpServers": {
    "ticktick": {
      "command": "ticktick-mcp",
      "env": {
        "TICKTICK_CLIENT_ID": "your-client-id",
        "TICKTICK_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

### Multiple accounts

```json
{
  "mcpServers": {
    "ticktick-work": {
      "command": "ticktick-mcp",
      "args": ["--profile", "work"],
      "env": {
        "TICKTICK_CLIENT_ID": "your-client-id",
        "TICKTICK_CLIENT_SECRET": "your-client-secret"
      }
    },
    "ticktick-personal": {
      "command": "ticktick-mcp",
      "args": ["--profile", "personal", "--read-only"],
      "env": {
        "TICKTICK_CLIENT_ID": "your-client-id",
        "TICKTICK_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

### WSL2

Add a keyring passphrase since there is no D-Bus/keychain:

```json
{
  "mcpServers": {
    "ticktick": {
      "command": "ticktick-mcp",
      "env": {
        "TICKTICK_CLIENT_ID": "your-client-id",
        "TICKTICK_CLIENT_SECRET": "your-client-secret",
        "TICKTICK_KEYRING_PASSPHRASE": "your-passphrase"
      }
    }
  }
}
```

## CLI Reference

### ticktick-mcp

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `default` | Credential profile name |
| `--read-only` | `false` | Register only read tools |
| `--allow-destructive` | `false` | Register destructive tools (delete) |
| `--cache-ttl` | `30s` | Cache TTL (`0` to disable) |
| `--no-cache` | `false` | Disable caching entirely |
| `--debug` | `false` | Enable debug logging |
| `--version` | | Print version and exit |
| `--healthcheck` | | Test credentials and exit |
| `--whoami` | | Print profile name and exit |
| `--list-profiles` | | List stored profiles and exit |

### ticktick-mcp auth

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `default` | Credential profile name |
| `--addr` | `127.0.0.1:8000` | OAuth callback server address |
| `--version` | | Print version and exit |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TICKTICK_CLIENT_ID` | OAuth2 client ID (required) |
| `TICKTICK_CLIENT_SECRET` | OAuth2 client secret (required) |
| `TICKTICK_PROFILE` | Default profile name (overridden by `--profile`) |
| `TICKTICK_READ_ONLY` | Set to `1` or `true` for read-only mode |
| `TICKTICK_ALLOW_DESTRUCTIVE` | Set to `1` or `true` to enable destructive tools |
| `TICKTICK_NO_CACHE` | Set to `1` or `true` to disable caching |
| `TICKTICK_CACHE_TTL` | Cache TTL duration (e.g. `30s`, `1m`) |
| `TICKTICK_KEYRING_PASSPHRASE` | Passphrase for encrypted file keyring |
| `TICKTICK_KEYRING_PASSPHRASE_FILE` | Path to file containing keyring passphrase |

## Capability Modes

The server registers tools based on the active capability mode:

| Mode | Flag | Tools |
|------|------|-------|
| **Read-only** | `--read-only` | 12 read tools |
| **Read-write** (default) | _(none)_ | 12 read + 8 write tools |
| **Read-write-destructive** | `--allow-destructive` | 12 read + 8 write + 2 destructive tools |

## Tool Reference

### Read Tools (always registered)

| Tool | Description |
|------|-------------|
| `get_projects` | List all projects |
| `get_project` | Get a single project by ID |
| `get_project_with_data` | Get a project with all tasks and columns |
| `get_task` | Get a single task by project ID and task ID |
| `get_all_tasks` | Get all tasks across all projects |
| `get_tasks_due_today` | Incomplete tasks due today |
| `get_tasks_due_this_week` | Incomplete tasks due this week |
| `get_overdue_tasks` | Incomplete tasks past their due date |
| `get_tasks_by_priority` | Tasks with a specific priority level |
| `search_tasks` | Search tasks by title/content (case-insensitive) |
| `get_engaged_tasks` | Started but not yet completed tasks |
| `get_next_tasks` | Incomplete tasks due in the next 7 days |

### Write Tools (read-write mode and above)

| Tool | Description |
|------|-------------|
| `create_task` | Create a task (inbox if no project specified) |
| `update_task` | Update an existing task (partial update) |
| `complete_task` | Mark a task as completed |
| `move_task` | Move a task between projects |
| `create_project` | Create a new project |
| `update_project` | Update an existing project (partial update) |
| `batch_create_tasks` | Create multiple tasks at once |
| `batch_complete_tasks` | Complete multiple tasks at once |

### Destructive Tools (destructive mode only)

| Tool | Description |
|------|-------------|
| `delete_task` | Permanently delete a task (requires `confirmed=true`) |
| `delete_project` | Permanently delete a project and all its tasks (requires `confirmed=true`) |

## Token Storage

Tokens are stored using the [99designs/keyring](https://github.com/99designs/keyring) library, which tries backends in order:

1. **macOS** -- Keychain
2. **Linux** -- Secret Service (GNOME Keyring / KWallet) via D-Bus
3. **Windows** -- Windows Credential Manager
4. **Fallback** -- AES-encrypted file at `~/.local/share/ticktick-mcp/`

The encrypted file fallback requires a passphrase via `TICKTICK_KEYRING_PASSPHRASE` or `TICKTICK_KEYRING_PASSPHRASE_FILE`.

### WSL2

WSL2 has no D-Bus or keychain by default. The keyring falls back to encrypted file storage. You must provide a passphrase:

```bash
# Option 1: environment variable
export TICKTICK_KEYRING_PASSPHRASE="your-passphrase"

# Option 2: passphrase file (must be chmod 600)
echo "your-passphrase" > ~/.ticktick-passphrase
chmod 600 ~/.ticktick-passphrase
export TICKTICK_KEYRING_PASSPHRASE_FILE="$HOME/.ticktick-passphrase"
```

## Troubleshooting

| Error | Fix |
|-------|-----|
| `no credentials found` | Run `ticktick-mcp auth` to authenticate |
| `encrypted file keyring requires a passphrase` | Set `TICKTICK_KEYRING_PASSPHRASE` or `TICKTICK_KEYRING_PASSPHRASE_FILE` |
| `passphrase file is accessible by other users` | Run `chmod 600` on the passphrase file |
| `unauthorized` / `token may be expired` | Re-run `ticktick-mcp auth` to get a fresh token |
| WSL2: no keychain available | Use the passphrase environment variable or passphrase file (see above) |

## Building

```bash
task build          # Build binary to bin/
task test           # Run tests with race detector
task lint           # Run golangci-lint
task vuln           # Run govulncheck
task clean          # Remove bin/
```

## License

See [LICENSE](LICENSE).
