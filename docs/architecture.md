# Architecture

FlightlessSomething is designed to run on low-resource servers — constrained RAM and CPU — while handling large benchmark datasets (up to 1 million data points per benchmark). Every architectural decision prioritizes memory efficiency over raw throughput.

## System Overview

The application compiles into a **single Go binary** with the Vue.js frontend embedded via Go's `//go:embed` directive. At runtime, the binary serves both the REST API and the single-page application from the same HTTP port (default `:5000`). There are no external runtime dependencies beyond a filesystem for SQLite and benchmark data files.

```
┌─────────────────────────────────────────────────┐
│                  Single Binary                  │
│                                                 │
│  ┌──────────────┐    ┌───────────────────────┐  │
│  │  Embedded    │    │     Go HTTP Server     │  │
│  │  Vue.js SPA  │    │     (Gin Framework)    │  │
│  │  (static)    │    │                        │  │
│  └──────┬───────┘    │  ┌──────────────────┐  │  │
│         │            │  │   REST API        │  │  │
│         │ serves     │  │   MCP Protocol    │  │  │
│         │ /assets/*  │  │   Auth Middleware  │  │  │
│         │ index.html │  │   Rate Limiter    │  │  │
│         ▼            │  └────────┬─────────┘  │  │
│     Browser          │           │             │  │
│                      │  ┌────────▼─────────┐  │  │
│                      │  │   GORM + SQLite   │  │  │
│                      │  └────────┬─────────┘  │  │
│                      └───────────┼────────────┘  │
└──────────────────────────────────┼───────────────┘
                                   │
                        ┌──────────▼─────────────────┐
                        │       Filesystem             │
                        │  ├── flightlesssomething.db  │
                        │  └── benchmarks/             │
                        │      ├── {id}.bin            │
                        │      ├── {id}.meta           │
                        │      └── {id}.stats          │
                        └──────────────────────────────┘
```

## Backend

### Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Low memory footprint, single binary output, built-in concurrency |
| HTTP framework | [Gin](https://github.com/gin-gonic/gin) | Lightweight, fast routing |
| ORM | [GORM](https://gorm.io/) with SQLite | Zero-configuration embedded database, no external process |
| Compression | [zstd](https://github.com/klauspost/compress) | High compression ratio with fast decompression |
| Serialization | Go's `encoding/gob` | Compact binary format, native to Go |
| Configuration | CLI flags + env vars (`FS_` prefix) via [ff](https://github.com/peterbourgon/ff) | Flexible deployment |

### Request Lifecycle

Every API handler follows a consistent pattern:

1. **Validate input** — check required fields, length limits, numeric bounds
2. **Check authentication/authorization** — session or Bearer token, admin flag
3. **Execute operation** — database query, file I/O, or both
4. **Audit log** — record the action to file-based JSON log (non-blocking; failures are logged to stdout but don't fail the request)
5. **Return JSON response**

### Pagination

All list endpoints use offset-based pagination:

- `page` (default 1) + `per_page` (default varies by endpoint, max 100)
- Responses include `total`, `page`, `per_page`, and `total_pages`

### Search

Multi-field `LIKE` queries with a minimum 3-character search term. Benchmark search covers title, description, run names, and hardware specifications. User search covers username and Discord ID.

### Rate Limiting

An in-memory sliding window rate limiter protects write operations:

- **Benchmark uploads**: 5 per 10 minutes per user (non-admins)
- **Admin login**: 3 failed attempts per 10 minutes (global lock)

A background goroutine cleans up expired entries every 5 minutes. The implementation uses `sync.RWMutex` for thread safety.

### MCP Protocol

The application exposes a [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) endpoint at `/mcp` using JSON-RPC 2.0. This allows AI assistants to interact with the application programmatically. Tools provide read-only benchmark access, metadata editing, and admin operations. Benchmark data upload, download, deletion, and API token management are intentionally excluded as they are unsuitable for MCP. All tools support an optional `jq` parameter for server-side result filtering and transformation.

The `/mcp` endpoint uses `MCPCors()` middleware with wildcard origin (`*`) to allow browser-based MCP clients such as MCP Inspector to connect directly.

## Frontend

### Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Framework | Vue 3 (Composition API) | Reactive UI with `<script setup>` syntax |
| State | [Pinia](https://pinia.vuejs.org/) | Lightweight state management |
| Routing | Vue Router | Client-side navigation with eagerly imported views |
| Build | [Vite](https://vitejs.dev/) | Fast development builds, optimized production output |
| Charts | [Highcharts](https://www.highcharts.com/) | Interactive benchmark visualization |
| Styling | Bootstrap 5 + Font Awesome | No custom CSS framework |
| Security | [DOMPurify](https://github.com/cure53/DOMPurify) + [Marked](https://marked.js.org/) | Safe Markdown rendering in descriptions |
| Date formatting | [Day.js](https://day.js.org/) | Lightweight date/time utilities for relative timestamps |

### Chunk Splitting

Vite is configured with manual chunk splitting to optimize loading:

- **`vendor`** chunk — Vue, Vue Router, Pinia (framework code, cached long-term)
- **`charts`** chunk — Highcharts (large library, loaded only when viewing benchmarks)

### Benchmark Data Loading

The backend pre-calculates all statistics and LTTB-downsampled series during upload and stores them in `.stats` files. The frontend performs no computation — it simply maps the backend response into the chart-ready format.

**Loading pipeline (`web/src/utils/benchmarkRunLoader.js`):**

- Fetches runs incrementally via `/api/benchmarks/:id/runs/:runIndex`
- Uses a worker pool pattern with parallel HTTP requests (not Web Workers) — concurrency is `Math.min(cores, totalRuns, 6)` where `cores` is `navigator.hardwareConcurrency` (defaulting to 4 if unavailable)
- Each response is a `PreCalculatedRun` object containing stats, LTTB-downsampled series, and density histogram data

**Format mapping (`web/src/utils/benchmarkDataProcessor.js`):**

- Lightweight mapping only — transforms the `PreCalculatedRun` response structure into the format expected by Highcharts
- No statistical computation on the frontend

**Client-side verification (`web/src/utils/statsCalculations.js`):**

- Contains percentile calculation functions (linear interpolation and MangoHud threshold methods)
- Used exclusively by the DebugCalc page for comparing client-side vs. backend results
- Not used during normal benchmark data loading

### API Client

All API calls go through a centralized client (`web/src/api/client.js`) that provides:

- Structured error handling via an `APIError` class
- Progress callbacks for download and parse operations
- Dynamic imports for benchmark data loaders (reducing initial bundle size)

### SPA Routing

The embedded SPA uses fallback routing:

- `/assets/*` and `/favicon.*` — static files from the embedded filesystem
- `/api/*`, `/auth/*`, `/health` — API routes (excluded from the `NoRoute` SPA fallback handler in `web.go`)
- `/mcp` — MCP routes (handled by explicitly registered Gin routes, not the `NoRoute` handler)
- Everything else — serves `index.html` for Vue Router to handle client-side

## Performance Design

This application is built to run on servers with as little as 512 MiB of RAM. The following techniques keep memory usage low while processing large datasets.

### Garbage Collection Tuning

The application sets `GOGC=50` at startup (default Go value is 100), which triggers garbage collection twice as frequently. This trades a small amount of CPU time for significantly lower peak memory usage. Operators can further tune with:

- `GOGC` — garbage collection target percentage
- `GOMEMLIMIT` — soft memory limit (e.g. `512MiB`), leveraging Go's memory-limit-aware GC

### Two-Pass CSV Parsing

When a user uploads a benchmark CSV file, it is parsed in two passes:

1. **First pass** — stream through the file counting lines, storing nothing in memory
2. **Second pass** — reopen the file and parse with exact-capacity pre-allocated slices

This eliminates slice growth and reallocation. Each of the 13 metric arrays (`DataFPS`, `DataFrameTime`, `DataCPULoad`, etc.) is pre-allocated to the exact line count.

### Streaming Data Storage (V2 Format)

Benchmark data is stored as zstd-compressed gob with a per-run encoding scheme:

```
.bin file:
  [zstd compression]
    ├── fileHeader { Version: 2, RunCount: N }
    ├── BenchmarkData (run 1)  ← individually gob-encoded
    ├── BenchmarkData (run 2)
    └── ...
```

This enables **streaming reads** — the server decodes one run at a time, without loading the entire benchmark into memory.

`HandleGetBenchmarkData` serves pre-calculated statistics from the `.stats` file rather than streaming `.bin` data. If the `.stats` file is unavailable the endpoint returns a 500 error — stats are always generated during upload and re-generated by the v3→v4 migration, so this case should not occur in normal operation.

During ZIP export:

- **ZIP export** (`ExportBenchmarkDataAsZip`) triggers `runtime.GC()` every 5 runs

The companion `.meta` file stores run count and labels as gob, enabling quick metadata access without decompressing the data file.

### Compression

zstd compression is configured with:

- **Encoder**: `SpeedDefault` level, 2 concurrent threads, 256 KB write buffer
- **Decoder**: 2 concurrent threads

This balances compression ratio, speed, and memory usage. Limiting concurrency to 2 threads avoids overwhelming low-CPU servers. Both `.bin` and `.stats` files use these same encoder settings.

### Data Limits

Hard limits prevent any single upload from consuming excessive resources:

| Limit | Value |
|-------|-------|
| Total data lines per benchmark | 1,000,000 |
| Data lines per single run | 500,000 |
| Benchmark title | 100 characters |
| Benchmark description | 5,000 characters |
| API tokens per user | 10 |

## Data Storage

### Database

SQLite with GORM auto-migration. The database file (`flightlesssomething.db`) stores user accounts, benchmark metadata, and API tokens. Schema version is tracked in a `schema_versions` table (current version: 5). Audit logs are written to a JSON log file in a `logs/` directory alongside the data directory (sibling, not inside), with automatic rotation (gzip-compressed) at 10 MB and retention of the 10 most recent rotated files.

### Benchmark Files

Benchmark data is stored on the filesystem, not in the database:

```
{dataDir}/benchmarks/
  ├── {id}.bin     zstd-compressed gob (V2 streaming format)
  ├── {id}.meta    gob-encoded metadata (run count + labels)
  └── {id}.stats   zstd-compressed gob (pre-calculated statistics + downsampled series)
```

Each `.bin` file contains a header followed by individually encoded runs. Each `.meta` file provides quick access to run count and labels without decompressing the data. Each `.stats` file contains a `[]*PreCalculatedRun` slice with per-metric statistics (for both linear interpolation and MangoHud threshold methods), LTTB-downsampled series (max 2000 points), and density histogram data — written during upload so the API can serve benchmark data with zero computation at read time.

### Schema Migrations

Migrations run automatically on startup:

- **v0 → v1**: Removed `ai_summary` column, created `schema_versions` table
- **v1 → v2**: Added `RunNames` and `Specifications` searchable fields to benchmarks
- **v2 → v3**: Migrated storage format from V1 (single array) to V2 (per-run streaming) and regenerated metadata files
- **v3 → v4**: Pre-calculated statistics for all benchmarks (`.stats` files) for instant loading
- **v4 → v5**: Dropped `audit_logs` table (audit logs moved to file-based JSON logging)

Legacy V1 data files are detected by reading the file header. If the header decode fails, the server falls back to legacy loading (full dataset in memory).

## Authentication

Three authentication methods, used in different contexts:

### Discord OAuth 2.0

Primary login for end users:

1. `GET /auth/login` generates a random state token and redirects to Discord
2. Discord callback exchanges the code for a token and fetches user info
3. The server creates or updates the User record and sets a session cookie
4. Ban status is checked before completing login

### Admin Login

System administrator login with username and password:

- `POST /auth/admin/login` with credentials
- Rate limited globally (3 failed attempts per 10 minutes)
- Creates a special user with `DiscordID = "admin"`

### API Token Authentication

For programmatic access (API and MCP):

- `Authorization: Bearer <64-char-hex-token>` header
- The `RequireAuthOrToken` middleware checks session cookies first, then falls back to Bearer tokens
- Tracks `LastUsedAt` on the token and `LastAPIActivityAt` on the user

### Session Security

- HttpOnly cookies (not accessible via JavaScript)
- SameSite=Lax (CSRF protection)
- Secure flag auto-detected from the OAuth redirect URL scheme (HTTPS = secure)
- Cookie store encrypted with the configured session secret

## Build and Deployment

### Build Pipeline

The `Makefile` orchestrates a two-stage build:

1. **`make build-web`** — `npm install && npm run build` produces `web/dist/` (the Dockerfile uses `npm ci --prefer-offline --no-audit` instead)
2. **`make build-server`** — depends on `make build-web`; copies `web/dist/` into `internal/app/web/`, generates `webfs_embed.go` with the `//go:embed` directive, then compiles the Go binary with version info from git tags and removes the temporary files

### Docker

A multi-stage Dockerfile minimizes the final image:

1. **Node stage** (node:25-alpine) — builds the Vue.js frontend
2. **Go stage** (go:1.26-alpine) — compiles the server with embedded assets
3. **Runtime stage** (alpine:3.23) — copies only the binary and CA certificates

The final image contains a single binary, CA certificates for HTTPS (Discord OAuth), and nothing else.

### Development

For local development, the backend and frontend run separately:

- **Backend**: `make run` or `go run ./cmd/server` on port 5000
- **Frontend**: `cd web && npm run dev` starts Vite dev server on port 3000, proxying `/api`, `/auth`, and `/health` requests to port 5000

> **Note**: The Vite dev proxy (`npm run dev`, port 3000) does not proxy `/mcp`. When developing locally against the Vite dev server, connect your MCP client directly to `http://localhost:5000/mcp`. In production, the reverse proxy forwards all paths (including `/mcp`) to port 5000 normally.

### Docker Compose

The `docker-compose.yml` mounts two volumes:

- `./data:/data` — SQLite database and benchmark files
- `./logs:/logs` — Audit log directory (sibling of `/data`, as set by `InitAuditLog`)

The `logs` volume must be mounted separately because audit logs are written to the **parent** of the data directory. Without this mount, audit logs would be lost when the container is removed.
