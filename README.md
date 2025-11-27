# FlightlessSomething

> ⚠️ **Note**: This project is purely vibe coded - built with passion, intuition, and good vibes.

A web application for storing and managing gaming benchmark data with Discord OAuth authentication and a modern Vue.js interface.

## Tech Stack

### Backend
- **Go 1.24** - Programming language
- **Gin** - HTTP web framework
- **GORM** - Database ORM
- **SQLite** - Database
- **Discord OAuth2** - User authentication
- **zstd** - Data compression

### Frontend
- **Vue.js 3** - JavaScript framework
- **Vite** - Build tool
- **Vue Router** - Client-side routing
- **Pinia** - State management
- **Bootstrap 5** - CSS framework
- **Highcharts** - Data visualization
- **dayjs** - Date formatting

## Quick Start

### Prerequisites
- Go 1.21 or higher
- Node.js 18 or higher

### Build & Run

```bash
# Build web UI
cd web && npm install && npm run build && cd ..

# Build server
go build -o server ./cmd/server

# Run server
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

### Using Docker

```bash
docker-compose up -d
```

## Configuration

All settings can be configured via CLI flags or environment variables (with `FS_` prefix):

- `bind` - Server address (default: `0.0.0.0:5000`)
- `data-dir` - Data directory path
- `session-secret` - Session encryption key
- `discord-client-id` - Discord OAuth client ID
- `discord-client-secret` - Discord OAuth client secret
- `discord-redirect-url` - OAuth callback URL
- `admin-username` - Admin account username
- `admin-password` - Admin account password

## Documentation

All detailed documentation is available in the [`docs/`](docs/) directory:

- [Benchmark Guide](docs/benchmarks.md) - How to capture and upload benchmark data
- [API Documentation](docs/api.md) - REST API endpoints and examples
- [Deployment Guide](docs/deployment.md) - Production deployment and CI/CD
- [Testing Guide](docs/testing.md) - Running tests and contributing
- [Migration Guide](docs/migration.md) - Migrating from old version
- [Web UI Guide](docs/webui.md) - Frontend development
- [Development Guide](docs/development.md) - Contributing and architecture

## License

MIT License - see individual files for details.
