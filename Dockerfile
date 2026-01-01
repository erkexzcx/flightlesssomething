# Build stage
FROM golang:1.25-trixie AS builder

# Install build dependencies (including Node.js for web UI)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the web UI
RUN cd web && npm install && npm run build

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
FROM gcr.io/distroless/base-debian13:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .

# Expose port
EXPOSE 5000

# Run the application
ENTRYPOINT ["/app/server"]
