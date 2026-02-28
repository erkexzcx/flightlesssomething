# API & MCP Reference

FlightlessSomething exposes a REST API and an MCP (Model Context Protocol) server. Both interfaces share the same underlying logic and data, but differ in transport and authentication methods.

## Authentication

### Discord OAuth (Web UI)

Browser-based users authenticate via Discord OAuth2. The flow is:

1. `GET /auth/login` — redirects to Discord's authorization page.
2. Discord redirects back to `GET /auth/login/callback` with an authorization code.
3. The server exchanges the code for an access token, fetches the Discord profile, and creates a session cookie.

Session cookies are `HttpOnly`, `SameSite=Lax`, and optionally `Secure` (when the redirect URL uses HTTPS).

### Admin Login

`POST /auth/admin/login` accepts `{"username":"…","password":"…"}` and creates a session for the built-in admin account. This endpoint is rate-limited to **3 failed attempts per 10 minutes** (global lock).

### API Token (Bearer)

Authenticated users can create API tokens (up to 10 per user) via the web UI or API. Tokens are 64-character hex strings passed in the `Authorization` header:

```
Authorization: Bearer <token>
```

API tokens work for all authenticated REST endpoints and for the MCP server.

### MCP Authentication

The MCP server (`POST /mcp`) uses the same Bearer token mechanism. The token is sent in the HTTP `Authorization` header of the JSON-RPC request. Tools are filtered by the caller's access level — unauthenticated callers only see public tools; authenticated callers see public + auth tools; admins see all tools.

## Rate Limits

| Scope | Limit | Window | Applies to |
|---|---|---|---|
| Benchmark uploads | 5 | 10 minutes | Non-admin users |
| Admin login failures | 3 | 10 minutes | Global |

## REST API Endpoints

### Public (no authentication required)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check. Returns `{"status":"ok","version":"…"}`. |
| `GET` | `/auth/login` | Initiates Discord OAuth flow. |
| `GET` | `/auth/login/callback` | Discord OAuth callback. |
| `GET` | `/api/benchmarks` | List/search benchmarks (paginated). |
| `GET` | `/api/benchmarks/:id` | Get benchmark metadata. |
| `GET` | `/api/benchmarks/:id/data` | Stream benchmark statistics as JSON. |
| `GET` | `/api/benchmarks/:id/runs/:runIndex` | Get a single run's data. |
| `GET` | `/api/benchmarks/:id/download` | Download benchmark as a ZIP of CSVs. |
| `GET` | `/api/auth/me` | Returns current user info or `401` if not authenticated. |

### Authenticated (session cookie or Bearer token)

These endpoints use the `RequireAuthOrToken` middleware — either a valid session or a Bearer token is accepted.

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/admin/login` | Admin username/password login. |
| `POST` | `/auth/logout` | End session. |
| `POST` | `/api/benchmarks` | Create a benchmark (multipart form with CSV files). |
| `PUT` | `/api/benchmarks/:id` | Update title, description, or run labels. |
| `DELETE` | `/api/benchmarks/:id` | Delete a benchmark and its data files. |
| `POST` | `/api/benchmarks/:id/runs` | Add runs to an existing benchmark (multipart). |
| `DELETE` | `/api/benchmarks/:id/runs/:run_index` | Delete a specific run from a benchmark. |
| `GET` | `/api/tokens` | List the current user's API tokens. |
| `POST` | `/api/tokens` | Create a new API token. |
| `DELETE` | `/api/tokens/:id` | Delete an API token. |

For write operations on benchmarks (`PUT`, `DELETE`, `POST` runs), the caller must be either the benchmark owner or an admin.

### Admin (session cookie + admin flag)

Admin endpoints require both an authenticated session and the `IsAdmin` flag. They use the `RequireAuth` and `RequireAdmin` middleware chain.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/users` | List users (paginated, searchable). |
| `DELETE` | `/api/admin/users/:id` | Delete a user account. |
| `DELETE` | `/api/admin/users/:id/benchmarks` | Delete all benchmarks for a user. |
| `PUT` | `/api/admin/users/:id/ban` | Ban or unban a user. |
| `PUT` | `/api/admin/users/:id/admin` | Grant or revoke admin privileges. |
| `GET` | `/api/admin/logs` | View audit logs (filtered, paginated). |

### MCP Transport

| Method | Path | Description |
|---|---|---|
| `POST` | `/mcp` | JSON-RPC 2.0 MCP endpoint. |
| `GET` | `/mcp` | Returns `405` — SSE is not supported. |
| `DELETE` | `/mcp` | Session termination (stateless, always succeeds). |

---

## Endpoint Details

### `GET /api/benchmarks`

List and search benchmarks with pagination and sorting.

**Query parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | `1` | Page number. |
| `per_page` | int | `10` | Results per page (1–100). |
| `search` | string | — | Space-separated keywords (AND logic). |
| `search_fields` | string | `title,description` | Comma-separated list of fields to search. Valid values: `title`, `description`, `user`, `run_name`, `specifications`. |
| `user_id` | int | — | Filter by user ID. |
| `sort_by` | string | `created_at` | Sort field: `title`, `created_at`, `updated_at`. |
| `sort_order` | string | `desc` | Sort direction: `asc`, `desc`. |

**Response:** `200 OK`

```json
{
  "benchmarks": [ ... ],
  "page": 1,
  "per_page": 10,
  "total": 42,
  "total_pages": 5
}
```

### `GET /api/benchmarks/:id`

Get a single benchmark's metadata including run count and labels.

**Response:** `200 OK` — Benchmark object with `run_count` and `run_labels`.

### `GET /api/benchmarks/:id/data`

Streams the benchmark's performance data as JSON. The response is written incrementally (one run at a time) to keep memory usage low.

**Response:** `200 OK` — JSON array of run objects, each containing performance metric arrays.

### `GET /api/benchmarks/:id/runs/:runIndex`

Get data for a single run within a benchmark.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | int | Benchmark ID. |
| `runIndex` | int | Zero-based run index. |

### `GET /api/benchmarks/:id/download`

Download all benchmark runs as a ZIP archive. Each run is exported as a separate CSV file inside the ZIP.

**Response:** `200 OK` — `application/zip` attachment.

### `POST /api/benchmarks`

Create a new benchmark. Requires authentication.

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | Yes | Benchmark title (max 100 characters). |
| `description` | string | No | Description in Markdown (max 5,000 characters). |
| `files` | file(s) | Yes | One or more MangoHud CSV or Afterburner HML files. |

**Limits:**

- Max 500,000 data lines per run.
- Max 1,000,000 total data lines across all runs.
- Rate limited to 5 uploads per 10 minutes (non-admins).

**Response:** `201 Created` — The created benchmark object.

### `PUT /api/benchmarks/:id`

Update a benchmark's metadata and/or run labels. Only the owner or an admin can update.

**Request body (JSON):**

```json
{
  "title": "New Title",
  "description": "Updated description",
  "labels": { "0": "Run A", "1": "Run B" }
}
```

All fields are optional — only provided fields are updated.

### `DELETE /api/benchmarks/:id`

Delete a benchmark and all its data files. Only the owner or an admin can delete.

### `POST /api/benchmarks/:id/runs`

Add additional runs to an existing benchmark. Same multipart format as creation.

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Description |
|---|---|---|---|
| `files` | file(s) | Yes | Additional MangoHud CSV or Afterburner HML files. |

The total data lines across existing + new runs must not exceed 1,000,000.

### `DELETE /api/benchmarks/:id/runs/:run_index`

Delete a specific run from a benchmark. Cannot delete the last remaining run. Only the owner or an admin can delete.

### `GET /api/tokens`

List all API tokens for the current user.

### `POST /api/tokens`

Create a new API token.

**Request body (JSON):**

```json
{ "name": "my-token" }
```

`name` is required (1–100 characters). Maximum 10 tokens per user.

**Response:** `201 Created` — The created token object (including the token string, which is only shown once).

### `DELETE /api/tokens/:id`

Delete an API token. Only the token owner can delete it.

### `GET /api/admin/users`

List users with optional search.

**Query parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | `1` | Page number. |
| `per_page` | int | `10` | Results per page (1–100). |
| `search` | string | — | Search by username or Discord ID. |

### `DELETE /api/admin/users/:id`

Delete a user. Cannot delete your own account.

**Query parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `delete_data` | string | `false` | Set to `"true"` to also delete all benchmark data files. |

### `DELETE /api/admin/users/:id/benchmarks`

Delete all benchmarks (and their data files) belonging to a user.

### `PUT /api/admin/users/:id/ban`

Ban or unban a user. Cannot ban your own account.

**Request body (JSON):**

```json
{ "banned": true }
```

### `PUT /api/admin/users/:id/admin`

Grant or revoke admin privileges. Cannot revoke your own admin privileges.

**Request body (JSON):**

```json
{ "is_admin": true }
```

### `GET /api/admin/logs`

List audit logs with optional filters.

**Query parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | `1` | Page number. |
| `per_page` | int | `50` | Results per page (1–100). |
| `user_id` | int | — | Filter by user who performed the action. |
| `action` | string | — | Filter by action (partial match). |
| `target_type` | string | — | Filter by target type (e.g. `user`, `benchmark`). |

---

## MCP Server

The MCP server is a stateless JSON-RPC 2.0 endpoint at `POST /mcp`. It implements the [Model Context Protocol](https://modelcontextprotocol.io/) specification (protocol version `2025-11-25`).

### Protocol Methods

| Method | Description |
|---|---|
| `initialize` | Returns server info, capabilities, and instructions for AI agents. |
| `notifications/initialized` | Client notification. Returns `202 Accepted`. |
| `tools/list` | Lists available tools (filtered by caller's auth level). |
| `tools/call` | Invokes a tool by name with arguments. |
| `ping` | Returns an empty result. |

### Connecting

To connect an AI agent to the MCP server:

```json
{
  "mcpServers": {
    "flightlesssomething": {
      "url": "https://flightlesssomething.example.com/mcp",
      "headers": {
        "Authorization": "Bearer <your-api-token>"
      }
    }
  }
}
```

Without a token, only public (read-only) tools are available. With a token, authenticated tools become available. Admin tokens unlock all tools.

The `initialize` response includes contextual information in its `instructions` field:
- **Server base URL** — the full URL for constructing curl commands (e.g., `https://flightlesssomething.ambrosia.one`).
- **Authenticated user context** — if an API token is provided, the response includes the user's ID, username, and admin status, eliminating the need for a separate "who am I" call.
- **Anonymous mode notice** — if no token is provided, the response indicates that only read-only operations are available.

### MCP Tools

#### Public (no authentication required)

| Tool | Description | Read-only |
|---|---|---|
| `list_benchmarks` | Search and list benchmarks with pagination, search, sorting, and username filtering. | Yes |
| `get_benchmark` | Get detailed benchmark metadata (title, description, user, run count, labels). | Yes |
| `get_benchmark_data` | Get benchmark metadata and computed statistics for all runs in a single call (min, max, avg, median, P1, P5, P10, P25, P75, P90, P95, P97, P99, IQR, std dev, variance). Optionally include downsampled raw data (up to 5,000 points). | Yes |
| `get_benchmark_run` | Get computed statistics for a single run. | Yes |

#### Authenticated (Bearer token required)

| Tool | Description | Read-only |
|---|---|---|
| `update_benchmark` | Update title, description, and/or run labels. Owner or admin only. | No |
| `delete_benchmark` | Delete a benchmark and all its data. Owner or admin only. | No |
| `delete_benchmark_run` | Delete a specific run. Cannot delete the last remaining run. Owner or admin only. | No |
| `list_api_tokens` | List the current user's API tokens. | Yes |
| `create_api_token` | Create a new API token (max 10 per user). | No |
| `delete_api_token` | Delete an API token belonging to the current user. | No |

#### Admin (Bearer token with admin privileges)

| Tool | Description | Read-only |
|---|---|---|
| `list_users` | List all users with pagination and search. | Yes |
| `list_audit_logs` | List audit logs with filters (user, action, target type). | Yes |
| `delete_user` | Delete a user account. Cannot delete your own account. | No |
| `delete_user_benchmarks` | Delete all benchmarks belonging to a user. | No |
| `ban_user` | Ban or unban a user. Cannot ban your own account. | No |
| `toggle_user_admin` | Grant or revoke admin privileges. Cannot revoke your own. | No |

### API–MCP Parity

Every REST API endpoint has a corresponding MCP tool **except**:

- **Benchmark file upload** (`POST /api/benchmarks`, `POST /api/benchmarks/:id/runs`) — requires multipart form data, unsuitable for MCP.
- **Benchmark ZIP download** (`GET /api/benchmarks/:id/download`) — large binary transfer, unsuitable for MCP.
- **Current user info** (`GET /api/auth/me`) — user context is provided in the `initialize` response instead, eliminating the need for a separate tool call.

For these operations, the MCP server's initialization instructions guide AI agents to use `curl` with the REST API directly.

### Tool Parameters

Each tool accepts a JSON object as `arguments` in the `tools/call` request. Below are the parameters for each tool.

#### `list_benchmarks`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `page` | int | No | Page number (default: 1). |
| `per_page` | int | No | Results per page, 1–100 (default: 10). |
| `search` | string | No | Search keywords (space-separated, AND logic). |
| `user_id` | int | No | Filter by user ID. |
| `username` | string | No | Filter by exact username (case-insensitive). Use instead of `user_id` when you know the username. |
| `sort_by` | string | No | `title`, `created_at`, or `updated_at` (default: `created_at`). |
| `sort_order` | string | No | `asc` or `desc` (default: `desc`). |

#### `get_benchmark`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Benchmark ID. |

#### `get_benchmark_data`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Benchmark ID. |
| `max_points` | int | No | Include downsampled raw data (0 = stats only, 1–5,000 for time series). |

#### `get_benchmark_run`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Benchmark ID. |
| `run_index` | int | Yes | Zero-based run index. |
| `max_points` | int | No | Include downsampled raw data (0 = stats only, 1–5,000 for time series). |

#### `update_benchmark`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Benchmark ID. |
| `title` | string | No | New title (max 100 characters). |
| `description` | string | No | New description in Markdown (max 5,000 characters). |
| `labels` | object | No | Map of run index (string key) to new label, e.g. `{"0": "Run A"}`. |

#### `delete_benchmark`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Benchmark ID. |

#### `delete_benchmark_run`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Benchmark ID. |
| `run_index` | int | Yes | Zero-based run index. |

#### `list_api_tokens`

No parameters.

#### `create_api_token`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Token name (1–100 characters). |

#### `delete_api_token`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `token_id` | int | Yes | Token ID to delete. |

#### `list_users`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `page` | int | No | Page number (default: 1). |
| `per_page` | int | No | Results per page, 1–100 (default: 10). |
| `search` | string | No | Search by username or Discord ID. |

#### `list_audit_logs`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `page` | int | No | Page number (default: 1). |
| `per_page` | int | No | Results per page, 1–100 (default: 50). |
| `user_id` | int | No | Filter by user who performed the action. |
| `action` | string | No | Filter by action (partial match). |
| `target_type` | string | No | Filter by target type (e.g. `user`, `benchmark`). |

#### `delete_user`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `user_id` | int | Yes | User ID to delete. |
| `delete_data` | bool | No | Also delete all benchmark data files (default: false). |

#### `delete_user_benchmarks`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `user_id` | int | Yes | User ID whose benchmarks to delete. |

#### `ban_user`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `user_id` | int | Yes | User ID to ban/unban. |
| `banned` | bool | Yes | `true` to ban, `false` to unban. |

#### `toggle_user_admin`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `user_id` | int | Yes | User ID to modify. |
| `is_admin` | bool | Yes | `true` to grant admin, `false` to revoke. |
