# Deployment Guide

## Environments

### Dev Environment
- **Trigger**: Auto-deploy on push to any branch except main
- **Build**: Docker image built from source
- **Purpose**: Testing and development
- **Data**: Copies from production for testing
- **Workflow**: `.github/workflows/deploy.yml`

### Prod Environment
- **Trigger**: Automatic deployment when a release is published
- **Build**: Uses GHCR release images (no source build needed)
- **Purpose**: Production deployment
- **Data**: Protected production data
- **Workflow**: `.github/workflows/release.yml`

## Production Deployment

### Automatic Deployment on Release

When you publish a release, the following happens automatically:

1. **Build**: Docker image is built (amd64 only) and pushed to GHCR
2. **Deploy**: Image is automatically pulled and deployed to production

To deploy to production:

1. Create and push a tag:
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. Create release on GitHub:
   - Go to repository "Releases" page
   - Click "Draft a new release"
   - Choose the tag
   - Add release notes
   - Click "Publish release"

3. The release workflow automatically:
   - Builds Docker image (amd64 only)
   - Pushes to `ghcr.io/erkexzcx/flightlesssomething`
   - Tags with version and `latest`
   - **Deploys to production server** (new!)

That's it! No manual deployment needed.

## Docker Images

Release images available at:
```
ghcr.io/erkexzcx/flightlesssomething:latest
ghcr.io/erkexzcx/flightlesssomething:v1.0.0
```

Pull and run:
```bash
docker pull ghcr.io/erkexzcx/flightlesssomething:latest

docker run -d \
  -p 5000:5000 \
  -v /path/to/data:/data \
  -e FS_BIND=0.0.0.0:5000 \
  -e FS_DATA_DIR=/data \
  -e FS_SESSION_SECRET=your-secret \
  -e FS_DISCORD_CLIENT_ID=your-id \
  -e FS_DISCORD_CLIENT_SECRET=your-secret \
  -e FS_DISCORD_REDIRECT_URL=https://yourdomain.com/auth/login/callback \
  -e FS_ADMIN_USERNAME=admin \
  -e FS_ADMIN_PASSWORD=your-password \
  ghcr.io/erkexzcx/flightlesssomething:latest
```

**Note**: Images are amd64 only. ARM users must build from source.

## GitHub Secrets

Configure these in repository settings:

- `DOMAIN_SUFFIX` - Domain suffix (e.g., `.example.com`)
- `SSH_HOST` - Deployment server hostname
- `SSH_PORT` - SSH port (typically 22)
- `SSH_USER` - SSH username
- `SSH_PRIVATE_KEY` - SSH private key
- `SESSION_SECRET` - Session encryption key
- `DISCORD_CLIENT_ID` - Discord OAuth client ID
- `DISCORD_CLIENT_SECRET` - Discord OAuth client secret
- `ADMIN_USERNAME` - Admin username
- `ADMIN_PASSWORD` - Admin password

## Server Setup

Deployment workflow automatically:
1. Builds/pulls Docker image
2. Transfers to server via SSH
3. Deploys to `~/fs-prod` or `~/fs-dev`
4. For dev: copies data from prod if exists

Containers named:
- `flightlesssomething-prod`
- `flightlesssomething-dev`

Both run on default bridge network, port 5000 internal.

### Reverse Proxy (Caddy)

Example Caddy configuration:
```
fs.example.com {
    encode zstd gzip
    header -Server
    reverse_proxy flightlesssomething-prod:5000 
}

fs-dev.example.com {
    encode zstd gzip
    header -Server
    reverse_proxy flightlesssomething-dev:5000 
}
```

Ensure Caddy is on same Docker network (bridge).

## Local Docker Build

Build from source:
```bash
docker build -t flightlesssomething .
```

Run with docker-compose:
```bash
cp .env.example .env
# Edit .env with your configuration
docker-compose up -d
```

## Environment Variables

All CLI flags can be set as environment variables with `FS_` prefix:

```bash
export FS_BIND="0.0.0.0:5000"
export FS_DATA_DIR="/data"
export FS_SESSION_SECRET="your-secret"
export FS_DISCORD_CLIENT_ID="your-id"
export FS_DISCORD_CLIENT_SECRET="your-secret"
export FS_DISCORD_REDIRECT_URL="http://localhost:5000/auth/login/callback"
export FS_ADMIN_USERNAME="admin"
export FS_ADMIN_PASSWORD="admin"

./server
```

## Discord OAuth Setup

1. Create Discord application at https://discord.com/developers/applications
2. Add OAuth2 redirect URL: `http://your-server:5000/auth/login/callback`
3. Set environment variables:
   - `FS_DISCORD_CLIENT_ID`
   - `FS_DISCORD_CLIENT_SECRET`
   - `FS_DISCORD_REDIRECT_URL`

## Monitoring

Check container logs:
```bash
# Production
docker logs -f flightlesssomething-prod

# Development
docker logs -f flightlesssomething-dev
```

Check container status:
```bash
docker ps | grep flightlesssomething
```

## Troubleshooting

### Container won't start
- Check environment variables are set
- Verify data directory has correct permissions
- Check logs for error messages

### OAuth not working
- Verify Discord redirect URL matches configuration
- Check Discord application credentials
- Ensure callback URL is accessible

### Database errors
- Check data directory permissions
- Ensure sufficient disk space
- Verify SQLite is not corrupted
