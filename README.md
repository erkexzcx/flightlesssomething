# FlightlessSomething

> ⚠️ **Note**: This project is purely vibe coded - built with passion, intuition, and good vibes.

A web application for storing and managing gaming benchmark data with Discord OAuth authentication and a modern Vue.js interface.

## Live Environments

- **Main**: [flightlesssomething.ambrosia.one](https://flightlesssomething.ambrosia.one/) - The main instance for general use
- **Development**: [flightlesssomething-dev.ambrosia.one](https://flightlesssomething-dev.ambrosia.one/) - Development and testing environment for experimenting with features, scripts, and automations. **Note: this instance is occasionally wiped, so any data uploaded there will be lost.**

## MCP Server

FlightlessSomething includes a built-in **MCP (Model Context Protocol) server**, allowing you to interact with your benchmark data directly from AI assistants like GitHub Copilot, Claude, and other MCP-compatible tools.

Head over to **[flightlesssomething.ambrosia.one/api-tokens](https://flightlesssomething.ambrosia.one/api-tokens)** — you'll find setup instructions and configuration snippets right on the page.

> **Note:** You need to **log in** first to access the API Tokens page. Once there, you can use the MCP server in two modes:
> - **Anonymous mode** — browse and query public benchmark data without an API token
> - **Authenticated mode** — use an API token to access your own data and perform actions on your behalf

## Deployment

### Docker Compose (recommended)

1. Edit `docker-compose.yml` and fill in your Discord OAuth credentials, session secret, and admin credentials.
2. Start the service:
   ```bash
   docker compose up -d
   ```

The application will be available at `http://localhost:5000`.

### Docker

```bash
docker run -d \
  -p 5000:5000 \
  -v ./data:/data \
  -e FS_SESSION_SECRET=your-secret-key \
  -e FS_DISCORD_CLIENT_ID=your-discord-client-id \
  -e FS_DISCORD_CLIENT_SECRET=your-discord-client-secret \
  -e FS_DISCORD_REDIRECT_URL=http://localhost:5000/auth/login/callback \
  -e FS_ADMIN_USERNAME=admin \
  -e FS_ADMIN_PASSWORD=your-secure-password \
  --restart unless-stopped \
  ghcr.io/erkexzcx/flightlesssomething:latest
```
