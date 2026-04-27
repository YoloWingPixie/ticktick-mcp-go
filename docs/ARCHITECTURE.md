# Architecture

## Overview

The project produces a single binary, `ticktick-mcp`, with two modes:

- **`ticktick-mcp`** (default) -- the MCP server. Communicates over stdio, launched by Claude Desktop.
- **`ticktick-mcp auth`** -- a one-shot OAuth2 helper. Starts a temporary localhost HTTP server to receive the OAuth callback, exchanges the code for tokens, and stores them in the OS keyring. Run once per profile, then never again unless the token expires without a refresh token.

## Package Dependency Graph

```
cmd/ticktick-mcp ──┬──> internal/server ──┬──> internal/ticktick
                   │                      └──> internal/safety
                   └──> internal/auth
```

All imports flow downward. No package imports a sibling at the same level.

## Package Responsibilities

### `cmd/ticktick-mcp`
Entry point. Dispatches to `auth` subcommand or the MCP server. For the server path: parses flags, loads tokens from the keyring, builds the HTTP client stack (rate limiter, OAuth2 transport), selects the capability mode, and starts the MCP server on stdio. For the auth path: generates a PKCE challenge, opens the browser, runs a temporary callback server, exchanges the code, and persists tokens.

### `internal/server`
The MCP layer. Registers tools based on the active capability mode, handles tool dispatch, manages the LRU cache and singleflight deduplication, and coordinates concurrent project data fetches. Contains all handler logic organized by domain: tasks, projects, filters, and batch operations.

### `internal/ticktick`
HTTP client for the TickTick Open API. Handles request construction, response parsing, operation-aware retry with exponential backoff, and error typing. Defines all domain types (Task, Project, Column, ProjectData).

### `internal/auth`
OAuth2 configuration, PKCE flow, token exchange, and token persistence. Wraps the `99designs/keyring` library for cross-platform credential storage. Implements `PersistingTokenSource` which automatically refreshes expired tokens and writes them back to the keyring.

### `internal/safety`
Input validation (IDs, titles, content, batch sizes, profile names) and HTTP transport hardening (rate limiting, TLS 1.2 minimum, redirect safety with credential stripping, connection timeouts).

## Security Model

**Token lifecycle.** `ticktick-mcp auth` performs the OAuth2 PKCE flow and stores the resulting tokens in the OS keyring (or encrypted file fallback). The server loads tokens at startup and uses a `PersistingTokenSource` that transparently refreshes expired tokens and writes them back.

**Capability modes.** The server registers tools in tiers: read-only (12 tools), read-write (+8 tools), or read-write-destructive (+2 tools). Destructive tools require an explicit `confirmed=true` parameter. The default mode is read-write.

**Input validation.** All tool inputs are validated before reaching the API client. Project and task IDs must match `^[a-f0-9]{24}$`. String fields have length limits. Batch operations are capped at 25 items.

**HTTP hardening.** TLS 1.2 minimum, authorization header stripped on cross-host redirects, redirect loop cap of 10.

## Performance

**Cache + singleflight.** Project lists and project data are cached in LRU caches with a configurable TTL (default 30s). Concurrent requests for the same key are deduplicated via `singleflight.Group`, so N simultaneous tool calls for the same project produce one API request.

**Concurrent fanout.** `get_all_tasks` fetches all project data concurrently with a concurrency limit of 8 (`errgroup` with `SetLimit`).

**Rate limiting.** A token-bucket rate limiter (10 req/s, burst 20) wraps the HTTP transport, applied before all outbound requests including token refreshes.

## Reliability

**Operation-aware retry.** The client categorizes each request as `SafeRead`, `IdempotentWrite`, or `NonIdempotentWrite`. Reads retry on 429, 5xx, and transport errors. Idempotent writes retry on 429 and 5xx. Non-idempotent writes (POST to create, DELETE) never retry. Backoff is exponential with jitter, capped at 10s, and respects `Retry-After` headers.

**Failure modes.** 401 responses immediately surface as `ErrUnauthorized` with a clear message to re-run `ticktick-mcp auth`. Batch operations validate all items before executing any, and return per-item results so callers can see which items succeeded and which failed.

## Key Design Decisions

**Move is implemented as copy + complete.** The TickTick API has no move endpoint. `move_task` creates a copy in the destination project and marks the original as completed, rather than deleting it. This preserves history.

**Batch operations use all-or-nothing validation.** All items in a batch are validated before any API call is made. If validation fails for any item, the entire batch is rejected. Execution itself can partially fail and returns per-item results.

**No partial results as success.** If any project data fetch fails during `get_all_tasks`, the entire operation fails. Returning partial data without indicating which projects are missing would be misleading.

**Cache invalidation is eager.** Write operations immediately invalidate the relevant cache entries rather than waiting for TTL expiry.

**Encrypted file fallback for keyring.** In environments without a system keychain (WSL2, headless Linux, containers), the keyring automatically falls back to AES-encrypted files. The passphrase must be provided via environment variable or file.

## Reading Order

1. `internal/ticktick/types.go` -- domain types
2. `internal/ticktick/client.go` -- HTTP client, retry logic
3. `internal/ticktick/projects.go` -- project API methods
4. `internal/ticktick/tasks.go` -- task API methods
5. `internal/safety/validate.go` -- input validation
6. `internal/safety/ratelimit.go` -- HTTP transport hardening
7. `internal/auth/keyring.go` -- token storage
8. `internal/auth/oauth.go` -- OAuth2 config and PKCE
9. `internal/auth/refresh.go` -- auto-refreshing token source
10. `internal/server/tool_contracts.go` -- tool input/output schemas
11. `internal/server/server.go` -- server setup, caching, tool registration
12. `internal/server/handlers_task.go` -- task handlers
13. `internal/server/handlers_project.go` -- project handlers
14. `internal/server/handlers_filter.go` -- filter/search handlers
15. `internal/server/handlers_batch.go` -- batch operation handlers
16. `cmd/ticktick-mcp/main.go` -- entry point (server + auth subcommand)
