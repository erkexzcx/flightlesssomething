# Web UI build stage
FROM node:25-alpine AS web-builder

WORKDIR /build

# Copy all web UI files
COPY web ./

# Install dependencies and build
RUN npm ci --prefer-offline --no-audit
RUN npm run build

# Go build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built web UI from web-builder stage
COPY --from=web-builder /build/dist ./web/dist

# Prepare web files for embedding
RUN mkdir -p internal/app/web && \
    cp -r web/dist internal/app/web/ && \
    { \
        echo 'package app'; \
        echo ''; \
        echo 'import "embed"'; \
        echo ''; \
        echo '//go:embed all:web/dist'; \
        echo 'var webFSEmbed embed.FS'; \
        echo ''; \
        echo 'func init() {'; \
        printf '\tWebFS = webFSEmbed\n'; \
        echo '}'; \
    } > internal/app/webfs_embed.go

# Build the application with version from git
RUN VERSION=$(git describe --tags --always 2>/dev/null || echo "dev") && \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s -X main.version=${VERSION}" -trimpath -tags netgo -o server ./cmd/server

# Runtime stage
FROM alpine:3.23

# Install CA certificates for HTTPS/TLS connections (Discord OAuth, etc.)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .

# Expose port
EXPOSE 5000

# Run the application
ENTRYPOINT ["/app/server"]
