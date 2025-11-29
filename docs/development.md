# Development Guide

## Architecture

### Backend (Go)

**Framework**: Gin HTTP framework
**ORM**: GORM with SQLite
**Config**: ff/v3 for CLI and environment variables

**Structure**:
```
cmd/server/          # Entry point
internal/app/        # Application core
  ├── config.go         # Configuration
  ├── server.go         # Server setup
  ├── models.go         # Database models
  ├── auth.go           # Authentication
  ├── benchmarks.go     # Benchmark handlers
  ├── admin.go          # Admin handlers
  ├── ratelimiter.go    # Rate limiting
  └── *_test.go         # Tests
```

### Frontend (Vue.js)

**Framework**: Vue.js 3 with Composition API
**Build**: Vite
**Router**: Vue Router
**State**: Pinia

See [Web UI Guide](webui.md) for details.

### Data Storage

**Metadata**: SQLite database
- Users
- Benchmarks (title, description, timestamps)
- API tokens
- Audit logs

**Benchmark Data**: Compressed binary files (zstd)
- Prevents database slowdown
- Efficient storage
- Fast read/write

**File Structure**:
```
data/
├── flightlesssomething.db    # SQLite database
└── benchmarks/
    ├── 1.bin                 # Compressed data
    ├── 1.meta                # Metadata cache
    ├── 2.bin
    └── 2.meta
```

## Development Setup

### Prerequisites

- Go 1.21+
- Node.js 18+
- SQLite3
- golangci-lint (for linting)

### Clone & Build

```bash
git clone https://github.com/erkexzcx/flightlesssomething.git
cd flightlesssomething

# Build web UI
cd web && npm install && npm run build && cd ..

# Build server
go build -o server ./cmd/server
```

### Run Development Server

```bash
./server \
  -bind="0.0.0.0:5000" \
  -data-dir="./data" \
  -session-secret="dev-secret" \
  -discord-client-id="your-id" \
  -discord-client-secret="your-secret" \
  -discord-redirect-url="http://localhost:5000/auth/login/callback" \
  -admin-username="admin" \
  -admin-password="admin"
```

Or with environment variables:
```bash
export FS_BIND="0.0.0.0:5000"
export FS_DATA_DIR="./data"
export FS_SESSION_SECRET="dev-secret"
# ... other variables

./server
```

## Code Style

### Go

Follow standard Go conventions:
- `gofmt` formatting
- Effective Go guidelines
- golangci-lint rules

Run linter:
```bash
golangci-lint run --timeout=5m
```

### JavaScript/Vue

ESLint with Vue plugin:
```bash
cd web
npm run lint
npm run lint:fix
```

## Testing

See [Testing Guide](testing.md) for complete details.

Quick reference:
```bash
# Backend tests
go test ./...

# Backend integration
./backend_test.sh

# Frontend unit tests
cd web && npm run test:unit

# Frontend E2E tests
cd web && npm test

# Linting
golangci-lint run
cd web && npm run lint
```

## Making Changes

### Adding New API Endpoint

1. Define handler in `internal/app/`
2. Add route in `internal/app/server.go`
3. Add tests in `internal/app/*_test.go`
4. Update API documentation in `docs/api.md`

Example:
```go
// Handler
func HandleNewEndpoint(c *gin.Context) {
    // Implementation
}

// Route
api := r.Group("/api")
api.GET("/new-endpoint", HandleNewEndpoint)

// Test
func TestHandleNewEndpoint(t *testing.T) {
    // Test implementation
}
```

### Adding Frontend Feature

1. Create component in `web/src/components/` or view in `web/src/views/`
2. Add route if needed in `web/src/router/index.js`
3. Update API client if new endpoint in `web/src/api/client.js`
4. Add tests in `web/tests/`

### Database Changes

1. Update models in `internal/app/models.go`
2. GORM auto-migrates on startup
3. Schema version tracking for major migrations
4. Old schema detection and automatic migration

## Security

### Authentication

- **Discord OAuth**: Regular users
- **Admin**: Username/password from environment
- **Sessions**: Cookie-based with secret key

### Rate Limiting

- Benchmark uploads: 5 per 10 minutes per user
- Admin login: 3 failed attempts locks for 10 minutes

### File Upload

- Validation of file formats
- Size limits enforced
- Sanitization of filenames

### Best Practices

1. Never commit secrets
2. Validate all inputs
3. Use parameterized queries (GORM handles this)
4. Escape user content in UI
5. Rate limit sensitive endpoints

## Debugging

### Backend

Use delve debugger:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server -- [flags...]
```

Or add log statements:
```go
log.Printf("Debug: %+v", variable)
```

### Frontend

Vue DevTools browser extension.

Console logging:
```javascript
console.log('Debug:', variable);
```

## Performance

### Database

- Indexes on frequently queried fields
- Compressed binary storage for large data
- Metadata caching in `.meta` files

### Frontend

- Code splitting with Vite
- Lazy loading routes
- Optimized bundle size (~41KB gzipped)

## Docker Development

Build and run locally:
```bash
docker build -t flightlesssomething-dev .
docker run -p 5000:5000 \
  -e FS_BIND=0.0.0.0:5000 \
  -e FS_DATA_DIR=/data \
  -v $(pwd)/data:/data \
  flightlesssomething-dev
```

Or use docker-compose:
```bash
cp .env.example .env
# Edit .env
docker-compose up
```

## CI/CD

GitHub Actions workflows:

**`.github/workflows/test.yml`**:
- Runs on every push/PR
- Linting, unit tests, integration tests
- All must pass for merge

**`.github/workflows/release.yml`**:
- Runs on release/tag
- Builds Docker image
- Pushes to GHCR

**`.github/workflows/deploy.yml`**:
- Manual trigger
- Deploys to dev or prod

## Contributing

1. Fork repository
2. Create feature branch
3. Make changes with tests
4. Run linters and tests
5. Submit PR

### Commit Messages

Use conventional commits:
- `feat: add new feature`
- `fix: fix bug`
- `docs: update documentation`
- `test: add tests`
- `refactor: refactor code`

### Pull Requests

- Descriptive title and description
- Reference issues if applicable
- All tests must pass
- Code reviewed before merge

## Release Process

1. Update version if needed
2. Create tag: `git tag v1.0.0`
3. Push tag: `git push origin v1.0.0`
4. Create GitHub release
5. Release workflow builds and pushes image
6. Manually deploy via Actions

## Useful Commands

```bash
# Build everything
make build

# Clean build artifacts
make clean

# Run server
./server [flags...]

# Run with custom config
./server -config config.txt

# Check dependency updates
go list -u -m all
cd web && npm outdated

# Format code
go fmt ./...
cd web && npm run lint:fix

# Run specific test
go test -v ./internal/app -run TestName

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://github.com/gin-gonic/gin)
- [GORM](https://gorm.io/)
- [Vue.js](https://vuejs.org/)
- [Vite](https://vitejs.dev/)
