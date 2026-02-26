# Copilot Instructions for FlightlessSomething

## Project Overview

FlightlessSomething is a full-stack web application for storing, managing, and visualizing gaming benchmark data. Users authenticate via Discord OAuth, upload MangoHud or Afterburner CSV benchmark files, and view interactive performance charts (FPS, frame time, CPU/GPU metrics). The app supports multi-user management, admin controls, audit logging, API token authentication, and an MCP (Model Context Protocol) server for AI assistant integration.

**Live instances:**
- Production: `https://flightlesssomething.ambrosia.one/`
- Development: `https://flightlesssomething-dev.ambrosia.one/`

---

## Architecture

### Single Binary Deployment

The application compiles into a **single Go binary** with the Vue.js frontend embedded via Go's `//go:embed` directive. The `Makefile` builds the frontend first (`npm run build` → `web/dist/`), then compiles the Go server with the static assets baked in. At runtime, the binary serves both the API and the SPA from the same port (default `:5000`).

### Backend (Go)

- **HTTP framework:** [Gin](https://github.com/gin-gonic/gin) with session middleware
- **ORM:** [GORM](https://gorm.io/) with SQLite driver
- **Authentication:** Discord OAuth2 (via `golang.org/x/oauth2`) + session cookies + API Bearer tokens
- **Compression:** zstd (via `github.com/klauspost/compress`) for benchmark data storage
- **Encoding:** gob for binary serialization of benchmark data
- **Configuration:** CLI flags + environment variables (prefix `FS_`) via `peterbourgon/ff`
- **Entry point:** `cmd/server/main.go`
- **All application logic:** `internal/app/` package

### Frontend (Vue.js)

- **Framework:** Vue 3 with Composition API
- **State management:** Pinia (stores in `web/src/stores/`)
- **Routing:** Vue Router with lazy loading
- **Build tool:** Vite (dev server on port 3000, proxies API to port 5000)
- **Styling:** Bootstrap 5 + Font Awesome
- **Charts:** Highcharts for benchmark visualization
- **Security:** DOMPurify for HTML sanitization, Marked for Markdown rendering
- **Web Workers:** Background threads for JSON parsing and stats calculation to avoid UI freezing

### Data Storage

- **Database:** SQLite file at `{dataDir}/flightlesssomething.db`
- **Benchmark data files:** `{dataDir}/benchmarks/{benchmarkID}/data.bin` (zstd-compressed gob, V2 format with per-run streaming)
- **Metadata files:** `{dataDir}/benchmarks/{benchmarkID}/data.meta` (JSON with run count and labels)

---

## Directory Structure

```
flightlesssomething/
├── cmd/server/main.go              # Entry point: GC tuning, config load, server start
├── internal/app/                   # All backend application logic
│   ├── admin.go                    # Admin user management handlers (list/delete/ban/admin-toggle)
│   ├── api_tokens.go               # API token CRUD + RequireAuthOrToken middleware
│   ├── audit.go                    # Audit log creation and listing with filters
│   ├── auth.go                     # Discord OAuth flow, admin login, session middleware
│   ├── benchmark_data.go           # CSV parsing, binary storage (V2), streaming JSON/ZIP export
│   ├── benchmarks.go               # Benchmark CRUD handlers (create/read/update/delete/search)
│   ├── config.go                   # Configuration parsing (flags + env vars)
│   ├── database.go                 # GORM/SQLite initialization, admin user seeding
│   ├── mcp.go                      # MCP server (JSON-RPC 2.0) with 17 tools
│   ├── migration.go                # Database schema versioning and migrations
│   ├── models.go                   # GORM models: User, Benchmark, APIToken, AuditLog
│   ├── ratelimiter.go              # In-memory sliding window rate limiter
│   ├── server.go                   # HTTP server setup, all route definitions
│   ├── storage_migration.go        # Benchmark data file format V1→V2 migration
│   ├── test_helpers.go             # Shared test utilities (setupTestDB, cleanupTestDB)
│   ├── web.go                      # Embedded SPA serving with fallback routing
│   └── *_test.go                   # Comprehensive test files (18 test files)
├── testdata/                       # Real benchmark CSV files for parsing tests
│   ├── afterburner/                # Afterburner HML format samples
│   └── mangohud/                   # MangoHud CSV format samples
├── web/                            # Vue.js frontend
│   ├── src/
│   │   ├── api/client.js           # Centralized API client with error handling
│   │   ├── components/             # Reusable components (Navbar, BenchmarkCharts)
│   │   ├── router/index.js         # Route definitions with navigation guards
│   │   ├── stores/                 # Pinia stores (auth, app state)
│   │   ├── utils/                  # Pure utility functions (stats, date formatting, validation)
│   │   ├── views/                  # Page-level components (8 views)
│   │   └── workers/                # Web Workers for CPU-intensive operations
│   ├── tests/                      # Frontend tests (Playwright E2E + unit tests)
│   ├── eslint.config.js            # ESLint flat config with Vue rules
│   ├── playwright.config.js        # Playwright E2E configuration
│   └── vite.config.js              # Vite build config with API proxy and chunk splitting
├── backend_test.sh                 # 20-scenario backend integration test script (bash + curl + jq)
├── .golangci.yml                   # Go linter config (21 linters enabled, strict rules)
├── Makefile                        # Build targets: build, build-web, build-server, clean, test
├── Dockerfile                      # Multi-stage: Node build → Go build → Alpine runtime
├── docker-compose.yml              # Local development setup
└── .env.example                    # Environment variable template
```

---

## Database Models

### User
| Field | Type | Notes |
|-------|------|-------|
| ID | uint (PK) | Auto-increment |
| DiscordID | string | Unique; "admin" for system admin |
| Username | string | Display name |
| IsAdmin | bool | Admin privileges |
| IsBanned | bool | Login blocked |
| LastWebActivityAt | *time.Time | Nullable, updated on session auth |
| LastAPIActivityAt | *time.Time | Nullable, updated on token auth |
| Benchmarks | []Benchmark | Has-many, cascade delete |
| APITokens | []APIToken | Has-many, cascade delete |

### Benchmark
| Field | Type | Notes |
|-------|------|-------|
| ID | uint (PK) | Also used as filesystem directory name |
| UserID | uint (FK) | Owner |
| Title | string | Max 100 chars |
| Description | string | Max 5,000 chars, Markdown supported |
| RunNames | string | Comma-separated labels for search indexing |
| Specifications | string | Concatenated specs for search indexing |

### APIToken
| Field | Type | Notes |
|-------|------|-------|
| ID | uint (PK) | Auto-increment |
| UserID | uint (FK) | Owner |
| Token | string | 64-char hex (32 random bytes), unique |
| Name | string | Max 100 chars |
| LastUsedAt | *time.Time | Updated on each API call |

### AuditLog
| Field | Type | Notes |
|-------|------|-------|
| ID | uint (PK) | Auto-increment |
| UserID | uint | Who performed the action |
| Action | string | Max 100 chars (e.g. "benchmark_created") |
| Description | string | Max 1,000 chars |
| TargetType | string | Max 50 chars (e.g. "benchmark", "user") |
| TargetID | uint | ID of the affected entity |

---

## API Endpoints

### Public (No Auth)
| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/health` | inline | Health check (returns `{"status":"ok","version":"..."}`) |
| GET | `/auth/login` | HandleLogin | Initiates Discord OAuth flow |
| GET | `/auth/login/callback` | HandleLoginCallback | Discord OAuth callback |
| GET | `/api/benchmarks` | HandleListBenchmarks | List/search benchmarks (paginated) |
| GET | `/api/benchmarks/:id` | HandleGetBenchmark | Get benchmark metadata |
| GET | `/api/benchmarks/:id/data` | HandleGetBenchmarkData | Stream benchmark statistics as JSON |
| GET | `/api/benchmarks/:id/runs/:runIndex` | HandleGetBenchmarkRun | Get single run statistics |
| GET | `/api/benchmarks/:id/download` | HandleDownloadBenchmarkData | Download benchmark as ZIP of CSVs |
| GET | `/api/auth/me` | HandleGetCurrentUser | Current user info (or 401) |

### Authenticated (Session or Bearer Token)
| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/auth/admin/login` | HandleAdminLogin | Admin username/password login |
| POST | `/auth/logout` | HandleLogout | End session |
| POST | `/api/benchmarks` | HandleCreateBenchmark | Upload benchmark (multipart form) |
| PUT | `/api/benchmarks/:id` | HandleUpdateBenchmark | Update title/description/labels |
| DELETE | `/api/benchmarks/:id` | HandleDeleteBenchmark | Delete benchmark and data files |
| POST | `/api/benchmarks/:id/runs` | HandleAddBenchmarkRuns | Add runs to existing benchmark |
| DELETE | `/api/benchmarks/:id/runs/:run_index` | HandleDeleteBenchmarkRun | Remove a specific run |
| GET | `/api/tokens` | HandleListAPITokens | List user's API tokens |
| POST | `/api/tokens` | HandleCreateAPIToken | Create API token (max 10/user) |
| DELETE | `/api/tokens/:id` | HandleDeleteAPIToken | Delete API token |

### Admin Only (Auth + IsAdmin)
| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/admin/users` | HandleListUsers | List users (paginated, searchable) |
| DELETE | `/api/admin/users/:id` | HandleDeleteUser | Delete user account |
| DELETE | `/api/admin/users/:id/benchmarks` | HandleDeleteUserBenchmarks | Delete user's benchmarks |
| PUT | `/api/admin/users/:id/ban` | HandleBanUser | Ban/unban user |
| PUT | `/api/admin/users/:id/admin` | HandleToggleUserAdmin | Grant/revoke admin |
| GET | `/api/admin/logs` | HandleListAuditLogs | View audit logs (filtered) |

### MCP
| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/mcp` | HandleMCP | JSON-RPC 2.0 MCP protocol |
| GET | `/mcp` | HandleMCPGet | Returns 405 (SSE not supported) |
| DELETE | `/mcp` | HandleMCPDelete | Session termination |

---

## MCP Tools (17 tools)

The MCP server exposes all non-file-transfer API functionality as tools. File upload/download operations are intentionally excluded because MCP is not suited for large binary transfers.

### Public (no auth required)
- `list_benchmarks` – Browse/search with pagination, user_id filter, sorting
- `get_benchmark` – View benchmark metadata
- `get_benchmark_data` – Statistics for all runs (min, max, avg, median, percentiles, stddev, variance), optional raw data points (downsampled to max 5000)
- `get_benchmark_run` – Statistics for a single run

### Authenticated (Bearer token)
- `get_current_user` – Authenticated user info
- `update_benchmark` – Edit title/description/labels (owner or admin)
- `delete_benchmark` – Delete benchmark (owner or admin)
- `delete_benchmark_run` – Delete specific run (cannot delete last run)
- `list_api_tokens` – List user's tokens
- `create_api_token` – Create token (max 10/user)
- `delete_api_token` – Delete token

### Admin (Bearer token + admin)
- `list_users` – Search users by username/Discord ID
- `list_audit_logs` – Filter logs by user_id, action, target_type
- `delete_user` – Delete user (cannot delete self)
- `delete_user_benchmarks` – Delete all user's benchmarks
- `ban_user` – Ban/unban (cannot ban self)
- `toggle_user_admin` – Grant/revoke admin (cannot revoke self)

### API vs MCP Parity

Everything in the REST API has an MCP equivalent **except** these intentionally excluded operations:
- **Benchmark file upload** (`POST /api/benchmarks`, `POST /api/benchmarks/:id/runs`) – requires multipart file transfer, unsuitable for MCP
- **Benchmark ZIP download** (`GET /api/benchmarks/:id/download`) – large binary transfer, unsuitable for MCP

When adding new API endpoints, always add the corresponding MCP tool unless it involves binary file transfer. See section "PR Checklist" rule 5.

---

## Benchmark Data Processing

### Supported Formats
1. **MangoHud CSV** – Header starts with `os,cpu,gpu,ram,kernel,driver,cpuscheduler`
2. **Afterburner HML** – Header starts with `Hardware monitoring log v`

### Metrics Extracted (13)
FPS, Frametime, CPU Load, GPU Load, CPU Temp, CPU Power, GPU Temp, GPU Core Clock, GPU Mem Clock, GPU VRAM Used, GPU Power, RAM Used, Swap Used

### Limits
- Max total data lines across all runs: **1,000,000**
- Max data lines per single run: **500,000**
- Rate limit for uploads: **5 per 10 minutes** (non-admins)
- Max benchmark title: **100 chars**
- Max benchmark description: **5,000 chars**
- Max API tokens per user: **10**

### Storage Format (V2)
Benchmark data is stored as zstd-compressed gob encoding with a header containing the format version and run count, followed by individually encoded runs. This enables streaming reads without loading all data into memory. A companion `.meta` JSON file stores run count and labels for quick metadata access.

### Streaming Architecture
- `StreamBenchmarkDataAsJSON` – Writes one run at a time to the HTTP response, triggering GC every 10 runs
- `ExportBenchmarkDataAsZip` – Streams each run as a separate CSV into a ZIP archive, triggering GC every 5 runs
- Memory-efficient two-pass CSV parsing: first pass counts lines, second pass pre-allocates exact capacity

---

## Authentication Flow

### Discord OAuth
1. `GET /auth/login` → Generates random state token, redirects to Discord
2. Discord callback → Exchanges code for token → Fetches Discord user info
3. Creates or updates User record → Sets session (UserID, Username, IsAdmin)
4. Checks ban status before completing login

### Admin Login
- `POST /auth/admin/login` with username/password
- Rate limited: 3 failed attempts per 10 minutes (global lock)
- Creates/updates system admin user with DiscordID="admin"

### API Token Auth
- `Authorization: Bearer <64-char-hex-token>` header
- Middleware `RequireAuthOrToken` checks session first, then Bearer token
- Tracks `LastUsedAt` on token and `LastAPIActivityAt` on user

### Session Configuration
- HttpOnly cookies, SameSite=Lax
- Configurable Secure flag via `FS_SESSION_SECURE` env var
- Session stores: UserID, Username, IsAdmin

---

## Testing

### Test Pyramid

This project has comprehensive testing at every level. **Everything must be tested.** Any code change must have corresponding test coverage.

#### 1. Go Unit Tests (`internal/app/*_test.go` – 18 test files)
- **Framework:** Go standard `testing` package
- **Pattern:** Table-driven tests with nested `t.Run()` subtests
- **Database:** Isolated temp SQLite databases via `setupTestDB()` + `t.TempDir()`
- **HTTP handlers:** `gin.TestMode` + `httptest.NewRecorder` with mock sessions
- **Test helpers:** `internal/app/test_helpers.go` (setupTestDB, cleanupTestDB, createTestUser, createMultipartFileHeaders, setupTestRouter)
- **Run:** `go test -v -race ./...`
- **Coverage:** `go test -v -race -coverprofile=coverage.out ./...`

Test files and what they cover:
| File | Coverage Area |
|------|--------------|
| `admin_test.go` | Admin handlers: list/delete/ban/admin-toggle users, self-protection |
| `api_tokens_test.go` | Token CRUD, auth middleware, token limits |
| `audit_test.go` | Audit log creation, filtering, pagination |
| `auth_test.go` | OAuth flow, sessions, cookie security flags |
| `benchmark_data_test.go` | File storage/retrieval, metadata, V1/V2 backward compat |
| `benchmark_export_test.go` | CSV export, ZIP creation, filename sanitization |
| `benchmark_error_test.go` | Multipart upload error handling, error messages |
| `benchmark_line_limit_test.go` | Per-run and total line limits validation |
| `benchmark_memory_test.go` | Memory usage with 100 runs × 10k data points |
| `benchmark_roundtrip_test.go` | Export → re-import CSV roundtrip data integrity |
| `benchmark_streaming_test.go` | Streaming 1M points with minimal memory overhead |
| `benchmarks_test.go` | List/get/delete benchmarks, search, run management |
| `config_test.go` | Config flag parsing |
| `mcp_test.go` | All 17 MCP tools: requests, responses, auth, errors |
| `migration_test.go` | Schema migrations, backward compat, timestamp preservation |
| `ratelimiter_test.go` | Rate limit logic, sliding window, cleanup |
| `ratelimiter_integration_test.go` | Rate limits applied to login/upload handlers |
| `testdata_parsing_test.go` | Real Afterburner/MangoHud file parsing + roundtrip |

#### 2. Go Linting (`.golangci.yml`)
- **21 linters enabled:** errcheck, govet, ineffassign, staticcheck, unused, misspell, unconvert, unparam, bodyclose, noctx, gosec, gocritic, revive, prealloc, copyloopvar, nilerr, errorlint, goprintffuncname, nolintlint, and more
- **Strict mode:** All gocritic checks enabled (diagnostic, style, performance, experimental, opinionated)
- **Test file relaxations:** gosec, gocritic, noctx, bodyclose disabled in `*_test.go`
- **Run:** `golangci-lint run --timeout=5m`

#### 3. Backend Integration Tests (`backend_test.sh` – 20 scenarios)
- **Framework:** Bash script using curl + jq
- **Scenarios:** Full API workflow: health → admin login → CRUD benchmarks → search → download → audit logs → API tokens → delete → logout → verify
- **Run:** `./backend_test.sh` (requires built server binary, set `PREBUILT_SERVER=./server`)
- **Environment:** Starts server with test credentials, uses temp data directory

#### 4. Frontend E2E Tests (`web/tests/basic.spec.js`)
- **Framework:** Playwright (Chromium)
- **Coverage:** Homepage loading, health endpoint, navigation, benchmark list, API responses, URL redirects
- **Config:** `web/playwright.config.js` – base URL `http://localhost:5000`, 2 retries in CI, screenshots on failure
- **Run:** `cd web && npm test`

#### 5. Frontend Unit Tests (`web/tests/*.test.js`)
- **Framework:** Native Node.js (no Jest/Vitest – custom assertions)
- **Files:** `benchmarkDataProcessor.test.js`, `dateFormatter.test.js`, `filenameValidator.test.js`
- **Run:** `node web/tests/<file>.test.js`
- **Coverage:** Stats calculations (percentiles, FPS from frametime), date formatting (relative dates, edge cases), filename validation (date pattern detection)

#### 6. Frontend Linting (`web/eslint.config.js`)
- **Framework:** ESLint with flat config
- **Rules:** JS recommended + Vue recommended + strict equality + no var + no eval
- **Run:** `cd web && npm run lint`

---

## CI/CD Pipeline

### test.yml (runs on PRs to main and pushes to main)

6 parallel/sequential jobs:

1. **lint-go** – `golangci-lint` with 5-minute timeout
2. **lint-frontend** – `cd web && npm run lint`
3. **unit-tests** – `go test -v -race -coverprofile=coverage.out ./...` + codecov upload
4. **build** – `make build` (full build with embedded web UI), uploads binary artifact
5. **backend-integration-test** (needs: build) – Downloads binary, runs `./backend_test.sh`
6. **e2e-tests** (needs: build) – Downloads binary, starts server, runs Playwright

### Deployment Workflows
- **deploy.yml** – Pushes to non-main branches auto-deploy to dev via SSH + Docker
- **release.yml** – GitHub Release triggers production deployment + GHCR image push
- **deploy-prod-manual.yml** – Manual production deployment
- **sync-prod-to-dev-data.yml** – Syncs production DB to dev

---

## Build Commands

| Command | Purpose |
|---------|---------|
| `make build` | Full build: web UI + Go binary with embedded assets |
| `make build-web` | Build Vue.js frontend only (`web/dist/`) |
| `make build-server` | Build Go binary with version from git tags |
| `make clean` | Remove build artifacts, node_modules, dist |
| `make run` | `go run ./cmd/server` (development, no web UI) |
| `make test` | `go test -v ./...` |
| `make test-integration` | `./backend_test.sh` |
| `go test -v -race ./...` | Unit tests with race detection |
| `cd web && npm run dev` | Frontend dev server (port 3000, proxies to 5000) |
| `cd web && npm run build` | Production frontend build |
| `cd web && npm run lint` | ESLint check |
| `cd web && npm test` | Playwright E2E tests |

---

## Configuration

All settings can be provided as CLI flags or environment variables (prefix `FS_`):

| Flag | Env Var | Default | Required | Purpose |
|------|---------|---------|----------|---------|
| `-bind` | `FS_BIND` | `0.0.0.0:5000` | No | Server address |
| `-data-dir` | `FS_DATA_DIR` | `/data` | No | Data storage directory |
| `-session-secret` | `FS_SESSION_SECRET` | – | Yes | Cookie encryption key |
| `-session-secure` | `FS_SESSION_SECURE` | `true` | No | HTTPS-only cookies |
| `-discord-client-id` | `FS_DISCORD_CLIENT_ID` | – | Yes | Discord OAuth app ID |
| `-discord-client-secret` | `FS_DISCORD_CLIENT_SECRET` | – | Yes | Discord OAuth secret |
| `-discord-redirect-url` | `FS_DISCORD_REDIRECT_URL` | – | Yes | OAuth callback URL |
| `-admin-username` | `FS_ADMIN_USERNAME` | – | Yes | Admin login username |
| `-admin-password` | `FS_ADMIN_PASSWORD` | – | Yes | Admin login password |

Memory tuning via environment variables:
- `GOGC` – Garbage collection target percentage (app default: 50, more aggressive than Go's standard default of 100; set in `cmd/server/main.go`)
- `GOMEMLIMIT` – Soft memory limit (e.g., "512MiB")

---

## PR Checklist

Every pull request **must** satisfy ALL of the following requirements:

### 1. All Tests Must Pass
- **Go linter:** `golangci-lint run --timeout=5m` must pass with zero warnings
- **Go unit tests:** `go test -v -race ./...` must pass
- **Frontend lint:** `cd web && npm run lint` must pass
- **Build:** `make build` must succeed
- **Backend integration tests:** `./backend_test.sh` must pass all 20 scenarios
- **E2E tests:** Playwright tests must pass
- PRs that touch Go code must run ALL Go-related checks (lint + unit tests + build + integration tests). PRs that touch frontend code must run all frontend checks (lint + E2E + unit tests). Ensure all code compiles, lints, and tests pass before requesting review.

### 2. No Dead Code
- Double-check for any redundant or dead code introduced in the PR
- Remove unused imports, variables, functions, and types
- Remove commented-out code blocks
- The Go linter (especially `unused`, `unparam`, `ineffassign`) will catch most issues, but manually verify too
- Check for unused CSS classes, Vue components, or JavaScript functions in frontend changes

### 3. Full Test Coverage
- Every new function, handler, or utility must have corresponding tests
- Every new API endpoint must be covered by unit tests AND integration tests in `backend_test.sh`
- Every new MCP tool must be covered in `mcp_test.go`
- Every new frontend utility function must have a test file in `web/tests/`
- Every new Vue view/component must be covered by Playwright E2E tests
- Test both success and error paths
- Test edge cases and boundary conditions

### 4. Manual Verification
- Every feature must be tested at least once before submitting the PR
- For backend changes: use curl or the backend integration test script to verify endpoints
- For frontend changes: use Playwright to verify UI behavior, take screenshots of UI changes
- For API changes: verify request/response format matches documentation
- For MCP changes: verify tool input/output schemas and error handling

### 5. API–MCP Parity
- Every new REST API endpoint must have a corresponding MCP tool, **unless** the operation involves binary file transfer (multipart uploads or file downloads)
- Currently excluded from MCP (intentionally): benchmark file upload (`POST /api/benchmarks`, `POST /api/benchmarks/:id/runs`) and benchmark ZIP download (`GET /api/benchmarks/:id/download`)
- When adding a new API endpoint, add the MCP tool in `internal/app/mcp.go` and add corresponding tests in `internal/app/mcp_test.go`
- Verify that MCP tool parameters, responses, and error handling match the REST API behavior

### 6. Security and Performance
- Follow security best practices: validate all inputs, use parameterized queries (GORM handles this), sanitize HTML output (DOMPurify on frontend)
- Never expose sensitive data (tokens, passwords, session secrets) in responses or logs
- Memory efficiency takes priority over CPU efficiency
- Use streaming instead of loading entire datasets into memory where possible
- Pre-allocate slices with known capacity instead of growing dynamically
- Trigger garbage collection in loops processing large datasets (see `gcFrequencyStreaming` and `gcFrequencyExport` patterns in `benchmark_data.go`)
- Use deferred cleanup for all file handles and resources
- Rate limit user-facing write operations
- Check file sizes and data line counts before processing

### 7. Documentation Updates
- Every functionality change requires a documentation update in the `docs/` directory
- Update the relevant section in the docs to reflect the change
- If adding a new feature, add a new documentation page or section
- Keep API endpoint documentation current with any route changes
- Document any new configuration options or environment variables

---

## Code Style and Patterns

### Go Backend

- **Error handling:** Return early on errors. Non-critical errors (audit logging, activity tracking) are logged with `fmt.Printf` but don't fail the operation.
- **Handler pattern:** Validate input → Check auth/permissions → Execute operation → Audit log → Return JSON response
- **Pagination:** `page` (default 1) + `per_page` (default varies, max 100). Responses include `total`, `page`, `per_page`, `total_pages`.
- **Search:** Multi-field LIKE queries with minimum 3-character search terms
- **Testing:** Table-driven tests with `t.Run()` subtests. Use `setupTestDB()`/`cleanupTestDB()` for database tests. Use `httptest.NewRecorder` for handler tests.
- **Memory management:** Two-pass file parsing (count lines, then pre-allocate). Streaming for large data. Periodic GC in loops.
- **Naming:** Standard Go conventions. Handlers are `Handle<Action>` (e.g., `HandleCreateBenchmark`). Test files match source files (e.g., `benchmarks.go` → `benchmarks_test.go`).

### Vue.js Frontend

- **Composition API:** All components use `<script setup>` syntax
- **API calls:** Always go through `web/src/api/client.js` – never use `fetch` directly in components
- **State management:** Pinia stores for auth and app-wide state. Local `ref()`/`reactive()` for component state.
- **Error handling:** API client throws `APIError` with structured error information. Components catch and display errors.
- **Heavy computation:** Offload to Web Workers (`workers/`) to keep UI responsive
- **Routing:** Lazy-loaded views via Vue Router. Navigation guard prevents authenticated users from seeing login page.
- **Styling:** Bootstrap 5 utility classes. No custom CSS framework.

### Testing Patterns

- **Go tests:** Use `t.Helper()` in test utilities. Use `t.Fatal()` / `t.Fatalf()` for setup failures. Use `t.Errorf()` for assertion failures. Clean up with `defer cleanupTestDB()`.
- **Frontend unit tests:** Custom assertion functions (`assertApprox`, `assertEquals`, `assertMatch`). No external test library – pure Node.js.
- **Playwright E2E:** Use `page.goto()`, `page.locator()`, `expect()`. Tests against running server on `localhost:5000`.
- **Integration tests:** Bash functions with curl + jq. Check HTTP status codes and JSON response fields. Sequential test execution with shared state (session cookie, created IDs).

---

## Key Implementation Details

### Rate Limiter
- In-memory sliding window implementation in `ratelimiter.go`
- Two instances: benchmark uploads (5/10min per user), admin login (3/10min global)
- Background cleanup goroutine runs every 5 minutes
- Thread-safe with `sync.RWMutex`

### Database Migrations
- Schema version tracked in `schema_versions` table
- Current version: 3
- Migrations: v0→v1 (remove ai_summary), v1→v2 (add search fields), v2→v3 (storage format V2 + metadata)
- Detect old database formats and migrate automatically

### Benchmark Data Format V2
- Header: format version + run count
- Body: individually gob-encoded runs compressed with zstd
- Enables streaming reads without loading entire dataset
- Backward compatible with V1 (fallback on header decode failure)

### SPA Routing
- `/assets/*`, `/favicon.*` → Static files from embedded filesystem
- `/api/*`, `/auth/*`, `/health` → API routes (return 404 for unknown paths)
- Everything else → `index.html` (Vue Router handles client-side routing)
- Legacy URL redirect: `/benchmark/:id` → `/benchmarks/:id`

### Searchable Metadata
- `Benchmark.RunNames` – Comma-separated run labels for full-text search
- `Benchmark.Specifications` – Concatenated unique specs (OS, CPU, GPU, RAM, kernel, scheduler)
- Both fields populated on create/update, searched via LIKE queries

---

## Docker

### Multi-Stage Build
1. **web-builder** (Node 25-alpine): `npm ci && npm run build` → produces `web/dist/`
2. **builder** (Go 1.26-alpine): Copies `web/dist/`, generates embed file, `go build` with ldflags version
3. **runtime** (Alpine 3.23): Copies binary + CA certs, exposes port 5000, data volume at `/data`

### Local Development
```bash
# Full stack via Docker Compose
docker compose up

# Backend only (Go)
make run  # or: go run ./cmd/server -bind=... -data-dir=... etc.

# Frontend only (with API proxy)
cd web && npm run dev  # Vite dev server on port 3000, proxies /api to port 5000
```

---

## Common Tasks

### Adding a New API Endpoint
1. Define the handler in `internal/app/<domain>.go`
2. Register the route in `internal/app/server.go`
3. Add unit tests in `internal/app/<domain>_test.go`
4. Add integration test scenario in `backend_test.sh`
5. Add corresponding MCP tool in `internal/app/mcp.go` (unless file transfer)
6. Add MCP tool tests in `internal/app/mcp_test.go`
7. Update documentation in `docs/`

### Adding a New Vue Page
1. Create component in `web/src/views/<Name>.vue`
2. Add route in `web/src/router/index.js`
3. Add API methods in `web/src/api/client.js` if needed
4. Add Playwright E2E test in `web/tests/basic.spec.js`
5. Add unit tests for any new utility functions in `web/tests/`

### Adding a New Database Field
1. Add field to model in `internal/app/models.go`
2. GORM auto-migrates on startup, but for complex changes, add a migration in `internal/app/migration.go`
3. Increment schema version
4. Update all relevant handlers and MCP tools
5. Update tests to cover the new field
6. Update documentation

### Modifying Benchmark Data Processing
1. Edit `internal/app/benchmark_data.go`
2. Maintain backward compatibility with existing stored data
3. Update storage format version if binary format changes
4. Add migration in `internal/app/storage_migration.go` if needed
5. Test with real data files in `testdata/`
6. Verify streaming and memory behavior with large datasets
