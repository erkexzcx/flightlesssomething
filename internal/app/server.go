// Package app provides the core application logic for the benchmark management system.
// It includes server initialization, HTTP handlers, database management, and authentication.
package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// Start initializes and starts the server
func Start(config *Config, version string) error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0o750); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize benchmarks directory
	if err := InitBenchmarksDir(config.DataDir); err != nil {
		return fmt.Errorf("failed to initialize benchmarks directory: %w", err)
	}

	// Initialize database
	db, err := InitDB(config.DataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Ensure system admin account exists and is up to date
	if err := EnsureSystemAdmin(db, config.AdminUsername, config.AdminPassword); err != nil {
		return fmt.Errorf("failed to ensure system admin: %w", err)
	}

	// Initialize Discord OAuth
	InitDiscordOAuth(config.DiscordClientID, config.DiscordClientSecret, config.DiscordRedirectURL)

	// Initialize rate limiters
	InitRateLimiters()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Disable trailing slash redirect to prevent issues with SPA
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Setup sessions
	store := cookie.NewStore([]byte(config.SessionSecret))
	secureCookie := strings.HasPrefix(config.DiscordRedirectURL, "https://")
	store.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("flightlesssomething_session", store))

	// Public routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": version})
	})

	// Auth routes
	r.GET("/auth/login", HandleLogin)
	r.GET("/auth/login/callback", HandleLoginCallback(db))
	r.POST("/auth/admin/login", HandleAdminLogin(config, db))
	r.POST("/auth/logout", HandleLogout(secureCookie))
	r.GET("/api/auth/me", HandleGetCurrentUser)

	// Public benchmark routes
	r.GET("/api/benchmarks", HandleListBenchmarks(db))
	r.GET("/api/benchmarks/:id", HandleGetBenchmark(db))
	r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
	r.GET("/api/benchmarks/:id/runs/:runIndex", HandleGetBenchmarkRun(db))
	r.GET("/api/benchmarks/:id/download", HandleDownloadBenchmarkData(db))

	// Debug calc endpoint (public, for verifying backend calculations)
	r.POST("/api/debugcalc", HandleDebugCalc())

	// Protected benchmark routes
	authorized := r.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.POST("/benchmarks", HandleCreateBenchmark(db))
	authorized.PUT("/benchmarks/:id", HandleUpdateBenchmark(db))
	authorized.DELETE("/benchmarks/:id", HandleDeleteBenchmark(db))
	authorized.DELETE("/benchmarks/:id/runs/:run_index", HandleDeleteBenchmarkRun(db))
	authorized.POST("/benchmarks/:id/runs", HandleAddBenchmarkRuns(db))

	// API token routes
	authorized.GET("/tokens", HandleListAPITokens(db))
	authorized.POST("/tokens", HandleCreateAPIToken(db))
	authorized.DELETE("/tokens/:id", HandleDeleteAPIToken(db))

	// Admin routes
	admin := r.Group("/api/admin")
	admin.Use(RequireAuth(), RequireAdmin())
	admin.GET("/users", HandleListUsers(db))
	admin.DELETE("/users/:id", HandleDeleteUser(db))
	admin.DELETE("/users/:id/benchmarks", HandleDeleteUserBenchmarks(db))
	admin.PUT("/users/:id/ban", HandleBanUser(db))
	admin.PUT("/users/:id/admin", HandleToggleUserAdmin(db))
	admin.GET("/logs", HandleListAuditLogs(db))

	// MCP (Model Context Protocol) server
	r.POST("/mcp", HandleMCP(db, version))
	r.GET("/mcp", HandleMCPGet)
	r.DELETE("/mcp", HandleMCPDelete)

	// Serve Vue.js SPA
	setupSPA(r)

	log.Printf("Starting server on %s (version: %s)\n", config.Bind, version)
	log.Printf("Data directory: %s\n", config.DataDir)
	log.Printf("Benchmarks directory: %s\n", filepath.Join(config.DataDir, "benchmarks"))

	return r.Run(config.Bind)
}
