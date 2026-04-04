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
	// Configure garbage collector for lower memory usage.
	// GOGC controls the aggressiveness of the garbage collector.
	// Lower values trigger GC more frequently, reducing memory usage at slight CPU cost.
	// Default is 100; we cap the accepted range at 1–500 to prevent misconfigured values
	// from causing excessive memory retention or thrashing.
	// Note: GOMEMLIMIT is handled automatically by the Go runtime (Go 1.19+) before
	// main() is entered; no manual debug.SetMemoryLimit call is required.
	gogc := 50
	if gogcEnv := os.Getenv("GOGC"); gogcEnv != "" {
		if val, err := strconv.Atoi(gogcEnv); err == nil && val >= 1 && val <= 500 {
			gogc = val
		}
	}
	debug.SetGCPercent(gogc)

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
