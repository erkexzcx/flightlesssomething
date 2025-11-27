# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies (including Node.js for web UI)
RUN apk add --no-cache git gcc musl-dev nodejs npm

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

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .

# Create data directory
RUN mkdir -p /data

# Expose port
EXPOSE 5000

# Run the application
ENTRYPOINT ["/app/server"]
