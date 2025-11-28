# FlightlessSomething - Copilot Instructions

## Repository Overview

**FlightlessSomething** is a web application for storing and managing gaming benchmark data with Discord OAuth authentication and a modern Vue.js interface. The repository is ~162MB with 26 Go files (~5,800 lines) and a Vue.js frontend.

**Tech Stack:**
- **Backend:** Go 1.25, Gin web framework, GORM ORM, SQLite database, Discord OAuth2, zstd compression
- **Frontend:** Vue.js 3 (Composition API), Vite build tool, Vue Router, Pinia state management, Bootstrap, Highcharts
- **Build Tools:** Make, npm, Docker
- **Target Runtime:** Linux/amd64 (primary), containerized deployment

## Build & Test Commands

### Prerequisites
- Go 1.25+ (command: `go version` to verify)
- Node.js 20+ (command: `node --version` to verify)
- golangci-lint for linting (install if needed: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest`)
- jq for integration tests (usually pre-installed on CI)

### Build Process (VALIDATED)

**IMPORTANT:** Always build the web UI before building the Go server. The server embeds the web UI at build time.

```bash
# Clean build (removes all artifacts)
make clean

# Build everything (web UI + server + migrate tool)
make build

# Build only web UI (required before server build)
make build-web
# This runs: cd web && npm install && npm run build

# Build only server (requires web UI built first)
make build-server
# Creates internal/app/webfs_embed.go and embeds web/dist into the binary

# Build only migration tool
make build-migrate
```

**Build artifacts:**
- `server` - Main server binary (~36MB)
- `migrate` - Migration tool binary
- `web/dist/` - Compiled web UI assets (automatically embedded into server)
- `internal/app/web/` and `internal/app/webfs_embed.go` - Temporary files (auto-cleaned after build)

**Timing:**
- `npm install` (fresh): ~4 seconds
- `npm run build`: ~2-3 seconds
- `make build` (full, clean): ~10-15 seconds
- `go test ./...` (with downloads): ~60-80 seconds, subsequent runs: ~5 seconds

### Testing (VALIDATED)

```bash
# Backend unit tests (always run before committing)
go test ./...
# Or with coverage:
go test -v -race -coverprofile=coverage.out ./...

# Backend integration tests (comprehensive API testing)
./backend_test.sh
# Requires: jq, builds server if not present
# Tests 20+ scenarios including auth, CRUD, uploads
# Uses PREBUILT_SERVER env var in CI to skip rebuild

# Frontend unit tests
cd web && npm run test:unit

# Frontend E2E tests (Playwright)
cd web && npm test
# Requires server running on localhost:5000
# First time: npx playwright install --with-deps chromium
```

### Linting (VALIDATED)

```bash
# Backend linting (must pass in CI)
golangci-lint run --timeout=5m
# Config: .golangci.yml (33 linters enabled, see file for details)
# Current status: 0 issues

# Frontend linting (warnings acceptable, errors must be fixed)
cd web && npm run lint
# Auto-fix: npm run lint:fix
# Current status: 590 warnings (style issues), 0 errors
```

## Project Structure

### Root Directory Files
```
.env.example          # Environment variables template
.gitignore           # Ignores: data/, *.db, web/node_modules, server, migrate
.golangci.yml        # Go linting configuration (33 linters)
Dockerfile           # Multi-stage build (golang:1.25-alpine + alpine)
Makefile             # Build automation (build, clean, test targets)
README.md            # Quick start guide
backend_test.sh      # Integration test script (executable)
docker-compose.yml   # Local development with Docker
go.mod, go.sum       # Go dependencies (Go 1.25 required)
test_migration.sh    # Database migration test script
```

### Directory Structure
```
cmd/
├── server/main.go   # Server entry point (~435 lines)
└── migrate/main.go  # Migration tool entry point (~10KB)

internal/app/        # Core application code (~5,800 lines total)
├── config.go        # CLI flags and environment variable parsing
├── server.go        # Gin router setup, middleware, routes
├── database.go      # GORM setup and auto-migration
├── models.go        # Database models (User, Benchmark, APIToken, AuditLog)
├── auth.go          # Discord OAuth + admin authentication
├── benchmarks.go    # Benchmark CRUD handlers
├── benchmark_data.go    # Binary data compression/decompression
├── admin.go         # Admin endpoints (users, audit logs)
├── api_tokens.go    # API token management
├── ratelimiter.go   # Rate limiting middleware
├── web.go           # SPA serving from embedded filesystem
└── *_test.go        # Test files (comprehensive coverage)

web/                 # Vue.js frontend
├── src/
│   ├── main.js      # Vue app initialization
│   ├── App.vue      # Root component with Navbar
│   ├── router/index.js        # Client-side routing
│   ├── stores/               # Pinia stores (auth, app state)
│   ├── components/           # Reusable components (Navbar, Charts)
│   ├── views/                # Page components (Login, Benchmarks, etc.)
│   ├── api/client.js         # Axios-based API client
│   └── utils/dateFormatter.js # Date formatting utilities
├── tests/           # Unit and E2E tests
├── package.json     # npm scripts and dependencies
├── vite.config.js   # Vite build configuration
└── playwright.config.js  # E2E test configuration

docs/                # Comprehensive documentation
├── development.md   # Architecture, development setup
├── testing.md       # Testing guide (all test types)
├── api.md          # REST API documentation
├── benchmarks.md   # Benchmark data format guide
├── deployment.md   # Production deployment guide
├── migration.md    # Version migration guide
└── webui.md        # Frontend development guide

testdata/            # Test fixtures (benchmark CSV files)
.github/
├── workflows/
│   ├── test.yml     # CI pipeline (lint, test, build, integration, E2E)
│   ├── deploy.yml   # Dev deployment (auto on non-main branches)
│   ├── release.yml  # Release + prod deployment (on GitHub releases)
│   └── deploy-prod-manual.yml  # Manual prod deployment
└── actions/deploy-via-ssh/  # Reusable deployment action
```

## CI/CD Pipeline

### Test Workflow (`.github/workflows/test.yml`)
**Triggers:** Every push to main, all PRs
**Jobs (all must pass):**
1. `lint-go` - golangci-lint with Go 1.25
2. `lint-frontend` - ESLint in web/ directory
3. `unit-tests` - Go tests with race detector and coverage
4. `build` - Full build (web UI + server), uploads server binary as artifact
5. `backend-integration-test` - Runs backend_test.sh with pre-built server
6. `e2e-tests` - Playwright tests with server started in background

**Environment Setup in CI:**
- Go 1.25, Node 20, chromium for Playwright
- jq installed for integration tests
- Server runs on port 5000 with test credentials

### Deployment Workflows
- **Deploy Dev:** Auto-deploys non-main branches to dev environment
- **Release:** Builds Docker image, pushes to GHCR, deploys to prod on GitHub releases
- **Manual Prod Deploy:** Workflow_dispatch for emergency deployments

All workflows use Docker for deployment (see Dockerfile for multi-stage build process).

## Critical Build Requirements

### ⚠️ Web UI Must Be Built Before Server
The Go server uses `//go:embed all:web/dist` to embed the Vue.js frontend. The Makefile handles this automatically:
1. Builds web UI to `web/dist/`
2. Copies `web/dist/` to `internal/app/web/`
3. Generates `internal/app/webfs_embed.go` with embed directive
4. Builds server binary
5. Cleans up temporary files (`internal/app/web/`, `webfs_embed.go`)

**DO NOT** commit `internal/app/webfs_embed.go` or `internal/app/web/` - they are build artifacts.

### Running the Server
```bash
./server \
  -bind="0.0.0.0:5000" \
  -data-dir="./data" \
  -session-secret="your-secret" \
  -discord-client-id="your-id" \
  -discord-client-secret="your-secret" \
  -discord-redirect-url="http://localhost:5000/auth/login/callback" \
  -admin-username="admin" \
  -admin-password="admin"
```

All flags can be set via environment variables with `FS_` prefix (e.g., `FS_BIND`, `FS_DATA_DIR`).

## Common Pitfalls & Workarounds

1. **Build fails with "pattern all:web/dist: no matching files found"**
   - Cause: Web UI not built before server build
   - Fix: Run `make build-web` or `make build` (not `go build` directly)

2. **Tests fail with database errors**
   - Cause: Test database cleanup issues or parallel test conflicts
   - Fix: Tests use unique database files (`setupTestDB` in test_helpers.go)
   - Each test should call `setupTestDB(t)` and `defer cleanupTestDB(t, db)`

3. **Frontend linting shows 590 warnings**
   - Expected: These are style warnings (attribute ordering, formatting)
   - Action: Fix only errors (currently 0), warnings are acceptable
   - Use `npm run lint:fix` to auto-fix some issues

4. **Integration tests timeout or fail**
   - Check: Server must finish starting (check `/tmp/backend-server.log`)
   - Wait time: Script waits 5 seconds after server start
   - In CI: Uses pre-built server binary to save time

5. **Go module downloads slow down first test run**
   - Expected: First `go test` or `go build` downloads dependencies (~60s)
   - Subsequent runs use cached modules (~5s)

6. **Playwright tests fail with "Target closed"**
   - Cause: Server not running or crashed
   - Fix: Start server separately on port 5000 before running `npm test`

## Development Workflow

### Making Backend Changes
1. Edit Go files in `internal/app/`
2. Run `go test ./...` to verify tests pass
3. Run `golangci-lint run --timeout=5m` to check linting
4. If adding API endpoints, update `docs/api.md`
5. Test with: `make build && ./server [flags]`

### Making Frontend Changes
1. Edit Vue files in `web/src/`
2. Run `cd web && npm run lint` to check linting
3. Run `cd web && npm run build` to verify build succeeds
4. Test with: `make build && ./server [flags]` (tests embedded UI)
5. For live development: `cd web && npm run dev` (not production mode)

### Adding Dependencies
- **Go:** Edit `go.mod` manually or use `go get`, then run `go mod tidy`
- **Frontend:** `cd web && npm install <package>`
- Always commit updated `go.mod`, `go.sum`, `web/package.json`, `web/package-lock.json`

## Configuration Files

- **.golangci.yml:** Comprehensive linting (errcheck, govet, staticcheck, gosec, etc.)
- **web/eslint.config.js:** ESLint with Vue plugin
- **web/vite.config.js:** Vite build settings (minification, chunking)
- **web/playwright.config.js:** E2E test configuration (baseURL: localhost:5000)

## Data Storage Architecture

- **Metadata:** SQLite database (`data/flightlesssomething.db`)
  - Users, benchmarks metadata, API tokens, audit logs
- **Benchmark Data:** Compressed binary files (`data/benchmarks/*.bin`)
  - zstd compression for efficiency
  - Metadata cache in `.meta` files
- **Auto-migration:** GORM auto-migrates schema on server startup

## Key Validation Steps

Before committing:
```bash
# 1. Lint code
golangci-lint run --timeout=5m
cd web && npm run lint

# 2. Run tests
go test ./...
./backend_test.sh

# 3. Verify build
make clean && make build

# 4. Optional: E2E tests
cd web && npm test  # (requires server running)
```

## Trust These Instructions

These instructions are validated and comprehensive. Rely on them for:
- Build commands and order
- Test execution
- Known issues and workarounds
- CI/CD pipeline behavior

Only search the codebase or run exploratory commands if:
- Instructions are incomplete for your specific task
- You encounter errors not documented here
- You need to understand implementation details beyond build/test

The documentation in `docs/` provides additional context but these instructions cover all build, test, and validation steps needed for coding tasks.
