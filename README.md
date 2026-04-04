# FlightlessSomething

A self-hosted web application for storing, managing, and visualizing gaming benchmark data. Upload your MangoHud or MSI Afterburner captures, compare runs side-by-side with interactive charts, and share results with others — all from a single, lightweight binary.

## Live Instances

| Environment | URL | Notes |
|---|---|---|
| Production | [flightlesssomething.ambrosia.one](https://flightlesssomething.ambrosia.one/) | Public instance for general use |
| Development | [flightlesssomething-dev.ambrosia.one](https://flightlesssomething-dev.ambrosia.one/) | Experimental; data is occasionally wiped |

## Features

- **Multi-run benchmarks** — group multiple captures into a single benchmark entry for side-by-side comparison
- **Interactive charts** — FPS, frametime, CPU/GPU load, temperatures, clocks, VRAM, RAM, and more (13 metrics total)
- **Pre-calculated statistics** — min, max, average, median, P1/P5/P10/P25/P75/P90/P95/P97/P99, standard deviation, variance, and density histograms
- **Dual format support** — MangoHud CSV (Linux) and MSI Afterburner HML (Windows)
- **Discord OAuth** — sign in with your Discord account; no passwords to manage
- **API tokens** — Bearer token authentication for scripted or programmatic access (up to 10 tokens per user)
- **MCP server** — built-in Model Context Protocol server so AI assistants can query benchmark data directly
- **Admin panel** — user management, bans, admin promotion, and audit logging
- **Single binary** — the Go backend and Vue.js frontend are compiled into one self-contained executable

## Supported Benchmark Formats

| Tool | Platform | File format |
|---|---|---|
| [MangoHud](https://github.com/flightlessmango/MangoHud) | Linux | `.csv` |
| [MSI Afterburner](https://www.msi.com/Landing/afterburner/graphics-cards) + RTSS | Windows | `.hml` |

Captured metrics: FPS, Frametime, CPU Load, GPU Load, CPU Temp, CPU Power, GPU Temp, GPU Core Clock, GPU Mem Clock, GPU VRAM Used, GPU Power, RAM Used, Swap Used.

Both formats are detected by file content, not extension, so any file extension is accepted at upload.

For capture instructions see [docs/benchmarks.md](docs/benchmarks.md).

## MCP Server

FlightlessSomething includes a built-in **MCP (Model Context Protocol) server**, letting AI assistants such as GitHub Copilot and Claude interact with benchmark data directly.

- **Anonymous mode** — browse and query public benchmarks without an API token
- **Authenticated mode** — use a Bearer token to access your own data and perform actions on your behalf

Setup instructions and ready-to-paste configuration snippets are available at **[flightlesssomething.ambrosia.one/api-tokens](https://flightlesssomething.ambrosia.one/api-tokens)** after logging in.

Full tool reference: [docs/api.md](docs/api.md).

## Deployment

### Discord Application Setup

Before deploying, you need a Discord OAuth application to handle authentication:

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and create a new application.
2. Navigate to **OAuth2** and copy the **Client ID** and **Client Secret**.
3. Under **Redirects**, add your callback URL: `https://your-domain.example.com/auth/login/callback` (or `http://localhost:5000/auth/login/callback` for local testing).

These values map to `FS_DISCORD_CLIENT_ID`, `FS_DISCORD_CLIENT_SECRET`, and `FS_DISCORD_REDIRECT_URL`.

### Docker Compose (recommended)

1. Clone the repository and copy `.env.example` to `.env`, then fill in your values:

   ```bash
   cp .env.example .env
   ```

2. Start the service:

   ```bash
   docker compose up -d
   ```

The application will be available at `http://localhost:5000`.

> **Note:** The `docker-compose.yml` builds the image locally from source. Two directories are created on the host: `./data` (database and benchmark files) and `./logs` (audit log — see `./logs/audit.json`).

### Docker (prebuilt image)

```bash
docker run -d \
  -p 5000:5000 \
  -v ./data:/data \
  -v ./logs:/logs \
  -e FS_SESSION_SECRET=your-secret-key \
  -e FS_DISCORD_CLIENT_ID=your-discord-client-id \
  -e FS_DISCORD_CLIENT_SECRET=your-discord-client-secret \
  -e FS_DISCORD_REDIRECT_URL=http://localhost:5000/auth/login/callback \
  -e FS_ADMIN_USERNAME=admin \
  -e FS_ADMIN_PASSWORD=your-secure-password \
  --restart unless-stopped \
  ghcr.io/erkexzcx/flightlesssomething:latest
```

> **Note:** Mount `./logs:/logs` to persist audit logs. Audit logs are written to `/logs/audit.json` (one level above the data directory).

### Building from Source

Requires **Go 1.26+** and **Node.js 25+**.

```bash
make build        # builds frontend + Go binary
./server -bind 0.0.0.0:5000 -data-dir ./data ...
```

## Configuration

All settings are available as CLI flags or environment variables (prefix `FS_`):

| Environment variable | Flag | Default | Required | Description |
|---|---|---|---|---|
| `FS_BIND` | `-bind` | `0.0.0.0:5000` | No | Listen address |
| `FS_DATA_DIR` | `-data-dir` | `/data` | No | Data storage directory |
| `FS_SESSION_SECRET` | `-session-secret` | — | **Yes** | Cookie encryption key |
| `FS_DISCORD_CLIENT_ID` | `-discord-client-id` | — | **Yes** | Discord OAuth app ID |
| `FS_DISCORD_CLIENT_SECRET` | `-discord-client-secret` | — | **Yes** | Discord OAuth secret |
| `FS_DISCORD_REDIRECT_URL` | `-discord-redirect-url` | — | **Yes** | OAuth callback URL |
| `FS_ADMIN_USERNAME` | `-admin-username` | — | **Yes** | Admin account username |
| `FS_ADMIN_PASSWORD` | `-admin-password` | — | **Yes** | Admin account password |
| — | `-version` | — | No | Print version and exit |

Optional memory tuning (set as environment variables):

| Variable | Example | Description |
|---|---|---|
| `GOGC` | `50` | GC target percentage. The application already defaults this to **50** (more aggressive than Go's built-in default of 100) for lower peak memory usage. Override only if needed. |
| `GOMEMLIMIT` | `512MiB` | Soft memory limit for the Go runtime. Supported units: `KiB`, `MiB`, `GiB`, `TiB`. |

## Documentation

- [docs/api.md](docs/api.md) — REST API and MCP tool reference
- [docs/architecture.md](docs/architecture.md) — system design, storage format, and data flow
- [docs/benchmarks.md](docs/benchmarks.md) — benchmark capture guide for MangoHud and MSI Afterburner
