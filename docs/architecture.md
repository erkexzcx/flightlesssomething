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
                        │      └── ...                 │
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
4. **Audit log** — record the action (non-blocking; failures are logged but don't fail the request)
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

The application exposes a [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) endpoint at `/mcp` using JSON-RPC 2.0. This allows AI assistants to interact with the application programmatically. Tools mirror the REST API, except for binary file transfer operations (upload/download) which are unsuitable for MCP.

## Frontend

### Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Framework | Vue 3 (Composition API) | Reactive UI with `<script setup>` syntax |
| State | [Pinia](https://pinia.vuejs.org/) | Lightweight state management |
| Routing | Vue Router | Client-side navigation with lazy loading |
| Build | [Vite](https://vitejs.dev/) | Fast development builds, optimized production output |
| Charts | [Highcharts](https://www.highcharts.com/) | Interactive benchmark visualization |
| Styling | Bootstrap 5 + Font Awesome | No custom CSS framework |
| Security | [DOMPurify](https://github.com/cure53/DOMPurify) + [Marked](https://marked.js.org/) | Safe Markdown rendering in descriptions |

### Chunk Splitting

Vite is configured with manual chunk splitting to optimize loading:

- **`vendor`** chunk — Vue, Vue Router, Pinia (framework code, cached long-term)
- **`charts`** chunk — Highcharts (large library, loaded only when viewing benchmarks)

### Web Workers

CPU-intensive operations run in Web Workers to keep the UI responsive:

- **`jsonParser.worker.js`** — parses large JSON responses in a background thread, preventing the main thread from freezing on benchmarks with hundreds of thousands of data points
- **`statsCalculator.worker.js`** — calculates statistics (percentiles, min/max, averages, standard deviation) in parallel, supporting both linear-interpolation and MangoHud-threshold calculation methods

### Incremental Data Loading

Benchmark data is fetched one run at a time via `/api/benchmarks/:id/runs/:runIndex`. Each run is processed immediately — downsampled and statistics calculated — then raw data is discarded. This prevents Vue's reactivity system from tracking massive arrays, keeping memory usage proportional to the downsampled display data rather than the full dataset.

### API Client

All API calls go through a centralized client (`web/src/api/client.js`) that provides:

- Structured error handling via an `APIError` class
- Progress callbacks for download and parse operations
- Dynamic imports for benchmark data loaders (reducing initial bundle size)

### SPA Routing

The embedded SPA uses fallback routing:

- `/assets/*` and `/favicon.*` — static files from the embedded filesystem
- `/api/*`, `/auth/*`, `/health`, `/mcp` — API routes
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

This enables **streaming reads** — the server decodes and writes one run at a time, without loading the entire benchmark into memory. During streaming:

- **JSON export** (`StreamBenchmarkDataAsJSON`) triggers `runtime.GC()` every 10 runs
- **ZIP export** (`ExportBenchmarkDataAsZip`) triggers `runtime.GC()` every 5 runs

The companion `.meta` file stores run count and labels as gob, enabling quick metadata access without decompressing the data file.

### Compression

zstd compression is configured with:

- **Encoder**: `SpeedDefault` level, 2 concurrent threads, 256 KB write buffer
- **Decoder**: 2 concurrent threads

This balances compression ratio, speed, and memory usage. Limiting concurrency to 2 threads avoids overwhelming low-CPU servers.

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

SQLite with GORM auto-migration. The database file (`flightlesssomething.db`) stores user accounts, benchmark metadata, API tokens, and audit logs. Schema version is tracked in a `schema_versions` table (current version: 3).

### Benchmark Files

Benchmark data is stored on the filesystem, not in the database:

```
{dataDir}/benchmarks/
  ├── {id}.bin     zstd-compressed gob (V2 streaming format)
  └── {id}.meta    gob-encoded metadata (run count + labels)
```

Each `.bin` file contains a header followed by individually encoded runs. Each `.meta` file provides quick access to run count and labels without decompressing the data.

### Schema Migrations

Migrations run automatically on startup:

- **v0 → v1**: Removed `ai_summary` column, created `schema_versions` table
- **v1 → v2**: Added `RunNames` and `Specifications` searchable fields to benchmarks
- **v2 → v3**: Migrated storage format from V1 (single array) to V2 (per-run streaming) and regenerated metadata files

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

1. **`make build-web`** — `npm ci && npm run build` produces `web/dist/`
2. **`make build-server`** — Go compiles with `web/dist/` embedded via `//go:embed`, producing a single binary with version info from git tags

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
