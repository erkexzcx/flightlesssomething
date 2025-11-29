.PHONY: build build-web build-server clean run test

# Build everything
build: build-web build-server

# Build the web UI
build-web:
	@echo "Building web UI..."
	cd web && npm install && npm run build

# Build the Go server with embedded web UI
build-server: build-web
	@echo "Building Go server..."
	@# Copy web/dist to internal/app/web/dist for embedding
	@rm -rf internal/app/web
	@mkdir -p internal/app/web
	@cp -r web/dist internal/app/web/
	@# Create webfs_embed.go for embedding
	@echo 'package app' > internal/app/webfs_embed.go
	@echo '' >> internal/app/webfs_embed.go
	@echo 'import "embed"' >> internal/app/webfs_embed.go
	@echo '' >> internal/app/webfs_embed.go
	@echo '//go:embed all:web/dist' >> internal/app/webfs_embed.go
	@echo 'var webFSEmbed embed.FS' >> internal/app/webfs_embed.go
	@echo '' >> internal/app/webfs_embed.go
	@echo 'func init() {' >> internal/app/webfs_embed.go
	@echo '	WebFS = webFSEmbed' >> internal/app/webfs_embed.go
	@echo '}' >> internal/app/webfs_embed.go
	@# Build the server with version from git
	@VERSION=$$(git describe --tags --always 2>/dev/null || echo "dev"); \
	go build -ldflags="-X main.version=$$VERSION" -o server ./cmd/server
	@# Clean up copied files
	@rm -rf internal/app/web internal/app/webfs_embed.go

# Clean build artifacts
clean:
	rm -rf web/node_modules web/dist server internal/app/web internal/app/webfs_embed.go

# Run the server (development mode without building web UI)
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Run backend integration tests
test-integration:
	./backend_test.sh
