package main

import (
	"fmt"
	"log"

	"github.com/erkexzcx/flightlesssomething/internal/app"
)

var (
	version = "dev"
)

func main() {
	config, err := app.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if config.Version {
		fmt.Printf("flightlesssomething version: %s\n", version)
		return
	}

	if err := app.Start(config, version); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
