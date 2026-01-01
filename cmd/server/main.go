package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/erkexzcx/flightlesssomething/internal/app"
)

var (
	version = "dev"
)

func main() {
	// Configure garbage collector for lower memory usage
	// GOGC controls the aggressiveness of the garbage collector
	// Lower values trigger GC more frequently, reducing memory usage at slight CPU cost
	// Default is 100, we use 50 for better memory efficiency
	gogc := 50
	if gogcEnv := os.Getenv("GOGC"); gogcEnv != "" {
		if val, err := strconv.Atoi(gogcEnv); err == nil && val > 0 {
			gogc = val
		}
	}
	debug.SetGCPercent(gogc)
	
	// Set memory limit if configured (Go 1.19+)
	// This provides a soft memory limit - GC will try to stay below this
	if memLimit := os.Getenv("GOMEMLIMIT"); memLimit != "" {
		debug.SetMemoryLimit(parseMemoryLimit(memLimit))
	}

	config, err := app.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if config.Version {
		fmt.Printf("flightlesssomething version: %s\n", version)
		fmt.Printf("GC percent: %d\n", gogc)
		return
	}

	log.Printf("Starting server with GOGC=%d", gogc)
	if err := app.Start(config, version); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// parseMemoryLimit parses memory limit string (e.g., "512MiB", "2GiB")
// Returns -1 if invalid or empty, which disables the limit
func parseMemoryLimit(s string) int64 {
	var limit int64 = -1
	
	// Must have at least 4 characters (e.g., "1MiB")
	if len(s) < 4 {
		return limit
	}
	
	// Check for valid suffix
	suffix := s[len(s)-3:]
	if suffix == "MiB" || suffix == "GiB" || suffix == "KiB" {
		// Parse the number part
		numStr := s[:len(s)-3]
		if val, err := strconv.ParseInt(numStr, 10, 64); err == nil && val > 0 {
			switch suffix {
			case "KiB":
				limit = val * 1024
			case "MiB":
				limit = val * 1024 * 1024
			case "GiB":
				limit = val * 1024 * 1024 * 1024
			}
		}
	}
	
	return limit
}
