package app

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebFS is the embedded web filesystem - will be set via webfs_embed.go during build
var WebFS embed.FS

// setupSPA configures serving the Vue.js SPA from embedded files
func setupSPA(r *gin.Engine) {
	// Try to get the dist subdirectory from the embedded filesystem
	distFS, err := fs.Sub(WebFS, "web/dist")
	if err != nil {
		log.Printf("Warning: Web UI not embedded, falling back to API-only mode: %v", err)
		log.Printf("Note: If building with Docker, ensure the Dockerfile builds the web UI first")
		return
	}

	log.Printf("Web UI loaded successfully")

	// Serve static assets (JS, CSS, images, etc.)
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, http.FS(distFS))
	})

	// Serve favicon
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("favicon.ico", http.FS(distFS))
	})
	r.GET("/favicon.svg", func(c *gin.Context) {
		c.FileFromFS("favicon.svg", http.FS(distFS))
	})

	// Serve index.html for the root path
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			log.Printf("Error reading index.html: %v", err)
			c.String(http.StatusInternalServerError, "Error loading page")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// For all other routes (SPA routes), serve index.html
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Don't serve index.html for API routes
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/health") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Serve index.html for all other routes (Vue Router will handle them)
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			log.Printf("Error reading index.html: %v", err)
			c.String(http.StatusInternalServerError, "Error loading page")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
